package api

import (
	"strings"

	"project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AppContentApi APP内容管理：单页内容/FAQ/用户反馈
type AppContentApi struct{}

// ---------------------------------------------------------------------------
// APP端（无需登录）
// ---------------------------------------------------------------------------

// GetPageForApp 获取单页内容（用户政策/隐私政策）
// @Summary 获取单页内容
// @Tags APP-Content
// @Produce json
// @Param X-TenantID header string false "租户ID（可选；缺省使用第一个租户）"
// @Param content_key path string true "内容Key(user_policy/privacy_policy)"
// @Param appid query string true "应用AppID"
// @Param lang query string false "语言(zh-CN/en-US)"
// @Success 200 {object} model.AppContentPageResp
// @Router /api/v1/app/content/pages/{content_key} [get]
func (*AppContentApi) GetPageForApp(c *gin.Context) {
	contentKey := strings.TrimSpace(c.Param("content_key"))
	var req model.AppContentPageGetReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantHeader := middleware.GetTenantIDFromHeader(c)
	lang := ""
	if req.Lang != nil {
		lang = *req.Lang
	}
	data, err := service.GroupApp.AppContent.GetPageForApp(c.Request.Context(), tenantHeader, req.AppID, contentKey, lang)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// ListFaqsForApp FAQ列表（无需登录）
// @Summary FAQ列表
// @Tags APP-Content
// @Produce json
// @Param X-TenantID header string false "租户ID（可选；缺省使用第一个租户）"
// @Param appid query string true "应用AppID"
// @Param lang query string false "语言(zh-CN/en-US)"
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Success 200 {object} model.AppFaqListResp
// @Router /api/v1/app/content/faqs [get]
func (*AppContentApi) ListFaqsForApp(c *gin.Context) {
	var req model.AppFaqListReq
	if !BindAndValidate(c, &req) {
		return
	}
	tenantHeader := middleware.GetTenantIDFromHeader(c)
	lang := ""
	if req.Lang != nil {
		lang = *req.Lang
	}
	data, err := service.GroupApp.AppContent.ListFaqsForApp(c.Request.Context(), tenantHeader, req.AppID, lang, req.Page, req.PageSize)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// ---------------------------------------------------------------------------
// APP端（需登录）：用户反馈
// ---------------------------------------------------------------------------

// CreateFeedbackForApp 提交反馈（需登录）
// @Summary 提交反馈
// @Tags APP-Content
// @Accept json
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param body body model.AppFeedbackCreateReq true "反馈内容"
// @Success 200 {object} model.Response
// @Router /api/v1/app/content/feedback [post]
func (*AppContentApi) CreateFeedbackForApp(c *gin.Context) {
	var req model.AppFeedbackCreateReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	tenantHeader := middleware.GetTenantIDFromHeader(c)
	id, err := service.GroupApp.AppContent.CreateFeedbackForApp(c.Request.Context(), claims, tenantHeader, req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", map[string]interface{}{"id": id})
}

// ListMyFeedbackForApp 我的反馈列表（需登录）
// @Summary 我的反馈列表
// @Tags APP-Content
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param appid query string false "应用AppID"
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Success 200 {object} model.AppFeedbackListResp
// @Router /api/v1/app/content/feedback/mine [get]
func (*AppContentApi) ListMyFeedbackForApp(c *gin.Context) {
	var req model.AppFeedbackListReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppContent.ListMyFeedbackForApp(c.Request.Context(), claims, req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetMyFeedbackForApp 我的反馈详情（需登录）
// @Summary 我的反馈详情
// @Tags APP-Content
// @Produce json
// @Param X-TenantID header string true "租户ID"
// @Param id path string true "反馈ID"
// @Success 200 {object} model.AppFeedbackItemResp
// @Router /api/v1/app/content/feedback/{id} [get]
func (*AppContentApi) GetMyFeedbackForApp(c *gin.Context) {
	id := c.Param("id")
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppContent.GetMyFeedbackForApp(c.Request.Context(), claims, id)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// ---------------------------------------------------------------------------
// 管理端（WEB）：单页内容
// ---------------------------------------------------------------------------

// AdminGetPage 获取单页内容（管理端）
// @Summary 获取单页内容
// @Tags APP-Content-Admin
// @Produce json
// @Param content_key path string true "内容Key(user_policy/privacy_policy)"
// @Param app_id query string true "应用ID(apps.id)"
// @Param lang query string false "语言(zh-CN/en-US)"
// @Success 200 {object} model.AdminContentPageResp
// @Router /api/v1/app_content/pages/{content_key} [get]
func (*AppContentApi) AdminGetPage(c *gin.Context) {
	contentKey := c.Param("content_key")
	var req model.AdminContentPageGetReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	lang := ""
	if req.Lang != nil {
		lang = *req.Lang
	}
	data, err := service.GroupApp.AppContent.AdminGetPage(c.Request.Context(), claims, req.AppID, strings.TrimSpace(contentKey), lang)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// AdminUpsertPage 保存单页内容（管理端）
// @Summary 保存单页内容
// @Tags APP-Content-Admin
// @Accept json
// @Produce json
// @Param content_key path string true "内容Key(user_policy/privacy_policy)"
// @Param body body model.AdminContentPageUpsertReq true "内容"
// @Success 200 {object} model.Response
// @Router /api/v1/app_content/pages/{content_key} [put]
func (*AppContentApi) AdminUpsertPage(c *gin.Context) {
	contentKey := c.Param("content_key")
	var req model.AdminContentPageUpsertReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppContent.AdminUpsertPage(c.Request.Context(), claims, strings.TrimSpace(contentKey), req); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// AdminPublishPage 发布单页内容
// @Summary 发布单页内容
// @Tags APP-Content-Admin
// @Accept json
// @Produce json
// @Param content_key path string true "内容Key(user_policy/privacy_policy)"
// @Param body body model.AdminContentPagePublishReq true "应用ID"
// @Success 200 {object} model.Response
// @Router /api/v1/app_content/pages/{content_key}/publish [post]
func (*AppContentApi) AdminPublishPage(c *gin.Context) {
	contentKey := c.Param("content_key")
	var req model.AdminContentPagePublishReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppContent.AdminSetPagePublish(c.Request.Context(), claims, strings.TrimSpace(contentKey), req.AppID, true); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// AdminUnpublishPage 下线单页内容
// @Summary 下线单页内容
// @Tags APP-Content-Admin
// @Accept json
// @Produce json
// @Param content_key path string true "内容Key(user_policy/privacy_policy)"
// @Param body body model.AdminContentPagePublishReq true "应用ID"
// @Success 200 {object} model.Response
// @Router /api/v1/app_content/pages/{content_key}/unpublish [post]
func (*AppContentApi) AdminUnpublishPage(c *gin.Context) {
	contentKey := c.Param("content_key")
	var req model.AdminContentPagePublishReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppContent.AdminSetPagePublish(c.Request.Context(), claims, strings.TrimSpace(contentKey), req.AppID, false); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// ---------------------------------------------------------------------------
// 管理端（WEB）：FAQ
// ---------------------------------------------------------------------------

// AdminListFaqs FAQ列表
// @Summary FAQ列表
// @Tags APP-Content-Admin
// @Produce json
// @Param app_id query string true "应用ID(apps.id)"
// @Param lang query string false "语言(zh-CN/en-US)"
// @Param keyword query string false "关键词"
// @Param published query bool false "是否发布"
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Success 200 {object} model.AdminFaqListResp
// @Router /api/v1/app_content/faqs [get]
func (*AppContentApi) AdminListFaqs(c *gin.Context) {
	var req model.AdminFaqListReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppContent.AdminListFaqs(c.Request.Context(), claims, req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// AdminGetFaq FAQ详情
// @Summary FAQ详情
// @Tags APP-Content-Admin
// @Produce json
// @Param id path string true "FAQ ID"
// @Success 200 {object} model.AdminFaqDetailResp
// @Router /api/v1/app_content/faqs/{id} [get]
func (*AppContentApi) AdminGetFaq(c *gin.Context) {
	id := c.Param("id")
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppContent.AdminGetFaq(c.Request.Context(), claims, id)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// AdminCreateFaq 新增FAQ
// @Summary 新增FAQ
// @Tags APP-Content-Admin
// @Accept json
// @Produce json
// @Param body body model.AdminFaqCreateReq true "FAQ"
// @Success 200 {object} model.Response
// @Router /api/v1/app_content/faqs [post]
func (*AppContentApi) AdminCreateFaq(c *gin.Context) {
	var req model.AdminFaqCreateReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	id, err := service.GroupApp.AppContent.AdminCreateFaq(c.Request.Context(), claims, req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", map[string]interface{}{"id": id})
}

// AdminUpdateFaq 更新FAQ
// @Summary 更新FAQ
// @Tags APP-Content-Admin
// @Accept json
// @Produce json
// @Param id path string true "FAQ ID"
// @Param body body model.AdminFaqUpdateReq true "FAQ"
// @Success 200 {object} model.Response
// @Router /api/v1/app_content/faqs/{id} [put]
func (*AppContentApi) AdminUpdateFaq(c *gin.Context) {
	id := c.Param("id")
	var req model.AdminFaqUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppContent.AdminUpdateFaq(c.Request.Context(), claims, id, req); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// AdminDeleteFaq 删除FAQ
// @Summary 删除FAQ
// @Tags APP-Content-Admin
// @Produce json
// @Param id path string true "FAQ ID"
// @Success 200 {object} model.Response
// @Router /api/v1/app_content/faqs/{id} [delete]
func (*AppContentApi) AdminDeleteFaq(c *gin.Context) {
	id := c.Param("id")
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppContent.AdminDeleteFaq(c.Request.Context(), claims, id); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// AdminBatchDeleteFaqs 批量删除FAQ
// @Summary 批量删除FAQ
// @Tags APP-Content-Admin
// @Accept json
// @Produce json
// @Param body body model.BatchDeleteReq true "批量删除"
// @Success 200 {object} model.Response
// @Router /api/v1/app_content/faqs/batch_delete [post]
func (*AppContentApi) AdminBatchDeleteFaqs(c *gin.Context) {
	var req model.BatchDeleteReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppContent.AdminBatchDeleteFaqs(c.Request.Context(), claims, req.IDs); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// ---------------------------------------------------------------------------
// 管理端（WEB）：用户反馈
// ---------------------------------------------------------------------------

// AdminListFeedback 用户反馈列表
// @Summary 用户反馈列表
// @Tags APP-Content-Admin
// @Produce json
// @Param app_id query string true "应用ID(apps.id)"
// @Param status query string false "状态"
// @Param keyword query string false "关键词"
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Success 200 {object} model.AdminFeedbackListResp
// @Router /api/v1/app_content/feedback [get]
func (*AppContentApi) AdminListFeedback(c *gin.Context) {
	var req model.AdminFeedbackListReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppContent.AdminListFeedback(c.Request.Context(), claims, req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// AdminGetFeedback 用户反馈详情
// @Summary 用户反馈详情
// @Tags APP-Content-Admin
// @Produce json
// @Param id path string true "反馈ID"
// @Success 200 {object} model.AdminFeedbackDetailResp
// @Router /api/v1/app_content/feedback/{id} [get]
func (*AppContentApi) AdminGetFeedback(c *gin.Context) {
	id := c.Param("id")
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppContent.AdminGetFeedback(c.Request.Context(), claims, id)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// AdminUpdateFeedback 更新反馈（状态/回复/备注）
// @Summary 更新反馈
// @Tags APP-Content-Admin
// @Accept json
// @Produce json
// @Param id path string true "反馈ID"
// @Param body body model.AdminFeedbackUpdateReq true "更新"
// @Success 200 {object} model.Response
// @Router /api/v1/app_content/feedback/{id} [put]
func (*AppContentApi) AdminUpdateFeedback(c *gin.Context) {
	id := c.Param("id")
	var req model.AdminFeedbackUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppContent.AdminUpdateFeedback(c.Request.Context(), claims, id, req); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}
