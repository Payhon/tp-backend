package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type VersionUpdate struct{}

func (*VersionUpdate) InitVersionUpdate(Router *gin.RouterGroup) {
	versionUpdateApi := Router.Group("version-updates")
	{
		versionUpdateApi.POST("", api.Controllers.VersionUpdateApi.CreateVersionUpdate)
		versionUpdateApi.GET("", api.Controllers.VersionUpdateApi.ListVersionUpdates)
		versionUpdateApi.GET(":id", api.Controllers.VersionUpdateApi.GetVersionUpdate)
		versionUpdateApi.PUT(":id", api.Controllers.VersionUpdateApi.UpdateVersionUpdate)
		versionUpdateApi.DELETE(":id", api.Controllers.VersionUpdateApi.DeleteVersionUpdate)
	}
}
