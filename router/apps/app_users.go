package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type AppUsers struct{}

func (*AppUsers) Init(Router *gin.RouterGroup) {
	appUsers := Router.Group("app_users")
	{
		appUsers.GET("", api.Controllers.AppUsersApi.List)
		appUsers.GET("source-options", api.Controllers.AppUsersApi.SourceOptions)
	}
}
