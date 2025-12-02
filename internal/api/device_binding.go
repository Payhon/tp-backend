package api

import (
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// DeviceBindingApi APP端设备绑定相关接口
type DeviceBindingApi struct{}

// BindDevice 绑定设备
// @Summary APP绑定设备
// @Description 终端用户通过设备编号与可选密钥绑定设备
// @Tags APP-Device
// @Accept json
// @Produce json
// @Param body body model.DeviceBindReq true "绑定请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/device/bind [post]
func (*DeviceBindingApi) BindDevice(c *gin.Context) {
	var req model.DeviceBindReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.DeviceBinding.BindDevice(req, userClaims); err != nil {
		c.Error(err)
		return
	}

	c.Set("data", map[string]interface{}{
		"message": "bind success",
	})
}

// UnbindDevice 解绑设备
// @Summary APP解绑设备
// @Description 终端用户解绑已绑定的设备
// @Tags APP-Device
// @Accept json
// @Produce json
// @Param body body model.DeviceUnbindReq true "解绑请求"
// @Success 200 {object} model.Response
// @Router /api/v1/app/device/unbind [post]
func (*DeviceBindingApi) UnbindDevice(c *gin.Context) {
	var req model.DeviceUnbindReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.DeviceBinding.UnbindDevice(req, userClaims); err != nil {
		c.Error(err)
		return
	}

	c.Set("data", nil)
}

// GetUserDevices 获取当前用户绑定设备列表
// @Summary 获取用户绑定设备列表
// @Description 分页查询用户绑定的设备列表（默认当前登录用户）
// @Tags APP-Device
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param user_id query string false "用户ID（可选，不传则为当前用户）"
// @Param device_number query string false "设备编号"
// @Success 200 {object} model.DeviceUserBindingListResp
// @Router /api/v1/app/device/list [get]
func (*DeviceBindingApi) GetUserDevices(c *gin.Context) {
	var req model.DeviceUserBindingListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.DeviceBinding.GetUserDevices(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

