package api

import (
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// DeviceProvisionApi 移动端设备开通（扫码/蓝牙绑定）
type DeviceProvisionApi struct{}

// GetProvisionConfig 获取移动端开通配置（DTU 域名端口）
// @Summary 获取移动端开通配置
// @Tags APP-Device
// @Produce json
// @Success 200 {object} model.DeviceProvisionConfigResp
// @Router /api/v1/app/device/provision/config [get]
func (*DeviceProvisionApi) GetProvisionConfig(c *gin.Context) {
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.DeviceProvision.GetProvisionConfig(c.Request.Context(), claims.TenantID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetProvisionInfo 扫码 UUID 查询设备信息
// @Summary 扫码UUID查询设备信息
// @Tags APP-Device
// @Produce json
// @Param item_uuid query string true "设备唯一编号(item_uuid)"
// @Success 200 {object} model.DeviceProvisionInfoResp
// @Router /api/v1/app/device/provision/info [get]
func (*DeviceProvisionApi) GetProvisionInfo(c *gin.Context) {
	var req model.DeviceProvisionInfoReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.DeviceProvision.GetProvisionInfo(c.Request.Context(), req, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// BindByItemUUID 按 item_uuid 绑定设备到当前账号
// @Summary 按item_uuid绑定设备
// @Tags APP-Device
// @Accept json
// @Produce json
// @Param body body model.DeviceProvisionBindReq true "绑定请求"
// @Success 200 {object} model.DeviceProvisionBindResp
// @Router /api/v1/app/device/provision/bind [post]
func (*DeviceProvisionApi) BindByItemUUID(c *gin.Context) {
	var req model.DeviceProvisionBindReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.DeviceProvision.BindByItemUUID(c.Request.Context(), req, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
