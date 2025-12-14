package api

import (
	"context"

	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// GetBatteryParams 参数远程查看（BMS，带经销商隔离）
// @Router /api/v1/battery/params/{id} [get]
func (*BatteryApi) GetBatteryParams(c *gin.Context) {
	deviceID := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.Battery.GetDeviceAttributes(context.Background(), deviceID, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// PutBatteryParams 参数远程修改（BMS，带经销商隔离）
// @Router /api/v1/battery/params/pub [post]
func (*BatteryApi) PutBatteryParams(c *gin.Context) {
	var req model.AttributePutMessage
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	err := service.GroupApp.Battery.PutDeviceAttributes(context.Background(), req, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// GetBatteryParamsFromDevice 请求设备上报参数（BMS，带经销商隔离）
// @Router /api/v1/battery/params/get [post]
func (*BatteryApi) GetBatteryParamsFromDevice(c *gin.Context) {
	var req model.AttributeGetMessageReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	err := service.GroupApp.Battery.RequestDeviceAttributes(context.Background(), req, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

