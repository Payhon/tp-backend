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

		// 标签管理
		batteryApi.GET("/tags", api.Controllers.BatteryTagApi.ListBatteryTags)
		batteryApi.POST("/tags", api.Controllers.BatteryTagApi.CreateBatteryTag)
		batteryApi.PUT("/tags/:id", api.Controllers.BatteryTagApi.UpdateBatteryTag)
		batteryApi.DELETE("/tags/:id", api.Controllers.BatteryTagApi.DeleteBatteryTag)
		batteryApi.POST("/tags/assign", api.Controllers.BatteryTagApi.AssignBatteryTags)
	}
}
