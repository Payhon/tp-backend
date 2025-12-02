package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type BatteryModel struct{}

func (*BatteryModel) InitBatteryModel(Router *gin.RouterGroup) {
	// 电池型号路由
	batteryModelApi := Router.Group("battery/model")
	{
		// 增
		batteryModelApi.POST("", api.Controllers.BatteryModelApi.CreateBatteryModel)

		// 删
		batteryModelApi.DELETE(":id", api.Controllers.BatteryModelApi.DeleteBatteryModel)

		// 改
		batteryModelApi.PUT(":id", api.Controllers.BatteryModelApi.UpdateBatteryModel)

		// 详情查询
		batteryModelApi.GET(":id", api.Controllers.BatteryModelApi.GetBatteryModelByID)

		// 分页查询
		batteryModelApi.GET("", api.Controllers.BatteryModelApi.GetBatteryModelList)
	}
}
