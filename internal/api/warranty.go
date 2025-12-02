package api

import (
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// WarrantyApi 维保管理接口
type WarrantyApi struct{}

// CreateWarrantyApplication 创建维保申请
// @Summary 创建维保申请
// @Description 终端用户或经销商创建维保/售后申请
// @Tags Warranty
// @Accept json
// @Produce json
// @Param body body model.WarrantyApplicationCreateReq true "维保申请信息"
// @Success 200 {object} model.WarrantyApplicationResp
// @Router /api/v1/warranty [post]
func (*WarrantyApi) CreateWarrantyApplication(c *gin.Context) {
	var req model.WarrantyApplicationCreateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.Warranty.CreateWarrantyApplication(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// UpdateWarrantyStatus 更新维保申请状态
// @Summary 更新维保申请状态
// @Description 审核/处理维保申请，更新状态与处理结果
// @Tags Warranty
// @Accept json
// @Produce json
// @Param id path string true "维保申请ID"
// @Param body body model.WarrantyApplicationUpdateReq true "更新内容"
// @Success 200 {object} model.Response
// @Router /api/v1/warranty/{id} [put]
func (*WarrantyApi) UpdateWarrantyStatus(c *gin.Context) {
	id := c.Param("id")
	var req model.WarrantyApplicationUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.Warranty.UpdateWarrantyStatus(id, req, userClaims); err != nil {
		c.Error(err)
		return
	}

	c.Set("data", nil)
}

// GetWarrantyList 获取维保申请列表
// @Summary 获取维保申请列表
// @Description 分页查询维保/售后申请列表
// @Tags Warranty
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param device_number query string false "设备编号"
// @Param user_id query string false "申请人用户ID"
// @Param type query string false "类型(REPAIR/RETURN/EXCHANGE)"
// @Param status query string false "状态(PENDING/APPROVED/REJECTED/PROCESSING/COMPLETED)"
// @Param start_time query string false "开始时间(yyyy-MM-dd HH:mm:ss)"
// @Param end_time query string false "结束时间(yyyy-MM-dd HH:mm:ss)"
// @Success 200 {object} model.WarrantyApplicationListResp
// @Router /api/v1/warranty [get]
func (*WarrantyApi) GetWarrantyList(c *gin.Context) {
	var req model.WarrantyApplicationListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.Warranty.GetWarrantyList(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// GetWarrantyDetail 获取维保申请详情
// @Summary 获取维保申请详情
// @Description 根据ID获取维保申请完整信息
// @Tags Warranty
// @Accept json
// @Produce json
// @Param id path string true "维保申请ID"
// @Success 200 {object} model.WarrantyApplicationResp
// @Router /api/v1/warranty/{id} [get]
func (*WarrantyApi) GetWarrantyDetail(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	data, err := service.GroupApp.Warranty.GetWarrantyDetail(id, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

