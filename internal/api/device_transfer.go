package api

import (
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type DeviceTransferApi struct{}

// TransferDevices 批量转移设备
// @Summary 批量转移设备
// @Description 将设备从一个经销商转移到另一个经销商，或转移回厂家
// @Tags 设备转移
// @Accept json
// @Produce json
// @Param body body model.DeviceTransferReq true "转移请求"
// @Success 200 {object} model.Response
// @Router /api/v1/device/transfer [post]
func (*DeviceTransferApi) TransferDevices(c *gin.Context) {
	var req model.DeviceTransferReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	err := service.GroupApp.DeviceTransfer.TransferDevices(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", map[string]interface{}{
		"message": "transfer success",
	})
}

// GetTransferHistory 获取设备转移记录
// @Summary 获取设备转移记录
// @Description 分页查询设备转移历史记录
// @Tags 设备转移
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param device_number query string false "设备编号"
// @Param from_dealer_id query string false "原经销商ID"
// @Param to_dealer_id query string false "目标经销商ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} model.DeviceTransferListResp
// @Router /api/v1/device/transfer/history [get]
func (*DeviceTransferApi) GetTransferHistory(c *gin.Context) {
	var req model.DeviceTransferListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.DeviceTransfer.GetTransferHistory(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}
