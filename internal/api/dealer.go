package api

import (
	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type DealerApi struct{}

// CreateDealer 创建经销商
// @Summary 创建经销商
// @Description 创建新的经销商
// @Tags 经销商管理
// @Accept json
// @Produce json
// @Param body body model.DealerCreateReq true "经销商信息"
// @Success 200 {object} model.Dealer
// @Router /api/v1/dealer [post]
func (*DealerApi) CreateDealer(c *gin.Context) {
	var req model.DealerCreateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.Dealer.CreateDealer(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// UpdateDealer 更新经销商
// @Summary 更新经销商
// @Description 更新经销商信息
// @Tags 经销商管理
// @Accept json
// @Produce json
// @Param id path string true "经销商ID"
// @Param body body model.DealerUpdateReq true "经销商信息"
// @Success 200 {object} model.Dealer
// @Router /api/v1/dealer/{id} [put]
func (*DealerApi) UpdateDealer(c *gin.Context) {
	id := c.Param("id")
	var req model.DealerUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.Dealer.UpdateDealer(id, req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// DeleteDealer 删除经销商
// @Summary 删除经销商
// @Description 删除经销商（需确保无下级经销商和关联设备）
// @Tags 经销商管理
// @Accept json
// @Produce json
// @Param id path string true "经销商ID"
// @Success 200 {object} model.Response
// @Router /api/v1/dealer/{id} [delete]
func (*DealerApi) DeleteDealer(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	err := service.GroupApp.Dealer.DeleteDealer(id, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", nil)
}

// GetDealerByID 获取经销商详情
// @Summary 获取经销商详情
// @Description 根据ID获取经销商详细信息
// @Tags 经销商管理
// @Accept json
// @Produce json
// @Param id path string true "经销商ID"
// @Success 200 {object} model.DealerResp
// @Router /api/v1/dealer/{id} [get]
func (*DealerApi) GetDealerByID(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	data, err := service.GroupApp.Dealer.GetDealerByID(id, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// GetDealerList 获取经销商列表
// @Summary 获取经销商列表
// @Description 分页查询经销商列表
// @Tags 经销商管理
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param name query string false "经销商名称"
// @Param phone query string false "联系电话"
// @Param province query string false "省份"
// @Param city query string false "城市"
// @Success 200 {object} model.DealerListResp
// @Router /api/v1/dealer [get]
func (*DealerApi) GetDealerList(c *gin.Context) {
	var req model.DealerListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.Dealer.GetDealerList(req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// GetDealerOverview 获取经销商穿透概览
// @Summary 获取经销商穿透概览
// @Description 经销商详情聚合数字（设备/终端用户/维保）
// @Tags 经销商管理
// @Accept json
// @Produce json
// @Param id path string true "经销商ID"
// @Success 200 {object} model.DealerOverviewResp
// @Router /api/v1/dealer/{id}/overview [get]
func (*DealerApi) GetDealerOverview(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerScopeID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.Dealer.GetDealerOverview(c, id, userClaims, dealerScopeID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetDealerPermissionTemplate 获取经销商权限模板
// @Summary 获取经销商权限模板
// @Tags 经销商管理
// @Produce json
// @Param id path string true "经销商ID"
// @Success 200 {object} model.DealerPermissionTemplateResp
// @Router /api/v1/dealer/{id}/permission_template [get]
func (*DealerApi) GetDealerPermissionTemplate(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerScopeID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.Dealer.GetDealerPermissionTemplate(c, id, userClaims, dealerScopeID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// SetDealerPermissionTemplate 设置经销商权限模板
// @Summary 设置经销商权限模板
// @Tags 经销商管理
// @Accept json
// @Produce json
// @Param id path string true "经销商ID"
// @Param body body model.DealerPermissionTemplateReq true "模板"
// @Success 200 {object} model.DealerPermissionTemplateResp
// @Router /api/v1/dealer/{id}/permission_template [put]
func (*DealerApi) SetDealerPermissionTemplate(c *gin.Context) {
	id := c.Param("id")
	var req model.DealerPermissionTemplateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerScopeID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.Dealer.SetDealerPermissionTemplate(c, id, req, userClaims, dealerScopeID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
