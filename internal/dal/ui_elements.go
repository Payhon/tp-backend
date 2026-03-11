package dal

import (
	"context"
	"sort"
	"strings"

	model "project/internal/model"
	query "project/internal/query"
	global "project/pkg/global"
	utils "project/pkg/utils"

	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gen"
)

func CreateUiElements(uielements *model.SysUIElement) error {
	return query.SysUIElement.Create(uielements)
}

func UpdateUiElements(uielements *model.SysUIElement) error {
	p := query.SysUIElement
	_, err := query.SysUIElement.Where(p.ID.Eq(uielements.ID)).Updates(uielements)
	if err != nil {
		logrus.Error(err)
	}
	return err
}

func DeleteUiElements(id string) error {
	_, err := query.SysUIElement.Where(query.SysUIElement.ID.Eq(id)).Delete()
	if err != nil {
		logrus.Error(err)
	}
	return err
}

func ServeUiElementsListByPage(uielements *model.ServeUiElementsListByPageReq) (int64, interface{}, error) {
	q := query.SysUIElement
	var count int64
	queryBuilder := q.WithContext(context.Background())
	queryBuilder = queryBuilder.Where(q.ParentID.Eq("0"))
	count, err := queryBuilder.Count()
	if err != nil {
		logrus.Error(err)
		return count, nil, err
	}
	if uielements.Page != 0 && uielements.PageSize != 0 {
		queryBuilder = queryBuilder.Limit(uielements.PageSize)
		queryBuilder = queryBuilder.Offset((uielements.Page - 1) * uielements.PageSize)
	}

	uielementsList, err := queryBuilder.Select().Order(q.Order_).Find()
	if err != nil {
		logrus.Error(err)
		return count, uielementsList, err
	}

	var uielementsListrsp []*model.UiElementsListRsp
	for i := range uielementsList {
		uielementsListrsp = append(uielementsListrsp, uielementsList[i].ToRsp())
		queryChildren(uielementsListrsp[i])
	}
	return count, uielementsListrsp, err
}

func ServeUiElementsListByAuthority(u *utils.UserClaims) (int64, interface{}, error) {
	// 系统管理员和租户管理员菜单树
	if u.Authority == "SYS_ADMIN" || u.Authority == "TENANT_ADMIN" {
		q := query.SysUIElement
		var count int64
		queryBuilder := q.WithContext(context.Background())
		queryBuilder = queryBuilder.Where(gen.Cond(datatypes.JSONQuery("authority").HasKey(u.Authority))...)
		uielementsList, err := queryBuilder.Where(q.ParentID.Eq("0"), q.ElementType.In(1, 2, 3)).Order(q.Order_).Find()
		if err != nil {
			logrus.Error(err)
			return count, uielementsList, err
		}
		count, err = queryBuilder.Count()

		var uielementsListrsp []*model.UiElementsListRsp
		for i := range uielementsList {
			uielementsListrsp = append(uielementsListrsp, uielementsList[i].ToRsp())
			queryChildrenByAuthority(uielementsListrsp[i], u.Authority)
		}
		return count, uielementsListrsp, err
	} else {
		// 租户用户菜单树
		// 从casbin_rule表查询当前用户拥有的根权限
		var uielementsList []*model.SysUIElement
		err := global.DB.Raw(`select * from
		(
		select distinct (crp.v1) from casbin_rule crp 
		inner join 
		(
		select cr.v1 from casbin_rule cr  where cr.ptype ='g' and cr.v0 = ? 
		) crr
		 on crr.v1 = crp.v0 where crp.ptype ='p'
		) t
		left join sys_ui_elements tf on t.v1 = tf.id 
		where tf.parent_id ='0' and tf.element_type in (1,2,3)
		order by tf.orders desc`, u.ID).Scan(&uielementsList)
		if err.Error != nil {
			return 0, nil, err.Error
		}
		var data []*model.UiElementsListRsp
		for i := range uielementsList {
			data = append(data, uielementsList[i].ToRsp())
			queryChildrenByUserID(data[i], u.ID)
		}
		return 0, data, nil
	}
}

// ServeUiElementsListByCodes 按 element_code 构建菜单树（自动补齐祖先节点）
func ServeUiElementsListByCodes(codes []string) (int64, []*model.UiElementsListRsp, error) {
	normalized := normalizeMenuCodes(codes)
	if len(normalized) == 0 {
		return 0, []*model.UiElementsListRsp{}, nil
	}

	rows, err := query.SysUIElement.
		Where(query.SysUIElement.ElementType.In(1, 2, 3)).
		Order(query.SysUIElement.Order_).
		Find()
	if err != nil {
		return 0, nil, err
	}

	idIndex := make(map[string]*model.SysUIElement, len(rows))
	codeIndex := make(map[string]*model.SysUIElement, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		idIndex[row.ID] = row
		if code := strings.TrimSpace(row.ElementCode); code != "" {
			codeIndex[code] = row
		}
	}

	included := make(map[string]struct{}, len(normalized))
	for _, code := range normalized {
		node, ok := codeIndex[code]
		if !ok || node == nil {
			continue
		}
		addAncestors(node.ID, idIndex, included)
	}

	if len(included) == 0 {
		return 0, []*model.UiElementsListRsp{}, nil
	}

	childrenByParent := make(map[string][]*model.SysUIElement)
	for id := range included {
		node := idIndex[id]
		if node == nil {
			continue
		}
		parentID := strings.TrimSpace(node.ParentID)
		if parentID == "" {
			parentID = "0"
		}
		childrenByParent[parentID] = append(childrenByParent[parentID], node)
	}

	for parentID := range childrenByParent {
		sort.Slice(childrenByParent[parentID], func(i, j int) bool {
			return menuOrderValue(childrenByParent[parentID][i]) < menuOrderValue(childrenByParent[parentID][j])
		})
	}

	roots := childrenByParent["0"]
	result := make([]*model.UiElementsListRsp, 0, len(roots))
	for _, root := range roots {
		result = append(result, buildMenuNode(root, childrenByParent))
	}
	return int64(len(result)), result, nil
}

// 获取租户下权限配置表单树
func GetTenantUiElementsList() (interface{}, error) {
	q := query.SysUIElement
	queryBuilder := q.WithContext(context.Background())
	queryBuilder = queryBuilder.Where(gen.Cond(datatypes.JSONQuery("authority").HasKey("TENANT_ADMIN"))...)
	uielementsList, err := queryBuilder.Where(q.ParentID.Eq("0"), q.ElementType.In(1, 2, 3, 4)).Order(q.Order_).Find()
	if err != nil {
		logrus.Error(err)
		return uielementsList, err
	}

	var uielementsListrsp []*model.UiElementsListRsp1
	for i := range uielementsList {
		uielementsListrsp = append(uielementsListrsp, uielementsList[i].ToRsp1())
		queryChildrenByAuthority1(uielementsListrsp[i], "TENANT_ADMIN")
	}
	return uielementsListrsp, err
}

func queryChildren(parent *model.UiElementsListRsp) {
	var children []*model.SysUIElement
	children, err := query.SysUIElement.Where(query.SysUIElement.ParentID.Eq(parent.ID)).Order(query.SysUIElement.Order_).Find()
	if err != nil {
		logrus.Error(err)
	}
	if children == nil {
		return
	}
	for i := range children {
		parent.Children = append(parent.Children, children[i].ToRsp())
		queryChildren(parent.Children[i])
	}
}

func queryChildrenByAuthority(parent *model.UiElementsListRsp, authority string) {
	var children []*model.SysUIElement
	children, err := query.SysUIElement.Where(
		query.SysUIElement.ParentID.Eq(parent.ID),
		query.SysUIElement.ElementType.In(1, 2, 3),
		query.SysUIElement.Where(gen.Cond(datatypes.JSONQuery("authority").HasKey(authority))...),
	).Order(query.SysUIElement.Order_).Find()
	if err != nil {
		logrus.Error(err)
	}
	if children == nil {
		return
	}
	for i := range children {
		parent.Children = append(parent.Children, children[i].ToRsp())
		queryChildrenByAuthority(parent.Children[i], authority)
	}
}
func queryChildrenByAuthority1(parent *model.UiElementsListRsp1, authority string) {
	var children []*model.SysUIElement
	children, err := query.SysUIElement.Where(
		query.SysUIElement.ParentID.Eq(parent.ID),
		query.SysUIElement.ElementType.In(1, 2, 3, 4),
		query.SysUIElement.Where(gen.Cond(datatypes.JSONQuery("authority").HasKey(authority))...),
	).Order(query.SysUIElement.Order_).Find()
	if err != nil {
		logrus.Error(err)
	}
	if children == nil {
		return
	}
	for i := range children {
		parent.Children = append(parent.Children, children[i].ToRsp1())
		queryChildrenByAuthority1(parent.Children[i], authority)
	}
}

func queryChildrenByUserID(parent *model.UiElementsListRsp, userID string) {
	var children []*model.SysUIElement
	err := global.DB.Raw(`select * from
		(
		select distinct (crp.v1) from casbin_rule crp 
		inner join 
		(
		select cr.v1 from casbin_rule cr  where cr.ptype ='g' and cr.v0 = ? 
		) crr
		 on crr.v1 = crp.v0 where crp.ptype ='p'
		) t
		left join sys_ui_elements tf on t.v1 = tf.id 
		where tf.parent_id =? and tf.element_type in (1,2,3)
		order by tf.orders desc`, userID, parent.ID).Scan(&children)
	if err.Error != nil {
		logrus.Error(err)
	}
	if children == nil {
		return
	}
	for i := range children {
		parent.Children = append(parent.Children, children[i].ToRsp())
		queryChildrenByUserID(parent.Children[i], userID)
	}
}

func normalizeMenuCodes(codes []string) []string {
	if len(codes) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}

func addAncestors(id string, idIndex map[string]*model.SysUIElement, included map[string]struct{}) {
	visited := map[string]struct{}{}
	current := strings.TrimSpace(id)
	for current != "" && current != "0" {
		if _, ok := visited[current]; ok {
			return
		}
		visited[current] = struct{}{}
		included[current] = struct{}{}

		node := idIndex[current]
		if node == nil {
			return
		}
		parentID := strings.TrimSpace(node.ParentID)
		if parentID == "" || parentID == current {
			return
		}
		current = parentID
	}
}

func menuOrderValue(node *model.SysUIElement) int {
	if node == nil || node.Order_ == nil {
		return 0
	}
	return int(*node.Order_)
}

func buildMenuNode(node *model.SysUIElement, childrenByParent map[string][]*model.SysUIElement) *model.UiElementsListRsp {
	if node == nil {
		return nil
	}
	rsp := node.ToRsp()
	children := childrenByParent[node.ID]
	if len(children) == 0 {
		return rsp
	}
	rsp.Children = make([]*model.UiElementsListRsp, 0, len(children))
	for _, child := range children {
		rsp.Children = append(rsp.Children, buildMenuNode(child, childrenByParent))
	}
	return rsp
}
