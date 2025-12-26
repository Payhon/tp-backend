package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	"project/pkg/global"
	"project/pkg/utils"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrgTypePermission 机构类型权限配置
type OrgTypePermission struct{}

type orgTypePermissionPO struct {
	TenantID               string         `gorm:"column:tenant_id;primaryKey"`
	OrgType                string         `gorm:"column:org_type;primaryKey"`
	UICodes                datatypes.JSON `gorm:"column:ui_codes"`
	DeviceParamPermissions *string        `gorm:"column:device_param_permissions"`
	CreatedAt              time.Time      `gorm:"column:created_at"`
	UpdatedAt              time.Time      `gorm:"column:updated_at"`
}

func (orgTypePermissionPO) TableName() string { return "org_type_permissions" }

func orgTypeRoleName(tenantID, orgType string) string {
	return fmt.Sprintf("TENANT_%s_ORGTYPE_%s", tenantID, orgType)
}

func normalizeUICodes(codes []string) []string {
	if len(codes) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
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

func (s *OrgTypePermission) resolveTenantID(claims *utils.UserClaims, tenantID string) (string, error) {
	if claims.Authority == "SYS_ADMIN" {
		if strings.TrimSpace(tenantID) == "" {
			return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"tenant_id": "tenant_id is required for SYS_ADMIN",
			})
		}
		return strings.TrimSpace(tenantID), nil
	}
	if claims.Authority != "TENANT_ADMIN" {
		return "", errcode.New(errcode.CodeNoPermission)
	}
	return claims.TenantID, nil
}

func (s *OrgTypePermission) List(ctx context.Context, claims *utils.UserClaims, tenantID string) ([]model.OrgTypePermissionResp, error) {
	resolvedTenantID, err := s.resolveTenantID(claims, tenantID)
	if err != nil {
		return nil, err
	}

	var rows []orgTypePermissionPO
	if err := global.DB.WithContext(ctx).
		Where("tenant_id = ?", resolvedTenantID).
		Find(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_org_type_permissions",
			"error":     err.Error(),
		})
	}

	out := make([]model.OrgTypePermissionResp, 0, len(rows))
	for _, r := range rows {
		var uiCodes []string
		_ = json.Unmarshal(r.UICodes, &uiCodes)
		resp := model.OrgTypePermissionResp{
			OrgType:                r.OrgType,
			UICodes:                uiCodes,
			DeviceParamPermissions: "",
		}
		if r.DeviceParamPermissions != nil {
			resp.DeviceParamPermissions = *r.DeviceParamPermissions
		}
		out = append(out, resp)
	}
	return out, nil
}

func (s *OrgTypePermission) Upsert(ctx context.Context, claims *utils.UserClaims, tenantID, orgType string, req *model.OrgTypePermissionUpsertReq) (*model.OrgTypePermissionResp, error) {
	resolvedTenantID, err := s.resolveTenantID(claims, tenantID)
	if err != nil {
		return nil, err
	}

	switch orgType {
	case model.OrgTypePACKFactory, model.OrgTypeDealer, model.OrgTypeStore:
	default:
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"org_type": orgType,
			"error":    "org_type must be one of PACK_FACTORY/DEALER/STORE",
		})
	}

	uiCodes := normalizeUICodes(req.UICodes)
	uiCodesJSON, _ := json.Marshal(uiCodes)

	now := time.Now().UTC()
	devicePerm := strings.TrimSpace(req.DeviceParamPermissions)
	po := &orgTypePermissionPO{
		TenantID:               resolvedTenantID,
		OrgType:                orgType,
		UICodes:                datatypes.JSON(uiCodesJSON),
		DeviceParamPermissions: &devicePerm,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := global.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "org_type"}},
			DoUpdates: clause.AssignmentColumns([]string{"ui_codes", "device_param_permissions", "updated_at"}),
		}).
		Create(po).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "upsert_org_type_permissions",
			"error":     err.Error(),
		})
	}

	// 同步 Casbin：机构类型角色 -> 菜单权限（sys_ui_elements.id）
	roleName := orgTypeRoleName(resolvedTenantID, orgType)
	if _, err := global.CasbinEnforcer.RemoveFilteredPolicy(0, roleName); err != nil {
		return nil, errcode.WithData(errcode.CodeSystemError, map[string]interface{}{
			"operation": "casbin_remove_role_policies",
			"role":      roleName,
			"error":     err.Error(),
		})
	}
	if len(uiCodes) > 0 {
		// element_code -> id
		var elementIDs []string
		if err := global.DB.WithContext(ctx).
			Table("sys_ui_elements").
			Select("id").
			Where("element_code IN ?", uiCodes).
			Scan(&elementIDs).Error; err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_ui_element_ids",
				"error":     err.Error(),
			})
		}

		var rules [][]string
		for _, id := range elementIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			rules = append(rules, []string{roleName, id, "allow"})
		}
		if len(rules) > 0 {
			if _, err := global.CasbinEnforcer.AddNamedPolicies("p", rules); err != nil {
			return nil, errcode.WithData(errcode.CodeSystemError, map[string]interface{}{
				"operation": "casbin_add_role_policies",
				"role":      roleName,
				"error":     err.Error(),
			})
			}
		}
	}

	// 给当前租户下该机构类型的所有业务账号补齐该角色
	var userIDs []string
	if err := global.DB.WithContext(ctx).
		Table("users AS u").
		Select("u.id").
		Joins("JOIN orgs AS o ON o.id = u.org_id").
		Where("u.tenant_id = ? AND u.user_kind = ? AND o.org_type = ?", resolvedTenantID, model.UserKindOrgUser, orgType).
		Scan(&userIDs).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_org_users",
			"error":     err.Error(),
		})
	}
	if len(userIDs) > 0 {
		var gRules [][]string
		for _, uid := range userIDs {
			gRules = append(gRules, []string{uid, roleName})
		}
		_, _ = global.CasbinEnforcer.AddNamedGroupingPolicies("g", gRules)
	}

	return &model.OrgTypePermissionResp{
		OrgType:                orgType,
		UICodes:                uiCodes,
		DeviceParamPermissions: devicePerm,
	}, nil
}

func (s *OrgTypePermission) GetAllowedUICodes(ctx context.Context, tenantID, orgType string) ([]string, bool, error) {
	var row orgTypePermissionPO
	if err := global.DB.WithContext(ctx).
		Where("tenant_id = ? AND org_type = ?", tenantID, orgType).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	var uiCodes []string
	_ = json.Unmarshal(row.UICodes, &uiCodes)
	return uiCodes, true, nil
}

func (s *OrgTypePermission) GetUserOrgType(ctx context.Context, tenantID, userID string) (string, bool, error) {
	var orgID string
	if err := global.DB.WithContext(ctx).
		Table("users").
		Select("org_id").
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Scan(&orgID).Error; err != nil {
		return "", false, err
	}
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return "", false, nil
	}

	var orgType string
	if err := global.DB.WithContext(ctx).
		Table("orgs").
		Select("org_type").
		Where("id = ? AND tenant_id = ?", orgID, tenantID).
		Scan(&orgType).Error; err != nil {
		return "", false, err
	}
	orgType = strings.TrimSpace(orgType)
	if orgType == "" {
		return "", false, nil
	}
	return orgType, true, nil
}

func (s *OrgTypePermission) GetDeviceParamOptions() ([]model.DeviceParamTreeNode, error) {
	// 相对 backend 工作目录（与 messages.yaml 同级）
	b, err := os.ReadFile("configs/device_param_permissions.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []model.DeviceParamTreeNode{}, nil
		}
		return nil, err
	}

	var opts []model.DeviceParamTreeNode
	if err := json.Unmarshal(b, &opts); err != nil {
		return nil, err
	}
	return opts, nil
}
