package apps

import (
	"project/internal/api"
	"project/internal/middleware"

	"github.com/gin-gonic/gin"
)

// AppContent APP内容管理：单页/FAQ/用户反馈
type AppContent struct{}

// Init 注册需要登录的接口（包含管理端接口 + APP登录后接口）
func (*AppContent) Init(Router *gin.RouterGroup) {
	// APP登录后：反馈（需 header tenant match token）
	appAuthed := Router.Group("app/content")
	appAuthed.Use(middleware.RequireTenantHeaderMatchClaims())
	{
		appAuthed.POST("feedback", api.Controllers.AppContentApi.CreateFeedbackForApp)
		appAuthed.GET("feedback/mine", api.Controllers.AppContentApi.ListMyFeedbackForApp)
		appAuthed.GET("feedback/:id", api.Controllers.AppContentApi.GetMyFeedbackForApp)
	}

	// WEB管理端：内容管理（走 JWT claims tenant）
	admin := Router.Group("app_content")
	{
		// 单页内容
		admin.GET("pages/:content_key", api.Controllers.AppContentApi.AdminGetPage)
		admin.PUT("pages/:content_key", api.Controllers.AppContentApi.AdminUpsertPage)
		admin.POST("pages/:content_key/publish", api.Controllers.AppContentApi.AdminPublishPage)
		admin.POST("pages/:content_key/unpublish", api.Controllers.AppContentApi.AdminUnpublishPage)

		// FAQ
		admin.GET("faqs", api.Controllers.AppContentApi.AdminListFaqs)
		admin.GET("faqs/:id", api.Controllers.AppContentApi.AdminGetFaq)
		admin.POST("faqs", api.Controllers.AppContentApi.AdminCreateFaq)
		admin.PUT("faqs/:id", api.Controllers.AppContentApi.AdminUpdateFaq)
		admin.DELETE("faqs/:id", api.Controllers.AppContentApi.AdminDeleteFaq)
		admin.POST("faqs/batch_delete", api.Controllers.AppContentApi.AdminBatchDeleteFaqs)

		// 用户反馈
		admin.GET("feedback", api.Controllers.AppContentApi.AdminListFeedback)
		admin.GET("feedback/:id", api.Controllers.AppContentApi.AdminGetFeedback)
		admin.PUT("feedback/:id", api.Controllers.AppContentApi.AdminUpdateFeedback)
	}
}
