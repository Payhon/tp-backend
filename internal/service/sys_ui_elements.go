package service

import (
	"context"
	"strings"
	"time"

	dal "project/internal/dal"
	model "project/internal/model"
	"project/pkg/errcode"
	utils "project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
)

type UiElements struct{}

func (*UiElements) CreateUiElements(CreateUiElementsReq *model.CreateUiElementsReq) error {

	var UiElements = model.SysUIElement{}

	UiElements.ID = uuid.New()
	UiElements.ParentID = CreateUiElementsReq.ParentID
	UiElements.ElementCode = CreateUiElementsReq.ElementCode
	UiElements.ElementType = int16(CreateUiElementsReq.ElementType)
	aa := int16(CreateUiElementsReq.Orders)
	UiElements.Order_ = &aa
	UiElements.Param1 = CreateUiElementsReq.Param1
	UiElements.Param2 = CreateUiElementsReq.Param2
	UiElements.Param3 = CreateUiElementsReq.Param3
	UiElements.CreatedAt = time.Now().UTC()
	UiElements.Authority = CreateUiElementsReq.Authority
	UiElements.Description = CreateUiElementsReq.Description
	UiElements.Remark = CreateUiElementsReq.Remark
	UiElements.Multilingual = CreateUiElementsReq.Multilingual
	UiElements.RoutePath = CreateUiElementsReq.RoutePath
	err := dal.CreateUiElements(&UiElements)

	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "create_ui_elements",
			"error":     err.Error(),
		})
	}

	return err
}

func (*UiElements) UpdateUiElements(UpdateUiElementsReq *model.UpdateUiElementsReq) error {
	var UiElements = model.SysUIElement{}
	UiElements.ID = UpdateUiElementsReq.Id
	UiElements.ParentID = *UpdateUiElementsReq.ParentID
	UiElements.ElementCode = *UpdateUiElementsReq.ElementCode
	UiElements.ElementType = *UpdateUiElementsReq.ElementType
	UiElements.Order_ = UpdateUiElementsReq.Orders
	UiElements.Param1 = UpdateUiElementsReq.Param1
	UiElements.Param2 = UpdateUiElementsReq.Param2
	UiElements.Param3 = UpdateUiElementsReq.Param3
	UiElements.Authority = *UpdateUiElementsReq.Authority
	UiElements.Description = UpdateUiElementsReq.Description
	UiElements.Multilingual = UpdateUiElementsReq.Multilingual
	UiElements.RoutePath = UpdateUiElementsReq.RoutePath
	UiElements.Remark = UpdateUiElementsReq.Remark

	err := dal.UpdateUiElements(&UiElements)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "update_ui_elements",
			"error":     err.Error(),
		})
	}
	return err
}

func (*UiElements) DeleteUiElements(id string) error {
	err := dal.DeleteUiElements(id)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "delete_ui_elements",
			"error":     err.Error(),
		})
	}
	return err
}

func (*UiElements) ServeUiElementsListByPage(Params *model.ServeUiElementsListByPageReq) (map[string]interface{}, error) {

	total, list, err := dal.ServeUiElementsListByPage(Params)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_ui_elements",
			"error":     err.Error(),
		})
	}
	UiElementsListRsp := make(map[string]interface{})
	UiElementsListRsp["total"] = total
	UiElementsListRsp["list"] = list

	return UiElementsListRsp, err
}

func filterMenuTreeByAllowedCodes(nodes []*model.UiElementsListRsp, allowed map[string]struct{}) []*model.UiElementsListRsp {
	if len(nodes) == 0 {
		return nodes
	}
	out := make([]*model.UiElementsListRsp, 0, len(nodes))
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if len(n.Children) > 0 {
			n.Children = filterMenuTreeByAllowedCodes(n.Children, allowed)
		}

		_, ok := allowed[n.ElementCode]
		if ok || len(n.Children) > 0 {
			out = append(out, n)
		}
	}
	return out
}

func (*UiElements) ServeUiElementsListByAuthority(ctx context.Context, u *utils.UserClaims) (map[string]interface{}, error) {
	total, list, err := dal.ServeUiElementsListByAuthority(u)
	if err != nil {
		logrus.Error("[ServeUiElementsListByAuthority] query failed:", err)
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_ui_elements",
			"user_id":   u.ID,
			"error":     err.Error(),
		})
	}

	// 机构类型菜单权限：仅对归属 PACK/经销商/门店 的业务账号生效
	// - 菜单来源仍然是用户原有菜单（casbin/authority）
		// - 在返回前按 org_type_permissions.ui_codes 做一次裁剪，保证不同机构类型看到的菜单一致
	if u.Authority != "SYS_ADMIN" && u.Authority != "TENANT_ADMIN" && strings.TrimSpace(u.TenantID) != "" && strings.TrimSpace(u.ID) != "" {
		orgType, ok, err := GroupApp.OrgTypePermission.GetUserOrgType(ctx, u.TenantID, u.ID)
		if err == nil && ok {
			switch orgType {
			case model.OrgTypePACKFactory, model.OrgTypeDealer, model.OrgTypeStore:
					allowed, exists, err := GroupApp.OrgTypePermission.GetAllowedUICodes(ctx, u.TenantID, orgType)
					if err == nil && exists {
						if typed, ok := list.([]*model.UiElementsListRsp); ok {
							allowedSet := make(map[string]struct{}, len(allowed))
							for _, code := range allowed {
								code = strings.TrimSpace(code)
								if code == "" {
									continue
								}
								allowedSet[code] = struct{}{}
							}
							list = filterMenuTreeByAllowedCodes(typed, allowedSet)
						}
					}
				}
			}
		}

	return map[string]interface{}{
		"total": total,
		"list":  list,
	}, nil
}

// 获取租户下权限配置表单树
func (*UiElements) GetTenantUiElementsList() (map[string]interface{}, error) {

	list, err := dal.GetTenantUiElementsList()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_ui_elements",
			"error":     err.Error(),
		})
	}
	UiElementsListRsp := make(map[string]interface{})
	UiElementsListRsp["list"] = list

	return UiElementsListRsp, err
}
