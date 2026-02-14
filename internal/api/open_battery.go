package api

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"project/internal/model"
	"project/internal/service"
	"project/pkg/errcode"
	"project/pkg/utils"
)

// OpenBatteryApi 第三方 MES 电池接入 API
type OpenBatteryApi struct{}

// CreateBattery 第三方新增电池
// @Summary 第三方新增电池
// @Description 通过 x-app-id / x-secret-key 鉴权，自动按密钥租户写入电池数据
// @Tags OpenAPI-MES
// @Accept json
// @Produce json
// @Param x-app-id header string true "AppId"
// @Param x-secret-key header string true "SecretKey"
// @Param body body model.BatteryCreateReq true "电池信息"
// @Success 200 {object} model.BatteryCreateResp
// @Router /api/v1/openapi/mes/battery [post]
func (*OpenBatteryApi) CreateBattery(c *gin.Context) {
	var req model.BatteryCreateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.Battery.CreateSingleBattery(context.Background(), req, userClaims, "")
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// GetBatteryBySerial 根据电池序列号查询电池信息
// @Summary 根据序列号查询电池信息
// @Description 通过 x-app-id / x-secret-key 鉴权，返回当前租户下该序列号的电池详情
// @Tags OpenAPI-MES
// @Accept json
// @Produce json
// @Param x-app-id header string true "AppId"
// @Param x-secret-key header string true "SecretKey"
// @Param serial_number path string true "电池序列号"
// @Success 200 {object} model.BatteryListItemResp
// @Router /api/v1/openapi/mes/battery/{serial_number} [get]
func (*OpenBatteryApi) GetBatteryBySerial(c *gin.Context) {
	serialNumber := strings.TrimSpace(c.Param("serial_number"))
	if serialNumber == "" {
		c.Error(errcode.NewWithMessage(errcode.CodeParamError, "serial_number is required"))
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.Battery.GetBatteryByDeviceNumber(context.Background(), serialNumber, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}
