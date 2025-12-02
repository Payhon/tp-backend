package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type DeviceTransfer struct{}

func (*DeviceTransfer) InitDeviceTransfer(Router *gin.RouterGroup) {
	// 设备转移路由
	transferApi := Router.Group("device/transfer")
	{
		// 批量转移设备
		transferApi.POST("", api.Controllers.DeviceTransferApi.TransferDevices)

		// 转移记录查询
		transferApi.GET("history", api.Controllers.DeviceTransferApi.GetTransferHistory)
	}
}
