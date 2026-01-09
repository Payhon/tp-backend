package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

// AppBattery APP端电池设备（详情/透传）
type AppBattery struct{}

func (*AppBattery) InitAppBattery(Router *gin.RouterGroup) {
	r := Router.Group("app/battery")
	{
		r.GET("detail/:device_id", api.Controllers.AppBatteryApi.GetBatteryDetail)
	}
}

