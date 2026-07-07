package service

import (
	"context"
	"errors"
	"strings"
	"time"

	dal "project/internal/dal"
	"project/internal/model"
	"project/pkg/errcode"
	"project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

type AppAuthConfig struct{}

func (*AppAuthConfig) UpsertAuthMessageTemplate(ctx context.Context, tenantID string, tpl dal.AuthMessageTemplate) error {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "tenant_id is empty"})
	}
	tpl.TenantID = tenantID
	tpl.Channel = strings.ToUpper(strings.TrimSpace(tpl.Channel))
	tpl.Scene = strings.ToUpper(strings.TrimSpace(tpl.Scene))
	if tpl.Channel == "" || tpl.Scene == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "channel/scene is empty"})
	}
	if tpl.Status == "" {
		tpl.Status = dal.TemplateStatusOpen
	}
	if tpl.ID == "" {
		tpl.ID = uuid.New()
	}
	return dal.UpsertAuthMessageTemplate(ctx, &tpl)
}

func (*AppAuthConfig) ListAuthMessageTemplates(ctx context.Context, tenantID string) ([]dal.AuthMessageTemplate, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "tenant_id is empty"})
	}
	return dal.ListAuthMessageTemplates(ctx, tenantID)
}

func (*AppAuthConfig) UpsertWxMpApp(ctx context.Context, tenantID, appid, secret, status string, remark *string) error {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" || strings.TrimSpace(appid) == "" || strings.TrimSpace(secret) == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "tenant_id/appid/secret is empty"})
	}
	if status == "" {
		status = "OPEN"
	}
	now := time.Now().UTC()
	app := &dal.WxMpApp{
		ID:        uuid.New(),
		TenantID:  tenantID,
		AppID:     strings.TrimSpace(appid),
		AppSecret: strings.TrimSpace(secret),
		Status:    strings.ToUpper(strings.TrimSpace(status)),
		Remark:    remark,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return dal.UpsertWxMpApp(ctx, app)
}

func (*AppAuthConfig) GetWxMpApp(ctx context.Context, tenantID string) (*dal.WxMpApp, error) {
	app, err := dal.GetWxMpAppByTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"error": "wx mp app not configured"})
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	return app, nil
}

type packWxMpAppCreateRow struct {
	ID        string    `gorm:"column:id"`
	TenantID  string    `gorm:"column:tenant_id"`
	AppID     string    `gorm:"column:appid"`
	AppType   int16     `gorm:"column:app_type"`
	Name      string    `gorm:"column:name"`
	Remark    *string   `gorm:"column:remark"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func trimStringPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func packWxMpResp(row *dal.PackWxMpConfig) *model.PackWxMpConfigResp {
	if row == nil {
		return nil
	}
	return &model.PackWxMpConfigResp{
		ID:                   row.ID,
		TenantID:             row.TenantID,
		OrgID:                row.OrgID,
		AppID:                row.AppID,
		WxAppID:              row.WxAppID,
		Status:               row.Status,
		HomeBannerURL:        row.HomeBannerURL,
		LoginLogoURL:         row.LoginLogoURL,
		WarrantyCardsEnabled: row.WarrantyCardsEnabled,
		Remark:               row.Remark,
		CreatedAt:            row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            row.UpdatedAt.Format(time.RFC3339),
	}
}

func ensurePackOrg(ctx context.Context, tenantID, orgID string) error {
	var org model.Org
	if err := global.DB.WithContext(ctx).
		Table("orgs").
		Select("id, tenant_id, org_type").
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(orgID)).
		First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.New(errcode.CodeNotFound)
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if org.OrgType != model.OrgTypePACKFactory {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "仅支持配置 PACK 厂小程序"})
	}
	return nil
}

func ensurePackWxMpConfigAccess(ctx context.Context, claims *utils.UserClaims, tenantID, orgID string) error {
	if claims == nil {
		return errcode.New(errcode.CodeNoPermission)
	}
	if claims.Authority == dal.SYS_ADMIN || claims.Authority == dal.TENANT_ADMIN {
		return nil
	}
	if claims.Authority != dal.TENANT_USER {
		return errcode.New(errcode.CodeNoPermission)
	}
	if strings.TrimSpace(claims.TenantID) != strings.TrimSpace(tenantID) {
		return errcode.New(errcode.CodeNoPermission)
	}
	if strings.TrimSpace(claims.OrgID) == "" || strings.TrimSpace(claims.OrgID) != strings.TrimSpace(orgID) {
		return errcode.New(errcode.CodeNoPermission)
	}
	var org struct {
		OrgType string `gorm:"column:org_type"`
	}
	if err := global.DB.WithContext(ctx).
		Table("orgs").
		Select("org_type").
		Where("tenant_id = ? AND id = ?", tenantID, orgID).
		Limit(1).
		Scan(&org).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if org.OrgType != model.OrgTypePACKFactory {
		return errcode.New(errcode.CodeNoPermission)
	}
	return nil
}

func getOrgName(ctx context.Context, tenantID, orgID string) string {
	var org struct {
		Name string `gorm:"column:name"`
	}
	if err := global.DB.WithContext(ctx).
		Table("orgs").
		Select("name").
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(orgID)).
		Limit(1).
		Scan(&org).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(org.Name)
}

func ensureAppForPackWxMp(ctx context.Context, tx *gorm.DB, tenantID, wxAppID, orgID string) (string, error) {
	var app struct {
		ID string `gorm:"column:id"`
	}
	if err := tx.WithContext(ctx).
		Table("apps").
		Select("id").
		Where("tenant_id = ? AND appid = ?", tenantID, wxAppID).
		Limit(1).
		Scan(&app).Error; err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if strings.TrimSpace(app.ID) != "" {
		return app.ID, nil
	}
	now := time.Now().UTC()
	row := packWxMpAppCreateRow{
		ID:        uuid.New(),
		TenantID:  tenantID,
		AppID:     wxAppID,
		AppType:   0,
		Name:      "PACK小程序-" + orgID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tx.WithContext(ctx).Table("apps").Create(&row).Error; err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return row.ID, nil
}

func (*AppAuthConfig) GetPackWxMpConfig(ctx context.Context, claims *utils.UserClaims, tenantID, orgID string) (*model.PackWxMpConfigResp, error) {
	tenantID = strings.TrimSpace(tenantID)
	orgID = strings.TrimSpace(orgID)
	if tenantID == "" || orgID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "tenant_id/org_id is empty"})
	}
	if err := ensurePackOrg(ctx, tenantID, orgID); err != nil {
		return nil, err
	}
	if err := ensurePackWxMpConfigAccess(ctx, claims, tenantID, orgID); err != nil {
		return nil, err
	}
	row, err := dal.GetPackWxMpConfigByOrg(ctx, tenantID, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.PackWxMpConfigResp{
				TenantID:             tenantID,
				OrgID:                orgID,
				Status:               "OPEN",
				WarrantyCardsEnabled: true,
			}, nil
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	return packWxMpResp(row), nil
}

func (*AppAuthConfig) UpsertPackWxMpConfig(ctx context.Context, claims *utils.UserClaims, tenantID, orgID string, req *model.UpsertPackWxMpConfigReq) (*model.PackWxMpConfigResp, error) {
	if req == nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "request is empty"})
	}
	tenantID = strings.TrimSpace(tenantID)
	orgID = strings.TrimSpace(orgID)
	wxAppID := strings.TrimSpace(req.WxAppID)
	if tenantID == "" || orgID == "" || wxAppID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "tenant_id/org_id/wx_appid is empty"})
	}
	if err := ensurePackOrg(ctx, tenantID, orgID); err != nil {
		return nil, err
	}
	if err := ensurePackWxMpConfigAccess(ctx, claims, tenantID, orgID); err != nil {
		return nil, err
	}
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status == "" {
		status = "OPEN"
	}
	if status != "OPEN" && status != "CLOSE" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"status": "must be OPEN or CLOSE"})
	}
	secret := trimStringPtrValue(req.AppSecret)
	warrantyCardsEnabled := true
	if existing, err := dal.GetPackWxMpConfigByOrg(ctx, tenantID, orgID); err == nil {
		if secret == "" {
			secret = existing.AppSecret
		}
		warrantyCardsEnabled = existing.WarrantyCardsEnabled
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	if req.WarrantyCardsEnabled != nil {
		warrantyCardsEnabled = *req.WarrantyCardsEnabled
	}
	if secret == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"app_secret": "required"})
	}

	var saved *dal.PackWxMpConfig
	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		appID, err := ensureAppForPackWxMp(ctx, tx, tenantID, wxAppID, orgID)
		if err != nil {
			return err
		}
		row := &dal.PackWxMpConfig{
			ID:                   uuid.New(),
			TenantID:             tenantID,
			OrgID:                orgID,
			AppID:                appID,
			WxAppID:              wxAppID,
			AppSecret:            secret,
			Status:               status,
			HomeBannerURL:        req.HomeBannerURL,
			LoginLogoURL:         req.LoginLogoURL,
			WarrantyCardsEnabled: warrantyCardsEnabled,
			Remark:               req.Remark,
		}
		if err := dal.UpsertPackWxMpConfig(ctx, tx, row); err != nil {
			return err
		}
		var out dal.PackWxMpConfig
		if err := tx.WithContext(ctx).
			Table("pack_wxmp_configs").
			Where("tenant_id = ? AND org_id = ?", tenantID, orgID).
			First(&out).Error; err != nil {
			return err
		}
		saved = &out
		return nil
	})
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	return packWxMpResp(saved), nil
}

func (*AppAuthConfig) GetWxMpRuntime(ctx context.Context, tenantID, wxAppID string) (*model.WxMpRuntimeResp, error) {
	tenantID = strings.TrimSpace(tenantID)
	wxAppID = strings.TrimSpace(wxAppID)
	if tenantID == "" || wxAppID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "tenant_id/appid is empty"})
	}
	row, err := dal.GetPackWxMpConfigByWxAppID(ctx, tenantID, wxAppID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			app, appErr := dal.GetWxMpAppByTenantAndAppID(ctx, tenantID, wxAppID)
			if appErr != nil {
				if errors.Is(appErr, gorm.ErrRecordNotFound) {
					return nil, errcode.New(errcode.CodeNotFound)
				}
				return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": appErr.Error()})
			}
			if strings.ToUpper(app.Status) != "OPEN" {
				return nil, errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"error": "wx miniapp disabled"})
			}
			return &model.WxMpRuntimeResp{
				AppID:                app.ID,
				WxAppID:              app.AppID,
				Status:               app.Status,
				SourceType:           "TENANT",
				LoginOnly:            false,
				WarrantyCardsEnabled: true,
			}, nil
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"error": err.Error()})
	}
	if strings.ToUpper(row.Status) != "OPEN" {
		return nil, errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"error": "wx miniapp disabled"})
	}
	return &model.WxMpRuntimeResp{
		AppID:                row.AppID,
		WxAppID:              row.WxAppID,
		Status:               row.Status,
		SourceType:           "PACK",
		LoginOnly:            true,
		HomeBannerURL:        row.HomeBannerURL,
		LoginLogoURL:         row.LoginLogoURL,
		WarrantyCardsEnabled: row.WarrantyCardsEnabled,
		OrgID:                row.OrgID,
		OrgName:              getOrgName(ctx, tenantID, row.OrgID),
	}, nil
}
