package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

// AppManage APP管理模块路由
type AppManage struct{}

func (*AppManage) InitAppManage(Router *gin.RouterGroup) {
	apps := Router.Group("apps")
	{
		apps.GET("", api.Controllers.AppManageApi.ListApps)
		apps.POST("", api.Controllers.AppManageApi.CreateApp)
		apps.POST("/batch_delete", api.Controllers.AppManageApi.BatchDeleteApps)
		apps.GET("/:id", api.Controllers.AppManageApi.GetApp)
		apps.PUT("/:id", api.Controllers.AppManageApi.UpdateApp)
		apps.DELETE("/:id", api.Controllers.AppManageApi.DeleteApp)
	}

	versions := Router.Group("app_versions")
	{
		versions.GET("", api.Controllers.AppManageApi.ListAppVersions)
		versions.POST("", api.Controllers.AppManageApi.CreateAppVersion)
		versions.POST("/batch_delete", api.Controllers.AppManageApi.BatchDeleteAppVersions)
		versions.GET("/:id", api.Controllers.AppManageApi.GetAppVersion)
		versions.PUT("/:id", api.Controllers.AppManageApi.UpdateAppVersion)
		versions.DELETE("/:id", api.Controllers.AppManageApi.DeleteAppVersion)
	}
}

