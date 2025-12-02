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
		appDeviceApi.GET("list", api.Controllers.DeviceBindingApi.GetUserDevices)
	}
}

