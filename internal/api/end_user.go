package api

import (
	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// EndUserApi BMS: 终端用户
type EndUserApi struct{}

// GetEndUserList 终端用户列表
// @Summary 终端用户列表
// @Tags BMS-EndUser
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param dealer_id query string false "经销商ID（厂家侧可选）"
// @Param phone query string false "手机号"
// @Param device_number query string false "设备编号"
// @Success 200 {object} model.EndUserListResp
// @Router /api/v1/end_user [get]
func (*EndUserApi) GetEndUserList(c *gin.Context) {
	var req model.EndUserListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.EndUser.GetEndUserList(c, req, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetEndUserDevices 用户绑定设备列表（穿透查看）
// @Summary 用户绑定设备列表
// @Tags BMS-EndUser
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param user_id query string true "用户ID"
// @Success 200 {object} model.EndUserDeviceListResp
// @Router /api/v1/end_user/devices [get]
func (*EndUserApi) GetEndUserDevices(c *gin.Context) {
	var req model.EndUserDeviceListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.EndUser.GetEndUserDevices(c, req, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// ForceUnbind 强制解绑
// @Summary 强制解绑
// @Tags BMS-EndUser
// @Accept json
// @Produce json
// @Param body body model.EndUserForceUnbindReq true "解绑请求"
// @Success 200 {object} model.Response
// @Router /api/v1/end_user/force_unbind [post]
func (*EndUserApi) ForceUnbind(c *gin.Context) {
	var req model.EndUserForceUnbindReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	if err := service.GroupApp.EndUser.ForceUnbind(c, req, userClaims, dealerID); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}
