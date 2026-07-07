package dal

import (
	"context"
	"time"

	"project/pkg/global"

	"gorm.io/gorm"
)

type PackWxMpConfig struct {
	ID                   string    `gorm:"column:id" json:"id"`
	TenantID             string    `gorm:"column:tenant_id" json:"tenant_id"`
	OrgID                string    `gorm:"column:org_id" json:"org_id"`
	AppID                string    `gorm:"column:app_id" json:"app_id"`
	WxAppID              string    `gorm:"column:wx_appid" json:"wx_appid"`
	AppSecret            string    `gorm:"column:app_secret" json:"app_secret"`
	Status               string    `gorm:"column:status" json:"status"`
	HomeBannerURL        *string   `gorm:"column:home_banner_url" json:"home_banner_url"`
	LoginLogoURL         *string   `gorm:"column:login_logo_url" json:"login_logo_url"`
	WarrantyCardsEnabled bool      `gorm:"column:warranty_cards_enabled" json:"warranty_cards_enabled"`
	Remark               *string   `gorm:"column:remark" json:"remark"`
	CreatedAt            time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (PackWxMpConfig) TableName() string { return "pack_wxmp_configs" }

func GetPackWxMpConfigByOrg(ctx context.Context, tenantID, orgID string) (*PackWxMpConfig, error) {
	var out PackWxMpConfig
	err := global.DB.WithContext(ctx).
		Table("pack_wxmp_configs").
		Where("tenant_id = ? AND org_id = ?", tenantID, orgID).
		First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func GetPackWxMpConfigByWxAppID(ctx context.Context, tenantID, wxAppID string) (*PackWxMpConfig, error) {
	var out PackWxMpConfig
	err := global.DB.WithContext(ctx).
		Table("pack_wxmp_configs").
		Where("tenant_id = ? AND wx_appid = ?", tenantID, wxAppID).
		First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func UpsertPackWxMpConfig(ctx context.Context, tx *gorm.DB, app *PackWxMpConfig) error {
	db := global.DB
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Exec(`
		INSERT INTO pack_wxmp_configs (
			id, tenant_id, org_id, app_id, wx_appid, app_secret,
			status, home_banner_url, login_logo_url, warranty_cards_enabled, remark, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (tenant_id, org_id)
		DO UPDATE SET
			app_id = EXCLUDED.app_id,
			wx_appid = EXCLUDED.wx_appid,
			app_secret = EXCLUDED.app_secret,
			status = EXCLUDED.status,
			home_banner_url = EXCLUDED.home_banner_url,
			login_logo_url = EXCLUDED.login_logo_url,
			warranty_cards_enabled = EXCLUDED.warranty_cards_enabled,
			remark = EXCLUDED.remark,
			updated_at = CURRENT_TIMESTAMP
	`, app.ID, app.TenantID, app.OrgID, app.AppID, app.WxAppID, app.AppSecret, app.Status, app.HomeBannerURL, app.LoginLogoURL, app.WarrantyCardsEnabled, app.Remark).Error
}
