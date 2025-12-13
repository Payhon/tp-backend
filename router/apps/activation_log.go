package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type ActivationLog struct{}

func (*ActivationLog) InitActivationLog(Router *gin.RouterGroup) {
	url := Router.Group("activation_logs")
	{
		url.GET("", api.Controllers.ActivationLogApi.GetActivationLogs)
	}
}

