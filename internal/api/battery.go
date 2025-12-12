package api

import (
	"context"

	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// BatteryApi BMS: 电池管理
type BatteryApi struct{}

// GetBatteryList 获取电池列表
// @Summary 获取电池列表
// @Description BMS 电池管理-电池列表（支持厂家/经销商视角数据隔离）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param device_number query string false "设备编号(序列号)"
// @Param battery_model_id query string false "电池型号ID"
// @Param is_online query int false "在线状态(1在线/0离线)"
// @Param activation_status query string false "激活状态(ACTIVE/INACTIVE)"
// @Param dealer_id query string false "经销商ID"
// @Param production_date_start query string false "出厂日期开始(YYYY-MM-DD)"
// @Param production_date_end query string false "出厂日期结束(YYYY-MM-DD)"
// @Param warranty_status query string false "质保状态(IN在保/OVER过保)"
// @Success 200 {object} model.BatteryListResp
// @Router /api/v1/battery [get]
func (*BatteryApi) GetBatteryList(c *gin.Context) {
	var req model.BatteryListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)

	// 经销商上下文（由 DealerAuthMiddleware 注入）
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.Battery.GetBatteryList(context.Background(), req, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}
