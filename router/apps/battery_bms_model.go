package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type BatteryBmsModel struct{}

func (*BatteryBmsModel) InitBatteryBmsModel(Router *gin.RouterGroup) {
	bmsModelApi := Router.Group("battery/bms-model")
	{
		bmsModelApi.POST("", api.Controllers.BatteryBmsModelApi.CreateBatteryBmsModel)
		bmsModelApi.DELETE(":id", api.Controllers.BatteryBmsModelApi.DeleteBatteryBmsModel)
		bmsModelApi.PUT(":id", api.Controllers.BatteryBmsModelApi.UpdateBatteryBmsModel)
		bmsModelApi.GET(":id", api.Controllers.BatteryBmsModelApi.GetBatteryBmsModelByID)
		bmsModelApi.GET("", api.Controllers.BatteryBmsModelApi.GetBatteryBmsModelList)
	}
}
