package api

import (
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type BatteryModelApi struct{}

// CreateBatteryModel 创建电池型号
// @Summary 创建电池型号
// @Description 创建新的电池型号
// @Tags 电池型号管理
// @Accept json
// @Produce json
// @Param body body model.BatteryModelCreateReq true "电池型号信息"
// @Success 200 {object} model.BatteryModel
// @Router /api/v1/battery/model [post]
func (*BatteryModelApi) CreateBatteryModel(c *gin.Context) {
	var req model.BatteryModelCreateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.BatteryModel.CreateBatteryModel(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// UpdateBatteryModel 更新电池型号
// @Summary 更新电池型号
// @Description 更新电池型号信息
// @Tags 电池型号管理
// @Accept json
// @Produce json
// @Param id path string true "电池型号ID"
// @Param body body model.BatteryModelUpdateReq true "电池型号信息"
// @Success 200 {object} model.BatteryModel
// @Router /api/v1/battery/model/{id} [put]
func (*BatteryModelApi) UpdateBatteryModel(c *gin.Context) {
	id := c.Param("id")
	var req model.BatteryModelUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.BatteryModel.UpdateBatteryModel(id, req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// DeleteBatteryModel 删除电池型号
// @Summary 删除电池型号
// @Description 删除电池型号（需确保无关联设备）
// @Tags 电池型号管理
// @Accept json
// @Produce json
// @Param id path string true "电池型号ID"
// @Success 200 {object} model.Response
// @Router /api/v1/battery/model/{id} [delete]
func (*BatteryModelApi) DeleteBatteryModel(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	
	err := service.GroupApp.BatteryModel.DeleteBatteryModel(id, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", nil)
}

// GetBatteryModelByID 获取电池型号详情
// @Summary 获取电池型号详情
// @Description 根据ID获取电池型号详细信息
// @Tags 电池型号管理
// @Accept json
// @Produce json
// @Param id path string true "电池型号ID"
// @Success 200 {object} model.BatteryModelResp
// @Router /api/v1/battery/model/{id} [get]
func (*BatteryModelApi) GetBatteryModelByID(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	
	data, err := service.GroupApp.BatteryModel.GetBatteryModelByID(id, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// GetBatteryModelList 获取电池型号列表
// @Summary 获取电池型号列表
// @Description 分页查询电池型号列表
// @Tags 电池型号管理
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param name query string false "型号名称"
// @Success 200 {object} model.BatteryModelListResp
// @Router /api/v1/battery/model [get]
func (*BatteryModelApi) GetBatteryModelList(c *gin.Context) {
	var req model.BatteryModelListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.BatteryModel.GetBatteryModelList(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}
