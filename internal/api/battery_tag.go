package api

import (
	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type BatteryTagApi struct{}

// CreateBatteryTag 新增标签
// @Router /api/v1/battery/tags [post]
func (*BatteryTagApi) CreateBatteryTag(c *gin.Context) {
	var req model.BatteryTagCreateReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.BatteryTag.Create(c, req, userClaims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// UpdateBatteryTag 更新标签
// @Router /api/v1/battery/tags/{id} [put]
func (*BatteryTagApi) UpdateBatteryTag(c *gin.Context) {
	id := c.Param("id")
	var req model.BatteryTagUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.BatteryTag.Update(c, id, req, userClaims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// DeleteBatteryTag 删除标签
// @Router /api/v1/battery/tags/{id} [delete]
func (*BatteryTagApi) DeleteBatteryTag(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.BatteryTag.Delete(c, id, userClaims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// ListBatteryTags 标签列表
// @Router /api/v1/battery/tags [get]
func (*BatteryTagApi) ListBatteryTags(c *gin.Context) {
	var req model.BatteryTagListReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.BatteryTag.List(c, req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// AssignBatteryTags 批量设置设备标签
// @Router /api/v1/battery/tags/assign [post]
func (*BatteryTagApi) AssignBatteryTags(c *gin.Context) {
	var req model.BatteryTagAssignReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerScopeID, _ := dealerIDVal.(string)

	if err := service.GroupApp.BatteryTag.Assign(c, req, userClaims, dealerScopeID); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

