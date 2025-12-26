package api

import (
	"strings"

	dal "project/internal/dal"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/errcode"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AppAuthConfigApi struct{}

func getTenantIDForConfig(c *gin.Context, claims *utils.UserClaims) (string, error) {
	if claims.Authority == dal.SYS_ADMIN {
		tenantID := strings.TrimSpace(c.Query("tenant_id"))
		if tenantID == "" {
			return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"error":     "tenant_id is required for SYS_ADMIN",
				"tenant_id": tenantID,
			})
		}
		return tenantID, nil
	}
	return claims.TenantID, nil
}

// ListTemplates 获取当前租户认证消息模板（用于WEB配置）
// @Tags APP-Auth-Config
// @Produce json
// @Param tenant_id query string false "SYS_ADMIN指定租户"
// @Success 200 {object} []dal.AuthMessageTemplate
// @Router /api/v1/app/auth/config/templates [get]
func (*AppAuthConfigApi) ListTemplates(c *gin.Context) {
	claims := c.MustGet("claims").(*utils.UserClaims)
	if claims.Authority != dal.TENANT_ADMIN && claims.Authority != dal.SYS_ADMIN {
		c.Error(errcode.New(errcode.CodeNoPermission))
		return
	}
	tenantID, err := getTenantIDForConfig(c, claims)
	if err != nil {
		c.Error(err)
		return
	}
	list, err := service.GroupApp.AppAuthConfig.ListAuthMessageTemplates(c.Request.Context(), tenantID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", list)
}

// UpsertTemplate 创建/更新认证消息模板（用于WEB配置）
// @Tags APP-Auth-Config
// @Accept json
// @Produce json
// @Param tenant_id query string false "SYS_ADMIN指定租户"
// @Param body body model.UpsertAuthMessageTemplateReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/config/templates [post]
func (*AppAuthConfigApi) UpsertTemplate(c *gin.Context) {
	var req model.UpsertAuthMessageTemplateReq
	if !BindAndValidate(c, &req) {
		return
	}

	claims := c.MustGet("claims").(*utils.UserClaims)
	if claims.Authority != dal.TENANT_ADMIN && claims.Authority != dal.SYS_ADMIN {
		c.Error(errcode.New(errcode.CodeNoPermission))
		return
	}

	tenantID, err := getTenantIDForConfig(c, claims)
	if err != nil {
		c.Error(err)
		return
	}

	tpl := dal.AuthMessageTemplate{
		Channel:              req.Channel,
		Scene:                req.Scene,
		Subject:              req.Subject,
		Content:              req.Content,
		Provider:             req.Provider,
		ProviderTemplateCode: req.ProviderTemplateCode,
		Status:               req.Status,
		Remark:               req.Remark,
	}
	if err := service.GroupApp.AppAuthConfig.UpsertAuthMessageTemplate(c.Request.Context(), tenantID, tpl); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// GetWxMpConfig 获取微信小程序配置（用于WEB配置）
// @Tags APP-Auth-Config
// @Produce json
// @Param tenant_id query string false "SYS_ADMIN指定租户"
// @Success 200 {object} dal.WxMpApp
// @Router /api/v1/app/auth/config/wxmp [get]
func (*AppAuthConfigApi) GetWxMpConfig(c *gin.Context) {
	claims := c.MustGet("claims").(*utils.UserClaims)
	if claims.Authority != dal.TENANT_ADMIN && claims.Authority != dal.SYS_ADMIN {
		c.Error(errcode.New(errcode.CodeNoPermission))
		return
	}
	tenantID, err := getTenantIDForConfig(c, claims)
	if err != nil {
		c.Error(err)
		return
	}
	app, err := service.GroupApp.AppAuthConfig.GetWxMpApp(c.Request.Context(), tenantID)
	if err != nil {
		c.Error(err)
		return
	}
	// 不返回 secret 明文
	app.AppSecret = ""
	c.Set("data", app)
}

// UpsertWxMpConfig 创建/更新微信小程序配置（用于WEB配置）
// @Tags APP-Auth-Config
// @Accept json
// @Produce json
// @Param tenant_id query string false "SYS_ADMIN指定租户"
// @Param body body model.UpsertWxMpAppReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/config/wxmp [post]
func (*AppAuthConfigApi) UpsertWxMpConfig(c *gin.Context) {
	var req model.UpsertWxMpAppReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if claims.Authority != dal.TENANT_ADMIN && claims.Authority != dal.SYS_ADMIN {
		c.Error(errcode.New(errcode.CodeNoPermission))
		return
	}
	tenantID, err := getTenantIDForConfig(c, claims)
	if err != nil {
		c.Error(err)
		return
	}
	if err := service.GroupApp.AppAuthConfig.UpsertWxMpApp(c.Request.Context(), tenantID, req.AppID, req.AppSecret, req.Status, req.Remark); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}
