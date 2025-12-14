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
		// 导出电池列表
		batteryApi.GET("/export", api.Controllers.BatteryApi.ExportBatteryList)
		// 获取导入模板
		batteryApi.GET("/import/template", api.Controllers.BatteryApi.GetBatteryImportTemplate)
		// 导入电池列表
		batteryApi.POST("/import", api.Controllers.BatteryApi.ImportBatteryList)
		// 批量分配经销商
		batteryApi.POST("/batch-assign-dealer", api.Controllers.BatteryApi.BatchAssignDealer)
		// 批量下发指令（在线）
		batteryApi.POST("/batch-command", api.Controllers.BatteryApi.BatchSendCommand)
		// 批量 OTA 推送
		batteryApi.POST("/batch-ota", api.Controllers.BatteryApi.BatchPushOTA)

		// 标签管理
		batteryApi.GET("/tags", api.Controllers.BatteryTagApi.ListBatteryTags)
		batteryApi.POST("/tags", api.Controllers.BatteryTagApi.CreateBatteryTag)
		batteryApi.PUT("/tags/:id", api.Controllers.BatteryTagApi.UpdateBatteryTag)
		batteryApi.DELETE("/tags/:id", api.Controllers.BatteryTagApi.DeleteBatteryTag)
		batteryApi.POST("/tags/assign", api.Controllers.BatteryTagApi.AssignBatteryTags)

		// 离线指令
		batteryApi.GET("/offline-commands", api.Controllers.OfflineCommandApi.ListOfflineCommands)
		batteryApi.POST("/offline-commands", api.Controllers.OfflineCommandApi.CreateOfflineCommand)
		batteryApi.GET("/offline-commands/:id", api.Controllers.OfflineCommandApi.GetOfflineCommandDetail)
		batteryApi.DELETE("/offline-commands/:id", api.Controllers.OfflineCommandApi.CancelOfflineCommand)

		// 参数远程查看/修改（BMS）
		batteryApi.GET("/params/:id", api.Controllers.BatteryApi.GetBatteryParams)
		batteryApi.POST("/params/pub", api.Controllers.BatteryApi.PutBatteryParams)
		batteryApi.POST("/params/get", api.Controllers.BatteryApi.GetBatteryParamsFromDevice)
	}
}
