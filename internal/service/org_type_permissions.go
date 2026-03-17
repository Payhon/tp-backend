package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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

type uiElementLite struct {
	ID          string `gorm:"column:id"`
	ParentID    string `gorm:"column:parent_id"`
	ElementCode string `gorm:"column:element_code"`
}

func expandUICodesWithAncestors(ctx context.Context, codes []string) ([]string, error) {
	normalized := normalizeUICodes(codes)
	if len(normalized) == 0 {
		return []string{}, nil
	}

	var rows []uiElementLite
	if err := global.DB.WithContext(ctx).
		Table("sys_ui_elements").
		Select("id, parent_id, element_code").
		Where("element_type IN (1,2,3,4)").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	idIndex := make(map[string]uiElementLite, len(rows))
	codeIndex := make(map[string]uiElementLite, len(rows))
	for _, row := range rows {
		idIndex[row.ID] = row
		code := strings.TrimSpace(row.ElementCode)
		if code != "" {
			codeIndex[code] = row
		}
	}

	includedIDs := make(map[string]struct{}, len(normalized))
	for _, code := range normalized {
		row, ok := codeIndex[code]
		if !ok {
			continue
		}
		addAncestorIDs(row.ID, idIndex, includedIDs)
	}

	expanded := make([]string, 0, len(includedIDs))
	seenCode := make(map[string]struct{}, len(includedIDs))

	// 保留用户配置顺序
	for _, code := range normalized {
		if _, ok := seenCode[code]; ok {
			continue
		}
		row, ok := codeIndex[code]
		if !ok {
			continue
		}
		if _, exists := includedIDs[row.ID]; !exists {
			continue
		}
		seenCode[code] = struct{}{}
		expanded = append(expanded, code)
	}

	// 追加祖先编码（按编码排序，确保输出稳定）
	extra := make([]string, 0, len(includedIDs))
	for _, row := range rows {
		if _, ok := includedIDs[row.ID]; !ok {
			continue
		}
		code := strings.TrimSpace(row.ElementCode)
		if code == "" {
			continue
		}
		if _, ok := seenCode[code]; ok {
			continue
		}
		seenCode[code] = struct{}{}
		extra = append(extra, code)
	}
	sort.Strings(extra)
	expanded = append(expanded, extra...)

	return expanded, nil
}

func addAncestorIDs(id string, idIndex map[string]uiElementLite, included map[string]struct{}) {
	current := strings.TrimSpace(id)
	visited := map[string]struct{}{}
	for current != "" && current != "0" {
		if _, ok := visited[current]; ok {
			return
		}
		visited[current] = struct{}{}
		included[current] = struct{}{}

		row, ok := idIndex[current]
		if !ok {
			return
		}
		parentID := strings.TrimSpace(row.ParentID)
		if parentID == "" || parentID == current {
			return
		}
		current = parentID
	}
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
	case model.OrgTypePACKFactory, model.OrgTypeDealer, model.OrgTypeStore, model.OrgTypeAppUser:
	default:
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"org_type": orgType,
			"error":    "org_type must be one of PACK_FACTORY/DEALER/STORE/APP_USER",
		})
	}

	uiCodes, err := expandUICodesWithAncestors(ctx, req.UICodes)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "expand_ui_codes_with_ancestors",
			"error":     err.Error(),
		})
	}
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

	// 给当前租户下该机构类型对应账号补齐角色（ORG_USER 或 APP_USER=END_USER）
	userIDs, err := getOrgTypeTargetUserIDs(ctx, resolvedTenantID, orgType)
	if err != nil {
		return nil, err
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

func getOrgTypeTargetUserIDs(ctx context.Context, tenantID, orgType string) ([]string, error) {
	db := global.DB.WithContext(ctx).Table("users AS u").Select("u.id").Where("u.tenant_id = ?", tenantID)

	switch orgType {
	case model.OrgTypeAppUser:
		db = db.Where("u.user_kind = ?", model.UserKindEndUser)
	default:
		db = db.Joins("JOIN orgs AS o ON o.id = u.org_id").
			Where("u.user_kind = ? AND o.org_type = ?", model.UserKindOrgUser, orgType)
	}

	var userIDs []string
	if err := db.Scan(&userIDs).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_org_type_users",
			"org_type":  orgType,
			"error":     err.Error(),
		})
	}
	return userIDs, nil
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

func splitDeviceParamPermissions(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func (s *OrgTypePermission) GetAllowedDeviceParamPermissions(ctx context.Context, tenantID, orgType string) ([]string, bool, error) {
	var row orgTypePermissionPO
	if err := global.DB.WithContext(ctx).
		Where("tenant_id = ? AND org_type = ?", tenantID, orgType).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	raw := ""
	if row.DeviceParamPermissions != nil {
		raw = *row.DeviceParamPermissions
	}
	return splitDeviceParamPermissions(raw), true, nil
}

func (s *OrgTypePermission) GetUserKind(ctx context.Context, tenantID, userID string) (string, error) {
	var kind sql.NullString
	if err := global.DB.WithContext(ctx).
		Table("users").
		Select("user_kind").
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Scan(&kind).Error; err != nil {
		return "", err
	}
	text := strings.TrimSpace(kind.String)
	if text == "" {
		text = model.UserKindEndUser
	}
	return text, nil
}

func (s *OrgTypePermission) GetUserOrgType(ctx context.Context, tenantID, userID string) (string, bool, error) {
	var orgID sql.NullString
	if err := global.DB.WithContext(ctx).
		Table("users").
		Select("org_id").
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Scan(&orgID).Error; err != nil {
		return "", false, err
	}
	orgIDText := strings.TrimSpace(orgID.String)
	if orgIDText == "" {
		return "", false, nil
	}

	var orgType sql.NullString
	if err := global.DB.WithContext(ctx).
		Table("orgs").
		Select("org_type").
		Where("id = ? AND tenant_id = ?", orgIDText, tenantID).
		Scan(&orgType).Error; err != nil {
		return "", false, err
	}
	orgTypeText := strings.TrimSpace(orgType.String)
	if orgTypeText == "" {
		return "", false, nil
	}
	return orgTypeText, true, nil
}

func (s *OrgTypePermission) GetCurrentDeviceParamPermissions(ctx context.Context, claims *utils.UserClaims) (*model.DeviceParamPermissionResp, error) {
	resp := &model.DeviceParamPermissionResp{
		OrgType:                "",
		OrgTypes:               []string{},
		AllowAll:               true,
		DeviceParamPermissions: []string{},
	}
	if claims == nil {
		return resp, nil
	}
	if claims.Authority == "SYS_ADMIN" || claims.Authority == "TENANT_ADMIN" {
		return resp, nil
	}

	tenantID := strings.TrimSpace(claims.TenantID)
	userID := strings.TrimSpace(claims.ID)
	if tenantID == "" || userID == "" {
		return resp, nil
	}

	userKind, err := s.GetUserKind(ctx, tenantID, userID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_user_kind",
			"user_id":   userID,
			"error":     err.Error(),
		})
	}

	orgTypes := []string{model.OrgTypeAppUser}
	orgType := ""
	if userKind == model.UserKindOrgUser {
		var ok bool
		orgType, ok, err = s.GetUserOrgType(ctx, tenantID, userID)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_user_org_type",
				"user_id":   userID,
				"error":     err.Error(),
			})
		}
		if ok {
			orgType = strings.TrimSpace(orgType)
			if orgType != "" {
				orgTypes = append(orgTypes, orgType)
			}
		}
	}

	resp.OrgType = orgType
	resp.OrgTypes = normalizeOrgTypes(orgTypes)

	merged := make([]string, 0)
	mergedSet := make(map[string]struct{})
	hasConfig := false
	for _, ot := range orgTypes {
		ot = strings.TrimSpace(ot)
		if ot == "" {
			continue
		}
		allowed, exists, err := s.GetAllowedDeviceParamPermissions(ctx, tenantID, ot)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_device_param_permissions",
				"org_type":  ot,
				"error":     err.Error(),
			})
		}
		if !exists {
			continue
		}
		hasConfig = true
		for _, key := range allowed {
			if _, ok := mergedSet[key]; ok {
				continue
			}
			mergedSet[key] = struct{}{}
			merged = append(merged, key)
		}
	}

	if hasConfig {
		resp.AllowAll = false
		resp.DeviceParamPermissions = merged
	}
	return resp, nil
}

func (s *OrgTypePermission) GetCurrentUIPermissions(ctx context.Context, claims *utils.UserClaims) (*model.UIPermissionResp, error) {
	resp := &model.UIPermissionResp{
		OrgType:  "",
		OrgTypes: []string{},
		AllowAll: true,
		UICodes:  []string{},
	}
	if claims == nil {
		return resp, nil
	}
	if claims.Authority == "SYS_ADMIN" || claims.Authority == "TENANT_ADMIN" {
		return resp, nil
	}

	tenantID := strings.TrimSpace(claims.TenantID)
	userID := strings.TrimSpace(claims.ID)
	if tenantID == "" || userID == "" {
		return resp, nil
	}

	userKind, err := s.GetUserKind(ctx, tenantID, userID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_user_kind",
			"user_id":   userID,
			"error":     err.Error(),
		})
	}

	orgTypes := []string{model.OrgTypeAppUser}
	orgType := ""
	if userKind == model.UserKindOrgUser {
		var ok bool
		orgType, ok, err = s.GetUserOrgType(ctx, tenantID, userID)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_user_org_type",
				"user_id":   userID,
				"error":     err.Error(),
			})
		}
		if ok {
			orgType = strings.TrimSpace(orgType)
			if orgType != "" {
				orgTypes = append(orgTypes, orgType)
			}
		}
	}

	resp.OrgType = orgType
	resp.OrgTypes = normalizeOrgTypes(orgTypes)

	merged := make([]string, 0)
	mergedSet := make(map[string]struct{})
	hasConfig := false
	for _, ot := range orgTypes {
		ot = strings.TrimSpace(ot)
		if ot == "" {
			continue
		}
		allowed, exists, err := s.GetAllowedUICodes(ctx, tenantID, ot)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_ui_permissions",
				"org_type":  ot,
				"error":     err.Error(),
			})
		}
		if !exists {
			continue
		}
		hasConfig = true
		for _, code := range allowed {
			code = strings.TrimSpace(code)
			if code == "" {
				continue
			}
			if _, ok := mergedSet[code]; ok {
				continue
			}
			mergedSet[code] = struct{}{}
			merged = append(merged, code)
		}
	}

	if hasConfig {
		resp.AllowAll = false
		resp.UICodes = merged
	}

	return resp, nil
}

func normalizeOrgTypes(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func (s *OrgTypePermission) GetDeviceParamOptions() ([]model.DeviceParamTreeNode, error) {
	return buildDeviceParamPermissionTree(), nil
}
