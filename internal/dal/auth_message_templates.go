package dal

import (
	"context"
	"time"

	"project/pkg/global"
)

const (
	TemplateChannelEmail = "EMAIL"
	TemplateChannelSMS   = "SMS"

	TemplateSceneLogin         = "LOGIN"
	TemplateSceneRegister      = "REGISTER"
	TemplateSceneResetPassword = "RESET_PASSWORD"
	TemplateSceneBind          = "BIND"

	TemplateStatusOpen  = "OPEN"
	TemplateStatusClose = "CLOSE"
)

type AuthMessageTemplate struct {
	ID                   string    `gorm:"column:id"`
	TenantID             string    `gorm:"column:tenant_id"`
	Channel              string    `gorm:"column:channel"`
	Scene                string    `gorm:"column:scene"`
	Subject              *string   `gorm:"column:subject"`
	Content              *string   `gorm:"column:content"`
	Provider             *string   `gorm:"column:provider"`
	ProviderTemplateCode *string   `gorm:"column:provider_template_code"`
	Status               string    `gorm:"column:status"`
	Remark               *string   `gorm:"column:remark"`
	CreatedAt            time.Time `gorm:"column:created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
}

func (AuthMessageTemplate) TableName() string { return "auth_message_templates" }

func GetAuthMessageTemplate(ctx context.Context, tenantID, channel, scene string) (*AuthMessageTemplate, error) {
	var out AuthMessageTemplate
	err := global.DB.WithContext(ctx).
		Table("auth_message_templates").
		Where("tenant_id = ? AND channel = ? AND scene = ?", tenantID, channel, scene).
		First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func UpsertAuthMessageTemplate(ctx context.Context, tpl *AuthMessageTemplate) error {
	return global.DB.WithContext(ctx).Exec(`
		INSERT INTO auth_message_templates (
			id, tenant_id, channel, scene, subject, content, provider, provider_template_code, status, remark, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON CONFLICT (tenant_id, channel, scene)
		DO UPDATE SET
			subject = EXCLUDED.subject,
			content = EXCLUDED.content,
			provider = EXCLUDED.provider,
			provider_template_code = EXCLUDED.provider_template_code,
			status = EXCLUDED.status,
			remark = EXCLUDED.remark,
			updated_at = NOW()
	`, tpl.ID, tpl.TenantID, tpl.Channel, tpl.Scene, tpl.Subject, tpl.Content, tpl.Provider, tpl.ProviderTemplateCode, tpl.Status, tpl.Remark).Error
}

func ListAuthMessageTemplates(ctx context.Context, tenantID string) ([]AuthMessageTemplate, error) {
	var list []AuthMessageTemplate
	err := global.DB.WithContext(ctx).
		Table("auth_message_templates").
		Where("tenant_id = ?", tenantID).
		Order("channel ASC, scene ASC").
		Find(&list).Error
	return list, err
}
