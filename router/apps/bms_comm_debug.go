package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type BmsCommDebug struct{}

func (*BmsCommDebug) InitBmsCommDebug(Router *gin.RouterGroup) {
	url := Router.Group("bms/comm-debug")
	{
		url.GET("logs", api.Controllers.BmsCommDebugApi.GetLogs)
		url.GET("logs/stream", api.Controllers.BmsCommDebugApi.StreamLogs)
	}
}
