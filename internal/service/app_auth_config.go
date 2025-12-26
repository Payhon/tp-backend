package service

import (
	"context"
	"errors"
	"strings"
	"time"

	dal "project/internal/dal"
	"project/pkg/errcode"

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
