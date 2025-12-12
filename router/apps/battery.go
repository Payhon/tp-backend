package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

// Battery BMS: 电池管理路由
type Battery struct{}

func (*Battery) InitBattery(Router *gin.RouterGroup) {
	batteryApi := Router.Group("battery")
	{
		// 电池列表（设备电池）
		batteryApi.GET("", api.Controllers.BatteryApi.GetBatteryList)
	}
}
