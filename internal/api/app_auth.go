package api

import (
	"project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AppAuthApi struct{}

// SendEmailCode 发送邮箱验证码（APP/小程序）
// @Summary APP发送邮箱验证码
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppSendEmailCodeReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/email/code [post]
func (*AppAuthApi) SendEmailCode(c *gin.Context) {
	var req model.AppSendEmailCodeReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.SendEmailCode(c.Request.Context(), tenantID, req.Email, req.Scene); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// SendPhoneCode 发送手机号验证码（APP/小程序）
// @Summary APP发送手机号验证码
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppSendPhoneCodeReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/phone/code [post]
func (*AppAuthApi) SendPhoneCode(c *gin.Context) {
	var req model.AppSendPhoneCodeReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.SendPhoneCode(c.Request.Context(), tenantID, req.PhonePrefix, req.PhoneNumber, req.Scene); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// PhoneLoginByCode 手机号验证码登录
// @Summary 手机号验证码登录
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppPhoneLoginByCodeReq true "请求"
// @Success 200 {object} model.LoginRsp
// @Router /api/v1/app/auth/phone/login_by_code [post]
func (*AppAuthApi) PhoneLoginByCode(c *gin.Context) {
	var req model.AppPhoneLoginByCodeReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantID := middleware.GetTenantIDFromHeader(c)
	rsp, err := service.GroupApp.AppAuth.PhoneLoginByCode(c.Request.Context(), tenantID, req.PhonePrefix, req.PhoneNumber, req.VerifyCode)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", rsp)
}

// EmailLoginByCode 邮箱验证码登录
// @Summary 邮箱验证码登录
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppEmailLoginByCodeReq true "请求"
// @Success 200 {object} model.LoginRsp
// @Router /api/v1/app/auth/email/login_by_code [post]
func (*AppAuthApi) EmailLoginByCode(c *gin.Context) {
	var req model.AppEmailLoginByCodeReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantID := middleware.GetTenantIDFromHeader(c)
	rsp, err := service.GroupApp.AppAuth.EmailLoginByCode(c.Request.Context(), tenantID, req.Email, req.VerifyCode)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", rsp)
}

// PhoneRegister 手机号注册
// @Summary 手机号注册
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppPhoneRegisterReq true "请求"
// @Success 200 {object} model.LoginRsp
// @Router /api/v1/app/auth/phone/register [post]
func (*AppAuthApi) PhoneRegister(c *gin.Context) {
	var req model.AppPhoneRegisterReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantID := middleware.GetTenantIDFromHeader(c)
	rsp, err := service.GroupApp.AppAuth.PhoneRegister(c.Request.Context(), tenantID, req.PhonePrefix, req.PhoneNumber, req.VerifyCode, req.Password)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", rsp)
}

// EmailRegister 邮箱注册
// @Summary 邮箱注册
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppEmailRegisterReq true "请求"
// @Success 200 {object} model.LoginRsp
// @Router /api/v1/app/auth/email/register [post]
func (*AppAuthApi) EmailRegister(c *gin.Context) {
	var req model.AppEmailRegisterReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantID := middleware.GetTenantIDFromHeader(c)
	rsp, err := service.GroupApp.AppAuth.EmailRegister(c.Request.Context(), tenantID, req.Email, req.VerifyCode, req.Password)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", rsp)
}

// PhoneResetPassword 手机号验证码找回/重置密码
// @Summary 手机号重置密码
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppPhoneResetPasswordReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/phone/reset_password [post]
func (*AppAuthApi) PhoneResetPassword(c *gin.Context) {
	var req model.AppPhoneResetPasswordReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.ResetPasswordByPhone(c.Request.Context(), tenantID, req.PhonePrefix, req.PhoneNumber, req.VerifyCode, req.Password); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// EmailResetPassword 邮箱验证码找回/重置密码
// @Summary 邮箱重置密码
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppEmailResetPasswordReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/email/reset_password [post]
func (*AppAuthApi) EmailResetPassword(c *gin.Context) {
	var req model.AppEmailResetPasswordReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.ResetPasswordByEmail(c.Request.Context(), tenantID, req.Email, req.VerifyCode, req.Password); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// WxmpLogin 微信小程序一键登录
// @Summary 微信小程序一键登录
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppWxmpLoginReq true "请求"
// @Success 200 {object} model.LoginRsp
// @Router /api/v1/app/auth/wxmp/login [post]
func (*AppAuthApi) WxmpLogin(c *gin.Context) {
	var req model.AppWxmpLoginReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantID := middleware.GetTenantIDFromHeader(c)
	rsp, err := service.GroupApp.AppAuth.WxmpLogin(c.Request.Context(), tenantID, req.Code)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", rsp)
}

// WxmpBindPhone 微信小程序一键绑定手机号（无需短信验证码）
// @Summary 微信小程序一键绑定手机号
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppWxmpBindPhoneReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/wxmp/bind_phone [post]
func (*AppAuthApi) WxmpBindPhone(c *gin.Context) {
	var req model.AppWxmpBindPhoneReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.WxmpBindPhone(c.Request.Context(), tenantID, userClaims.ID, req.PhoneCode); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// WxmpProfile 微信小程序用户信息解析/保存（wx.getUserProfile 返回）
// @Summary 微信小程序用户信息解析/保存
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppWxmpProfileReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/wxmp/profile [post]
func (*AppAuthApi) WxmpProfile(c *gin.Context) {
	var req model.AppWxmpProfileReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.WxmpUpdateProfile(c.Request.Context(), tenantID, userClaims.ID, &req); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// WxmpBind 微信小程序绑定微信身份（openid）
// @Summary 微信小程序绑定微信身份
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppWxmpBindReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/wxmp/bind [post]
func (*AppAuthApi) WxmpBind(c *gin.Context) {
	var req model.AppWxmpBindReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.WxmpBindOpenID(c.Request.Context(), tenantID, userClaims.ID, req.Code); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// Profile 更新个人资料（昵称/头像）
// @Summary 更新个人资料
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppProfileUpdateReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/profile [post]
func (*AppAuthApi) Profile(c *gin.Context) {
	var req model.AppProfileUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.UpdateProfile(c.Request.Context(), tenantID, userClaims.ID, &req); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// SetUsername 设置用户名（仅允许设置一次）
// @Summary 设置用户名
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppSetUsernameReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/username [post]
func (*AppAuthApi) SetUsername(c *gin.Context) {
	var req model.AppSetUsernameReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.SetUsername(c.Request.Context(), tenantID, userClaims.ID, req.Name); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// Bindings 当前账号绑定信息
// @Summary 当前账号绑定信息
// @Tags APP-Auth
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Success 200 {object} model.AppAuthBindingsResp
// @Router /api/v1/app/auth/bindings [get]
func (*AppAuthApi) Bindings(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	tenantID := middleware.GetTenantIDFromHeader(c)
	data, err := service.GroupApp.AppAuth.GetBindings(c.Request.Context(), tenantID, userClaims.ID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// BindPhone 绑定手机号
// @Summary 绑定手机号
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppBindPhoneReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/bind/phone [post]
func (*AppAuthApi) BindPhone(c *gin.Context) {
	var req model.AppBindPhoneReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.BindPhone(c.Request.Context(), tenantID, userClaims.ID, req.PhonePrefix, req.PhoneNumber, req.VerifyCode); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// BindEmail 绑定邮箱
// @Summary 绑定邮箱
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppBindEmailReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/bind/email [post]
func (*AppAuthApi) BindEmail(c *gin.Context) {
	var req model.AppBindEmailReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	tenantID := middleware.GetTenantIDFromHeader(c)
	if err := service.GroupApp.AppAuth.BindEmail(c.Request.Context(), tenantID, userClaims.ID, req.Email, req.VerifyCode); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// Unbind 解绑账号（手机/邮箱）
// @Summary 解绑账号（手机/邮箱）
// @Tags APP-Auth
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppUnbindReq true "请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/auth/unbind [post]
func (*AppAuthApi) Unbind(c *gin.Context) {
	var req model.AppUnbindReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	tenantID := middleware.GetTenantIDFromHeader(c)

	switch req.IdentityType {
	case "PHONE":
		if err := service.GroupApp.AppAuth.UnbindIdentity(c.Request.Context(), tenantID, userClaims.ID, "PHONE"); err != nil {
			c.Error(err)
			return
		}
	case "EMAIL":
		if err := service.GroupApp.AppAuth.UnbindIdentity(c.Request.Context(), tenantID, userClaims.ID, "EMAIL"); err != nil {
			c.Error(err)
			return
		}
	}
	c.Set("data", nil)
}
