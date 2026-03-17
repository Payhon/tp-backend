package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

// DeviceBinding APP端设备绑定路由
type DeviceBinding struct{}

func (*DeviceBinding) InitDeviceBinding(Router *gin.RouterGroup) {
	appDeviceApi := Router.Group("app/device")
	{
		appDeviceApi.POST("bind", api.Controllers.DeviceBindingApi.BindDevice)
		appDeviceApi.POST("unbind", api.Controllers.DeviceBindingApi.UnbindDevice)
		appDeviceApi.POST("remove", api.Controllers.DeviceBindingApi.RemoveOrgAddedDevice)
		appDeviceApi.GET("list", api.Controllers.DeviceBindingApi.GetUserDevices)
		appDeviceApi.GET("org/list", api.Controllers.DeviceBindingApi.GetOrgDevices)

		// 移动端：扫码/蓝牙开通绑定（item_uuid -> bind）
		provision := appDeviceApi.Group("provision")
		{
			provision.GET("config", api.Controllers.DeviceProvisionApi.GetProvisionConfig)
			provision.GET("info", api.Controllers.DeviceProvisionApi.GetProvisionInfo)
			provision.POST("bind", api.Controllers.DeviceProvisionApi.BindByItemUUID)
		}
	}

	appOrgApi := Router.Group("app/org")
	{
		appOrgApi.GET("options", api.Controllers.DeviceBindingApi.GetOrgOptions)
	}
}
