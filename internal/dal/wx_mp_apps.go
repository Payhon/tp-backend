package dal

import (
	"context"
	"time"

	"project/pkg/global"
)

type WxMpApp struct {
	ID        string    `gorm:"column:id"`
	TenantID  string    `gorm:"column:tenant_id"`
	AppID     string    `gorm:"column:appid"`
	AppSecret string    `gorm:"column:app_secret"`
	Status    string    `gorm:"column:status"`
	Remark    *string   `gorm:"column:remark"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (WxMpApp) TableName() string { return "wx_mp_apps" }

func GetWxMpAppByTenant(ctx context.Context, tenantID string) (*WxMpApp, error) {
	var out WxMpApp
	err := global.DB.WithContext(ctx).
		Table("wx_mp_apps").
		Where("tenant_id = ?", tenantID).
		First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func UpsertWxMpApp(ctx context.Context, app *WxMpApp) error {
	// postgres upsert
	return global.DB.WithContext(ctx).Exec(`
		INSERT INTO wx_mp_apps (id, tenant_id, appid, app_secret, status, remark, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET
			appid = EXCLUDED.appid,
			app_secret = EXCLUDED.app_secret,
			status = EXCLUDED.status,
			remark = EXCLUDED.remark,
			updated_at = NOW()
	`, app.ID, app.TenantID, app.AppID, app.AppSecret, app.Status, app.Remark).Error
}
