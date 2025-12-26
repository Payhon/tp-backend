package dal

import (
	"context"
	"time"

	"project/pkg/global"

	"gorm.io/gorm"
)

const (
	IdentityTypePhone      = "PHONE"
	IdentityTypeEmail      = "EMAIL"
	IdentityTypeWxmpOpenID = "WXMP_OPENID"

	CredentialTypePassword = "PASSWORD"
	CredentialTypeCode     = "CODE"
)

type UserIdentity struct {
	ID             string     `gorm:"column:id"`
	UserID         string     `gorm:"column:user_id"`
	TenantID       string     `gorm:"column:tenant_id"`
	IdentityType   string     `gorm:"column:identity_type"`
	Identifier     string     `gorm:"column:identifier"`
	CredentialType string     `gorm:"column:credential_type"`
	PasswordHash   *string    `gorm:"column:password_hash"`
	VerifiedAt     *time.Time `gorm:"column:verified_at"`
	IsPrimary      bool       `gorm:"column:is_primary"`
	Status         string     `gorm:"column:status"`
	Extra          *string    `gorm:"column:extra"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (UserIdentity) TableName() string { return "user_identities" }

func GetUserIdentity(ctx context.Context, tenantID, identityType, identifier string) (*UserIdentity, error) {
	var out UserIdentity
	err := global.DB.WithContext(ctx).
		Table("user_identities").
		Where("tenant_id = ? AND identity_type = ? AND identifier = ?", tenantID, identityType, identifier).
		First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func ListUserIdentitiesByUser(ctx context.Context, tenantID, userID string) ([]UserIdentity, error) {
	var list []UserIdentity
	err := global.DB.WithContext(ctx).
		Table("user_identities").
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Order("is_primary DESC, created_at ASC").
		Find(&list).Error
	return list, err
}

func CreateUserIdentity(ctx context.Context, tx *gorm.DB, identity *UserIdentity) error {
	db := global.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Table("user_identities").Create(identity).Error
}

func UpdateUserIdentity(ctx context.Context, tx *gorm.DB, tenantID, id string, updates map[string]interface{}) error {
	db := global.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).
		Table("user_identities").
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Updates(updates).Error
}

func DeleteUserIdentity(ctx context.Context, tx *gorm.DB, tenantID, id string) error {
	db := global.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).
		Table("user_identities").
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&UserIdentity{}).Error
}
