package api

import (
	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type OfflineCommandApi struct{}

// CreateOfflineCommand 创建离线指令
// @Router /api/v1/battery/offline-commands [post]
func (*OfflineCommandApi) CreateOfflineCommand(c *gin.Context) {
	var req model.OfflineCommandCreateReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.OfflineCommand.Create(c, req, userClaims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// ListOfflineCommands 离线指令列表
// @Router /api/v1/battery/offline-commands [get]
func (*OfflineCommandApi) ListOfflineCommands(c *gin.Context) {
	var req model.OfflineCommandListReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.OfflineCommand.List(c, req, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetOfflineCommandDetail 离线指令详情
// @Router /api/v1/battery/offline-commands/{id} [get]
func (*OfflineCommandApi) GetOfflineCommandDetail(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.OfflineCommand.Detail(c, id, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// CancelOfflineCommand 撤销未执行的离线指令
// @Router /api/v1/battery/offline-commands/{id} [delete]
func (*OfflineCommandApi) CancelOfflineCommand(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	if err := service.GroupApp.OfflineCommand.Cancel(c, id, userClaims, dealerID); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

