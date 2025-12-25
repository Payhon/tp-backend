package service

import (
	"context"
	"time"

	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	"project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

// OrgService 组织管理服务
type OrgService struct{}

// CreateOrg 创建组织并维护闭包表
func (*OrgService) CreateOrg(ctx context.Context, req *model.OrgCreateReq, claims *utils.UserClaims) (*model.Org, error) {
	now := time.Now()
	orgID := uuid.New()

	org := &model.Org{
		ID:            orgID,
		Name:          req.Name,
		OrgType:       req.OrgType,
		ParentID:      req.ParentID,
		TenantID:      claims.TenantID,
		ContactPerson: req.ContactPerson,
		Phone:         req.Phone,
		Email:         req.Email,
		Province:      req.Province,
		City:          req.City,
		District:      req.District,
		Address:       req.Address,
		Status:        StringPtr(model.OrgStatusNormal),
		CreatedAt:     &now,
		UpdatedAt:     &now,
		Remark:        req.Remark,
	}

	// 开启事务
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 创建组织记录
		if err := tx.Create(org).Error; err != nil {
			return err
		}

		// 2. 维护闭包表
		// 2.1 插入自环记录（自己是自己的祖先，depth=0）
		selfClosure := &model.OrgClosure{
			TenantID:     claims.TenantID,
			AncestorID:   orgID,
			DescendantID: orgID,
			Depth:        0,
		}
		if err := tx.Create(selfClosure).Error; err != nil {
			return err
		}

		// 2.2 如果有父节点，复制父节点的所有祖先关系，并将 descendant_id 设为当前节点
		if req.ParentID != nil && *req.ParentID != "" {
			// 查询父节点的所有祖先（包含父节点自身）
			var parentClosures []model.OrgClosure
			if err := tx.Where("tenant_id = ? AND descendant_id = ?", claims.TenantID, *req.ParentID).
				Find(&parentClosures).Error; err != nil {
				return err
			}

			// 为每个祖先创建到新节点的闭包记录
			for _, pc := range parentClosures {
				newClosure := &model.OrgClosure{
					TenantID:     claims.TenantID,
					AncestorID:   pc.AncestorID,
					DescendantID: orgID,
					Depth:        pc.Depth + 1,
				}
				if err := tx.Create(newClosure).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return org, nil
}

// UpdateOrg 更新组织信息（不支持修改 parent_id，需要单独的移动操作）
func (*OrgService) UpdateOrg(ctx context.Context, orgID string, req *model.OrgUpdateReq, claims *utils.UserClaims) error {
	now := time.Now()
	updates := map[string]interface{}{
		"updated_at": now,
	}

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ContactPerson != nil {
		updates["contact_person"] = *req.ContactPerson
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Province != nil {
		updates["province"] = *req.Province
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.District != nil {
		updates["district"] = *req.District
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}

	result := global.DB.Model(&model.Org{}).
		Where("id = ? AND tenant_id = ?", orgID, claims.TenantID).
		Updates(updates)

	if result.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": result.Error.Error(),
		})
	}
	if result.RowsAffected == 0 {
		return errcode.New(errcode.CodeNotFound)
	}

	return nil
}

// DeleteOrg 删除组织（需要先确保没有子组织和关联设备）
func (*OrgService) DeleteOrg(ctx context.Context, orgID string, claims *utils.UserClaims) error {
	// 检查是否有子组织
	var childCount int64
	if err := global.DB.Model(&model.Org{}).
		Where("parent_id = ? AND tenant_id = ?", orgID, claims.TenantID).
		Count(&childCount).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	if childCount > 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "存在子组织，无法删除",
		})
	}

	// 检查是否有关联设备
	var deviceCount int64
	if err := global.DB.Model(&model.DeviceBattery{}).
		Where("owner_org_id = ?", orgID).
		Count(&deviceCount).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	if deviceCount > 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "存在关联设备，无法删除",
		})
	}

	// 开启事务删除
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 删除闭包表记录
		if err := tx.Where("tenant_id = ? AND (ancestor_id = ? OR descendant_id = ?)",
			claims.TenantID, orgID, orgID).
			Delete(&model.OrgClosure{}).Error; err != nil {
			return err
		}

		// 删除组织记录
		if err := tx.Where("id = ? AND tenant_id = ?", orgID, claims.TenantID).
			Delete(&model.Org{}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return nil
}

// GetOrgByID 获取单个组织详情
func (*OrgService) GetOrgByID(ctx context.Context, orgID string, claims *utils.UserClaims) (*model.Org, error) {
	org, err := query.Org.WithContext(ctx).
		Where(query.Org.ID.Eq(orgID), query.Org.TenantID.Eq(claims.TenantID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(errcode.CodeNotFound)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	return org, nil
}

// GetOrgList 获取组织列表（支持按类型筛选）
func (*OrgService) GetOrgList(ctx context.Context, req *model.OrgListReq, claims *utils.UserClaims) (*model.OrgListResp, error) {
	q := query.Org
	queryBuilder := q.WithContext(ctx).Where(q.TenantID.Eq(claims.TenantID))

	// 按类型筛选
	if req.OrgType != nil && *req.OrgType != "" {
		queryBuilder = queryBuilder.Where(q.OrgType.Eq(*req.OrgType))
	}

	// 按名称模糊搜索
	if req.Name != nil && *req.Name != "" {
		queryBuilder = queryBuilder.Where(q.Name.Like("%" + *req.Name + "%"))
	}

	// 按状态筛选
	if req.Status != nil && *req.Status != "" {
		queryBuilder = queryBuilder.Where(q.Status.Eq(*req.Status))
	}

	// 按父组织筛选
	if req.ParentID != nil && *req.ParentID != "" {
		queryBuilder = queryBuilder.Where(q.ParentID.Eq(*req.ParentID))
	}

	// 统计总数
	total, err := queryBuilder.Count()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	orgs, err := queryBuilder.
		Order(q.CreatedAt.Desc()).
		Offset(offset).
		Limit(req.PageSize).
		Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return &model.OrgListResp{
		List:     orgs,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetOrgTree 获取组织树结构
func (*OrgService) GetOrgTree(ctx context.Context, claims *utils.UserClaims, orgType *string) ([]*model.OrgTreeNode, error) {
	q := query.Org
	queryBuilder := q.WithContext(ctx).Where(q.TenantID.Eq(claims.TenantID))

	// 按类型筛选
	if orgType != nil && *orgType != "" {
		queryBuilder = queryBuilder.Where(q.OrgType.Eq(*orgType))
	}

	orgs, err := queryBuilder.Order(q.CreatedAt).Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 构建树结构
	return buildOrgTree(orgs), nil
}

// buildOrgTree 将平铺的组织列表构建为树结构
func buildOrgTree(orgs []*model.Org) []*model.OrgTreeNode {
	nodeMap := make(map[string]*model.OrgTreeNode)
	var roots []*model.OrgTreeNode

	// 第一遍：创建所有节点
	for _, org := range orgs {
		nodeMap[org.ID] = &model.OrgTreeNode{
			Org:      org,
			Children: []*model.OrgTreeNode{},
		}
	}

	// 第二遍：建立父子关系
	for _, org := range orgs {
		node := nodeMap[org.ID]
		if org.ParentID == nil || *org.ParentID == "" {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[*org.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// 父节点不在结果集中（可能被类型筛选过滤），作为根节点
			roots = append(roots, node)
		}
	}

	return roots
}

// GetDescendantOrgIDs 获取指定组织的所有后代组织ID（包含自身）
func (*OrgService) GetDescendantOrgIDs(ctx context.Context, tenantID, orgID string) ([]string, error) {
	var descendants []string
	err := global.DB.WithContext(ctx).
		Table("org_closure").
		Select("descendant_id").
		Where("tenant_id = ? AND ancestor_id = ?", tenantID, orgID).
		Pluck("descendant_id", &descendants).Error
	if err != nil {
		return nil, err
	}
	if len(descendants) == 0 {
		descendants = []string{orgID}
	}
	return descendants, nil
}

// InitTenantRootOrg 为租户初始化根组织（BMS_FACTORY 类型）
func (*OrgService) InitTenantRootOrg(ctx context.Context, tenantID, tenantName string) (*model.Org, error) {
	now := time.Now()
	orgID := uuid.New()

	org := &model.Org{
		ID:        orgID,
		Name:      tenantName + " (BMS厂家)",
		OrgType:   model.OrgTypeBMSFactory,
		ParentID:  nil,
		TenantID:  tenantID,
		Status:    StringPtr(model.OrgStatusNormal),
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(org).Error; err != nil {
			return err
		}

		// 自环闭包记录
		selfClosure := &model.OrgClosure{
			TenantID:     tenantID,
			AncestorID:   orgID,
			DescendantID: orgID,
			Depth:        0,
		}
		return tx.Create(selfClosure).Error
	})

	if err != nil {
		return nil, err
	}
	return org, nil
}
