package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

// EndUser BMS: 终端用户路由
type EndUser struct{}

func (*EndUser) InitEndUser(Router *gin.RouterGroup) {
	endUserApi := Router.Group("end_user")
	{
		endUserApi.GET("", api.Controllers.EndUserApi.GetEndUserList)
		endUserApi.GET("/devices", api.Controllers.EndUserApi.GetEndUserDevices)
		endUserApi.POST("/force_unbind", api.Controllers.EndUserApi.ForceUnbind)
	}
}
