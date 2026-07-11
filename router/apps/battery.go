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
		// 添加单个电池
		batteryApi.POST("/single", api.Controllers.BatteryApi.CreateSingleBattery)
		// 编辑单个电池
		batteryApi.PUT("/single/:id", api.Controllers.BatteryApi.UpdateSingleBattery)
		// 删除电池
		batteryApi.DELETE("/:id", api.Controllers.BatteryApi.DeleteBattery)
		// 导出电池列表
		batteryApi.GET("/export", api.Controllers.BatteryApi.ExportBatteryList)
		// 获取导入模板
		batteryApi.GET("/import/template", api.Controllers.BatteryApi.GetBatteryImportTemplate)
		// 导入电池列表
		batteryApi.POST("/import", api.Controllers.BatteryApi.ImportBatteryList)
		// 导入任务状态/日志
		batteryApi.GET("/import/jobs/:id", api.Controllers.BatteryApi.GetBatteryImportJobStatus)
		batteryApi.GET("/import/jobs/:id/logs", api.Controllers.BatteryApi.GetBatteryImportJobLogs)
		// 运营日志
		batteryApi.GET("/operation_logs", api.Controllers.BatteryApi.GetBatteryOperationLogList)
		// 质保截止日期补偿任务
		batteryApi.POST("/warranty/recalculate-jobs", api.Controllers.UserWarrantyInfoApi.CreateBatteryWarrantyRecalcJob)
		batteryApi.GET("/warranty/recalculate-jobs/:id", api.Controllers.UserWarrantyInfoApi.GetBatteryWarrantyRecalcJobStatus)
		batteryApi.GET("/warranty/recalculate-jobs/:id/logs", api.Controllers.UserWarrantyInfoApi.GetBatteryWarrantyRecalcJobLogs)
		// 质保信息
		batteryApi.GET("/:device_id/warranty", api.Controllers.UserWarrantyInfoApi.GetBatteryWarranty)
		batteryApi.PUT("/:device_id/warranty", api.Controllers.UserWarrantyInfoApi.UpdateBatteryWarranty)
		// 生命周期操作
		batteryApi.POST("/factory_out", api.Controllers.BatteryApi.FactoryOutBattery)
		batteryApi.POST("/batch-factory-out", api.Controllers.BatteryApi.BatchFactoryOutBattery)
		batteryApi.POST("/factory_restore", api.Controllers.BatteryApi.FactoryRestoreBattery)
		batteryApi.POST("/transfer", api.Controllers.BatteryApi.TransferBattery)
		batteryApi.GET("/rollback/preview", api.Controllers.BatteryApi.PreviewRollbackBattery)
		batteryApi.POST("/rollback", api.Controllers.BatteryApi.RollbackBattery)
		batteryApi.POST("/activate", api.Controllers.BatteryApi.ActivateBattery)
		batteryApi.POST("/complete-info", api.Controllers.BatteryApi.CompleteBatteryInfo)
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
		batteryApi.GET("/params/set/logs", api.Controllers.BatteryApi.GetBatteryParamSetLogs)

		// BLE Relay（WEB -> APP -> BLE）
		batteryApi.GET("/relay/status/:id", api.Controllers.BatteryApi.GetBatteryRelayStatus)
		batteryApi.POST("/relay/command", api.Controllers.BatteryApi.SendBatteryRelayCommand)
		batteryApi.GET("/relay/command/:id", api.Controllers.BatteryApi.GetBatteryRelayCommand)

		// 历史数据（遥测 + 属性）与异步导出
		batteryApi.GET("/history/devices", api.Controllers.BatteryApi.GetBMSHistoryDeviceList)
		batteryApi.GET("/history", api.Controllers.BatteryApi.GetBMSHistoryData)
		batteryApi.POST("/history/export", api.Controllers.BatteryApi.CreateBMSHistoryExportJob)
		batteryApi.GET("/history/export/jobs/pending", api.Controllers.BatteryApi.GetBMSHistoryPendingExportJobs)
		batteryApi.GET("/history/export/download", api.Controllers.BatteryApi.DownloadBMSHistoryExport)
		batteryApi.GET("/history/export/ws", api.Controllers.BatteryApi.ServeBMSHistoryExportWS)
	}
}
