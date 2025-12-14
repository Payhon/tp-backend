package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type BatteryMaintenance struct{}

func (*BatteryMaintenance) InitBatteryMaintenance(Router *gin.RouterGroup) {
	url := Router.Group("battery_maintenance")
	{
		url.POST("", api.Controllers.BatteryMaintenanceApi.CreateBatteryMaintenance)
		url.GET("", api.Controllers.BatteryMaintenanceApi.ListBatteryMaintenance)
		url.GET(":id", api.Controllers.BatteryMaintenanceApi.GetBatteryMaintenanceDetail)
	}
}
