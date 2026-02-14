package dal

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	model "project/internal/model"
	global "project/pkg/global"
)

// OpenAPIKeyEntity open_api_keys 扩展字段实体
type OpenAPIKeyEntity struct {
	ID        string     `gorm:"column:id"`
	TenantID  string     `gorm:"column:tenant_id"`
	APIKey    string     `gorm:"column:api_key"`
	SecretKey *string    `gorm:"column:secret_key"`
	Status    *int16     `gorm:"column:status"`
	Name      string     `gorm:"column:name"`
	Remark    *string    `gorm:"column:remark"`
	ExpiredAt *time.Time `gorm:"column:expired_at"`
	CreatedAt *time.Time `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
	CreatedID *string    `gorm:"column:created_id"`
	LastUsed  *time.Time `gorm:"column:last_used_at"`
}

func (OpenAPIKeyEntity) TableName() string {
	return model.TableNameOpenAPIKey
}

// CreateOpenAPIKey 创建新的 OpenAPI 密钥
func CreateOpenAPIKey(key *OpenAPIKeyEntity) error {
	return global.DB.Table(model.TableNameOpenAPIKey).Create(key).Error
}

// GetOpenAPIKeyByID 根据ID获取 OpenAPI 密钥信息（基础字段）
func GetOpenAPIKeyByID(id string) (*model.OpenAPIKey, error) {
	var key model.OpenAPIKey
	if err := global.DB.Table(model.TableNameOpenAPIKey).
		Where("id = ?", id).
		Take(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// GetOpenAPIKeyEntityByID 根据ID获取 OpenAPI 密钥扩展信息
func GetOpenAPIKeyEntityByID(id string) (*OpenAPIKeyEntity, error) {
	var key OpenAPIKeyEntity
	if err := global.DB.Table(model.TableNameOpenAPIKey).
		Where("id = ?", id).
		Take(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// GetOpenAPIKeyByAppKey 根据 AppKey 获取 OpenAPI 密钥信息（基础字段）
func GetOpenAPIKeyByAppKey(appKey string) (*model.OpenAPIKey, error) {
	var key model.OpenAPIKey
	if err := global.DB.Table(model.TableNameOpenAPIKey).
		Where("api_key = ?", appKey).
		Take(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// CountTenantOpenAPIKeys 统计租户下密钥数量
func CountTenantOpenAPIKeys(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := global.DB.WithContext(ctx).Table(model.TableNameOpenAPIKey).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error
	return count, err
}

// GetOpenAPIKeyListByPage 分页获取 OpenAPI 密钥列表
func GetOpenAPIKeyListByPage(listReq *model.OpenAPIKeyListReq, tenantID string) (int64, []model.OpenAPIKeyListRsp, error) {
	db := global.DB.WithContext(context.Background()).
		Table(model.TableNameOpenAPIKey + " AS k").
		Joins("LEFT JOIN users u ON u.id = k.created_id")

	if tenantID != "" {
		db = db.Where("k.tenant_id = ?", tenantID)
	}

	if listReq.Status != nil {
		db = db.Where("k.status = ?", *listReq.Status)
	}

	if listReq.Keyword != nil && strings.TrimSpace(*listReq.Keyword) != "" {
		kw := "%" + strings.TrimSpace(*listReq.Keyword) + "%"
		db = db.Where("(k.api_key ILIKE ? OR k.remark ILIKE ? OR k.name ILIKE ?)", kw, kw, kw)
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, nil, err
	}

	if listReq.Page > 0 && listReq.PageSize > 0 {
		db = db.Offset((listReq.Page - 1) * listReq.PageSize).Limit(listReq.PageSize)
	}

	list := make([]model.OpenAPIKeyListRsp, 0)
	if err := db.Select(`
		k.id,
		k.tenant_id,
		k.api_key,
		k.secret_key,
		k.remark,
		k.name,
		k.status,
		k.expired_at,
		k.last_used_at,
		k.created_at,
		k.updated_at,
		k.created_id,
		u.id AS user_id,
		u.email AS email,
		u.name AS user_name
	`).Order("k.created_at DESC").Scan(&list).Error; err != nil {
		return 0, nil, err
	}

	return count, list, nil
}

// UpdateOpenAPIKey 更新 OpenAPI 密钥信息
func UpdateOpenAPIKey(id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now().UTC()
	return global.DB.Table(model.TableNameOpenAPIKey).
		Where("id = ?", id).
		Updates(updates).Error
}

// DeleteOpenAPIKey 删除 OpenAPI 密钥
func DeleteOpenAPIKey(id string) error {
	return global.DB.Table(model.TableNameOpenAPIKey).
		Where("id = ?", id).
		Delete(&OpenAPIKeyEntity{}).Error
}

func isKeyExpired(expiredAt *time.Time) bool {
	return expiredAt != nil && expiredAt.Before(time.Now().UTC())
}

// VerifyOpenAPIKey 兼容旧认证方式（仅校验 x-api-key）
func VerifyOpenAPIKey(ctx context.Context, appKey string) (string, string, error) {
	var key OpenAPIKeyEntity
	err := global.DB.WithContext(ctx).Table(model.TableNameOpenAPIKey).
		Where("api_key = ? AND status = 1", appKey).
		Take(&key).Error
	if err != nil {
		return "", "", err
	}

	if isKeyExpired(key.ExpiredAt) {
		return "", "", gorm.ErrRecordNotFound
	}

	createdID := ""
	if key.CreatedID != nil {
		createdID = *key.CreatedID
	}
	return key.TenantID, createdID, nil
}

// VerifyOpenAPICredentials 校验 app_id + secret_key
func VerifyOpenAPICredentials(ctx context.Context, appID, secretKey string) (tenantID, createdID, keyID string, err error) {
	var key OpenAPIKeyEntity
	err = global.DB.WithContext(ctx).Table(model.TableNameOpenAPIKey).
		Where("api_key = ? AND status = 1", appID).
		Take(&key).Error
	if err != nil {
		return "", "", "", err
	}

	if isKeyExpired(key.ExpiredAt) {
		return "", "", "", gorm.ErrRecordNotFound
	}

	if key.SecretKey == nil || *key.SecretKey == "" {
		return "", "", "", gorm.ErrRecordNotFound
	}

	if subtle.ConstantTimeCompare([]byte(*key.SecretKey), []byte(secretKey)) != 1 {
		return "", "", "", errors.New("invalid secret key")
	}

	now := time.Now().UTC()
	_ = global.DB.WithContext(ctx).Table(model.TableNameOpenAPIKey).
		Where("id = ?", key.ID).
		Update("last_used_at", &now).Error

	createdID = ""
	if key.CreatedID != nil {
		createdID = *key.CreatedID
	}
	return key.TenantID, createdID, key.ID, nil
}
