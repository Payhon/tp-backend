package service

import (
	"context"
	"strings"
	"time"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"

	"project/internal/dal"
	"project/internal/model"
	"project/pkg/errcode"
	"project/pkg/utils"
)

type OpenAPIKey struct{}

func resolveOpenAPIKeyRemark(req *model.CreateOpenAPIKeyReq) string {
	if req.Remark != nil && strings.TrimSpace(*req.Remark) != "" {
		return strings.TrimSpace(*req.Remark)
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		return strings.TrimSpace(*req.Name)
	}
	return "MES 默认密钥"
}

func parseOpenAPIExpiredAt(v *string) (*time.Time, error) {
	if v == nil {
		return nil, nil
	}

	raw := strings.TrimSpace(*v)
	if raw == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, raw, time.Local)
		if err == nil {
			utc := t.UTC()
			return &utc, nil
		}
	}
	return nil, errcode.NewWithMessage(errcode.CodeParamError, "invalid expired_at format")
}

func resolveKeyTenantID(req *model.CreateOpenAPIKeyReq, claims *utils.UserClaims) (string, error) {
	if claims.Authority != "SYS_ADMIN" && claims.Authority != "TENANT_ADMIN" {
		return "", errcode.WithVars(errcode.CodeNoPermission, map[string]interface{}{
			"required_role": "SYS_ADMIN or TENANT_ADMIN",
			"current_role":  claims.Authority,
		})
	}

	if claims.Authority == "TENANT_ADMIN" {
		return claims.TenantID, nil
	}

	if req.TenantID == nil || strings.TrimSpace(*req.TenantID) == "" {
		return "", errcode.NewWithMessage(errcode.CodeParamError, "tenant_id is required for SYS_ADMIN")
	}
	return strings.TrimSpace(*req.TenantID), nil
}

func buildOpenAPIKeyResp(entity *dal.OpenAPIKeyEntity) *model.OpenAPIKeyListRsp {
	remark := entity.Remark
	if remark == nil || strings.TrimSpace(*remark) == "" {
		remark = &entity.Name
	}

	name := entity.Name
	if strings.TrimSpace(name) == "" && remark != nil {
		name = strings.TrimSpace(*remark)
	}

	return &model.OpenAPIKeyListRsp{
		ID:         entity.ID,
		TenantID:   entity.TenantID,
		AppID:      entity.APIKey,
		APIKey:     entity.APIKey,
		SecretKey:  entity.SecretKey,
		Remark:     remark,
		Name:       &name,
		Status:     entity.Status,
		ExpiredAt:  entity.ExpiredAt,
		LastUsedAt: entity.LastUsed,
		CreatedAt:  entity.CreatedAt,
		UpdatedAt:  entity.UpdatedAt,
		CreatedID:  entity.CreatedID,
	}
}

// EnsureTenantDefaultOpenAPIKey 租户首次创建时自动创建默认密钥
func (*OpenAPIKey) EnsureTenantDefaultOpenAPIKey(ctx context.Context, tenantID, creatorID string) error {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil
	}

	count, err := dal.CountTenantOpenAPIKeys(ctx, tenantID)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	appID, err := utils.GenerateAppID()
	if err != nil {
		return err
	}
	secretKey, err := utils.GenerateAPIKey()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	expiredAt := now.AddDate(1, 0, 0)
	status := int16(1)
	remark := "租户初始化默认密钥"

	key := &dal.OpenAPIKeyEntity{
		ID:        uuid.New(),
		TenantID:  tenantID,
		APIKey:    appID,
		SecretKey: &secretKey,
		Status:    &status,
		Name:      remark,
		Remark:    &remark,
		ExpiredAt: &expiredAt,
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	if strings.TrimSpace(creatorID) != "" {
		key.CreatedID = &creatorID
	}

	return dal.CreateOpenAPIKey(key)
}

// CreateOpenAPIKey 创建 OpenAPI 密钥
func (*OpenAPIKey) CreateOpenAPIKey(req *model.CreateOpenAPIKeyReq, claims *utils.UserClaims) (*model.OpenAPIKeyListRsp, error) {
	tenantID, err := resolveKeyTenantID(req, claims)
	if err != nil {
		return nil, err
	}

	expiredAt, err := parseOpenAPIExpiredAt(req.ExpiredAt)
	if err != nil {
		return nil, err
	}
	if expiredAt == nil {
		defaultExpiredAt := time.Now().UTC().AddDate(1, 0, 0)
		expiredAt = &defaultExpiredAt
	}

	appID, err := utils.GenerateAppID()
	if err != nil {
		logrus.Errorf("生成 AppID 失败: %v", err)
		return nil, errcode.New(errcode.CodeSystemError)
	}
	secretKey, err := utils.GenerateAPIKey()
	if err != nil {
		logrus.Errorf("生成 SecretKey 失败: %v", err)
		return nil, errcode.New(errcode.CodeSystemError)
	}

	status := int16(1)
	if req.Status != nil {
		status = *req.Status
	}

	now := time.Now().UTC()
	remark := resolveOpenAPIKeyRemark(req)
	entity := &dal.OpenAPIKeyEntity{
		ID:        uuid.New(),
		TenantID:  tenantID,
		APIKey:    appID,
		SecretKey: &secretKey,
		Status:    &status,
		Name:      remark,
		Remark:    &remark,
		ExpiredAt: expiredAt,
		CreatedAt: &now,
		UpdatedAt: &now,
		CreatedID: &claims.ID,
	}

	if err := dal.CreateOpenAPIKey(entity); err != nil {
		logrus.Errorf("创建 OpenAPI 密钥失败: %v", err)
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return buildOpenAPIKeyResp(entity), nil
}

// GetOpenAPIKeyList 获取 OpenAPI 密钥列表
func (*OpenAPIKey) GetOpenAPIKeyList(req *model.OpenAPIKeyListReq, claims *utils.UserClaims) (map[string]interface{}, error) {
	var tenantID string
	switch claims.Authority {
	case "SYS_ADMIN":
		if req.TenantID != nil {
			tenantID = strings.TrimSpace(*req.TenantID)
		}
	case "TENANT_ADMIN", "TENANT_USER":
		tenantID = claims.TenantID
	default:
		return nil, errcode.WithVars(errcode.CodeNoPermission, map[string]interface{}{
			"required_role": "SYS_ADMIN or TENANT_ADMIN",
			"current_role":  claims.Authority,
		})
	}

	total, list, err := dal.GetOpenAPIKeyListByPage(req, tenantID)
	if err != nil {
		logrus.Errorf("查询 OpenAPI 密钥列表失败: %v", err)
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return map[string]interface{}{
		"total": total,
		"list":  list,
	}, nil
}

// UpdateOpenAPIKey 更新 OpenAPI 密钥
func (*OpenAPIKey) UpdateOpenAPIKey(req *model.UpdateOpenAPIKeyReq, claims *utils.UserClaims) (*model.OpenAPIKeyListRsp, error) {
	key, err := dal.GetOpenAPIKeyEntityByID(req.ID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"error": err.Error(),
			"id":    req.ID,
		})
	}

	if claims.Authority != "SYS_ADMIN" {
		if claims.Authority != "TENANT_ADMIN" || key.TenantID != claims.TenantID {
			return nil, errcode.WithVars(errcode.CodeNoPermission, map[string]interface{}{
				"required_role": "SYS_ADMIN or TENANT_ADMIN",
				"current_role":  claims.Authority,
			})
		}
	}

	updates := make(map[string]interface{})

	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if req.Remark != nil {
		remark := strings.TrimSpace(*req.Remark)
		updates["remark"] = remark
		updates["name"] = remark
	}

	if req.Name != nil && req.Remark == nil {
		remark := strings.TrimSpace(*req.Name)
		updates["remark"] = remark
		updates["name"] = remark
	}

	if req.ExpiredAt != nil {
		expiredAt, parseErr := parseOpenAPIExpiredAt(req.ExpiredAt)
		if parseErr != nil {
			return nil, parseErr
		}
		updates["expired_at"] = expiredAt
	}

	var rotatedSecret *string
	if req.RotateSecret != nil && *req.RotateSecret {
		newSecret, genErr := utils.GenerateAPIKey()
		if genErr != nil {
			return nil, errcode.New(errcode.CodeSystemError)
		}
		rotatedSecret = &newSecret
		updates["secret_key"] = newSecret
	}

	if len(updates) > 0 {
		if err := dal.UpdateOpenAPIKey(req.ID, updates); err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"error": err.Error(),
				"id":    req.ID,
			})
		}
	}

	latest, err := dal.GetOpenAPIKeyEntityByID(req.ID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"error": err.Error(),
			"id":    req.ID,
		})
	}
	if rotatedSecret != nil {
		latest.SecretKey = rotatedSecret
	}
	return buildOpenAPIKeyResp(latest), nil
}

// DeleteOpenAPIKey 删除 OpenAPI 密钥
func (*OpenAPIKey) DeleteOpenAPIKey(id string, claims *utils.UserClaims) error {
	key, err := dal.GetOpenAPIKeyByID(id)
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
	}

	if claims.Authority != "SYS_ADMIN" {
		if claims.Authority != "TENANT_ADMIN" || key.TenantID != claims.TenantID {
			return errcode.WithVars(errcode.CodeNoPermission, map[string]interface{}{
				"required_role": "SYS_ADMIN or TENANT_ADMIN",
				"current_role":  claims.Authority,
			})
		}
	}

	if err := dal.DeleteOpenAPIKey(id); err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
	}
	return nil
}
