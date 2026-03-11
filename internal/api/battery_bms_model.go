package api

import (
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type BatteryBmsModelApi struct{}

// CreateBatteryBmsModel 创建 BMS 型号
// @Router /api/v1/battery/bms-model [post]
func (*BatteryBmsModelApi) CreateBatteryBmsModel(c *gin.Context) {
	var req model.BatteryBmsModelCreateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.BatteryBmsModel.CreateBatteryBmsModel(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// UpdateBatteryBmsModel 更新 BMS 型号
// @Router /api/v1/battery/bms-model/{id} [put]
func (*BatteryBmsModelApi) UpdateBatteryBmsModel(c *gin.Context) {
	id := c.Param("id")
	var req model.BatteryBmsModelUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.BatteryBmsModel.UpdateBatteryBmsModel(id, req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// DeleteBatteryBmsModel 删除 BMS 型号
// @Router /api/v1/battery/bms-model/{id} [delete]
func (*BatteryBmsModelApi) DeleteBatteryBmsModel(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	if err := service.GroupApp.BatteryBmsModel.DeleteBatteryBmsModel(id, userClaims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// GetBatteryBmsModelByID 获取 BMS 型号详情
// @Router /api/v1/battery/bms-model/{id} [get]
func (*BatteryBmsModelApi) GetBatteryBmsModelByID(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	data, err := service.GroupApp.BatteryBmsModel.GetBatteryBmsModelByID(id, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBatteryBmsModelList 获取 BMS 型号列表
// @Router /api/v1/battery/bms-model [get]
func (*BatteryBmsModelApi) GetBatteryBmsModelList(c *gin.Context) {
	var req model.BatteryBmsModelListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.BatteryBmsModel.GetBatteryBmsModelList(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
