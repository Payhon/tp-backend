package api

import (
	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// ActivationLogApi 激活日志
type ActivationLogApi struct{}

// GetActivationLogs 激活日志分页查询
// @Router   /api/v1/activation_logs [get]
func (*ActivationLogApi) GetActivationLogs(c *gin.Context) {
	var req model.ActivationLogListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerScopeID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.ActivationLog.GetActivationLogList(c, req, userClaims, dealerScopeID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

