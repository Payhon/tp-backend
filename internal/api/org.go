package api

import (
	"context"

	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type OrgApi struct{}

// CreateOrg 创建组织
// @Summary 创建组织
// @Description 创建新的组织（BMS厂家/PACK厂家/经销商/门店）
// @Tags 组织管理
// @Accept json
// @Produce json
// @Param body body model.OrgCreateReq true "组织信息"
// @Success 200 {object} model.Org
// @Router /api/v1/org [post]
func (*OrgApi) CreateOrg(c *gin.Context) {
	var req model.OrgCreateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.OrgService.CreateOrg(context.Background(), &req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// UpdateOrg 更新组织
// @Summary 更新组织
// @Description 更新组织信息（不支持修改上级组织）
// @Tags 组织管理
// @Accept json
// @Produce json
// @Param id path string true "组织ID"
// @Param body body model.OrgUpdateReq true "组织信息"
// @Success 200 {object} model.Response
// @Router /api/v1/org/{id} [put]
func (*OrgApi) UpdateOrg(c *gin.Context) {
	id := c.Param("id")
	var req model.OrgUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	err := service.GroupApp.OrgService.UpdateOrg(context.Background(), id, &req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", nil)
}

// DeleteOrg 删除组织
// @Summary 删除组织
// @Description 删除组织（需确保无子组织和关联设备）
// @Tags 组织管理
// @Accept json
// @Produce json
// @Param id path string true "组织ID"
// @Success 200 {object} model.Response
// @Router /api/v1/org/{id} [delete]
func (*OrgApi) DeleteOrg(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	err := service.GroupApp.OrgService.DeleteOrg(context.Background(), id, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", nil)
}

// GetOrgByID 获取组织详情
// @Summary 获取组织详情
// @Description 根据ID获取组织详细信息
// @Tags 组织管理
// @Accept json
// @Produce json
// @Param id path string true "组织ID"
// @Success 200 {object} model.Org
// @Router /api/v1/org/{id} [get]
func (*OrgApi) GetOrgByID(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)

	data, err := service.GroupApp.OrgService.GetOrgByID(context.Background(), id, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// GetOrgList 获取组织列表
// @Summary 获取组织列表
// @Description 分页查询组织列表，支持按类型筛选
// @Tags 组织管理
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param org_type query string false "组织类型: BMS_FACTORY, PACK_FACTORY, DEALER, STORE"
// @Param name query string false "组织名称"
// @Param status query string false "状态: N-正常, F-禁用"
// @Param parent_id query string false "父组织ID"
// @Success 200 {object} model.OrgListResp
// @Router /api/v1/org [get]
func (*OrgApi) GetOrgList(c *gin.Context) {
	var req model.OrgListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.OrgService.GetOrgList(context.Background(), &req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// GetOrgTree 获取组织树
// @Summary 获取组织树
// @Description 获取组织树结构，可按类型筛选
// @Tags 组织管理
// @Accept json
// @Produce json
// @Param org_type query string false "组织类型筛选"
// @Success 200 {object} []model.OrgTreeNode
// @Router /api/v1/org/tree [get]
func (*OrgApi) GetOrgTree(c *gin.Context) {
	var req model.OrgTreeReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.OrgService.GetOrgTree(context.Background(), userClaims, req.OrgType)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}
