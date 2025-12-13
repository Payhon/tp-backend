package api

import (
	"strconv"

	middleware "project/internal/middleware"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// BmsDashboardApi BMS: Dashboard
type BmsDashboardApi struct{}

// GetKpi BMS Dashboard 指标卡
// @Summary BMS Dashboard 指标卡
// @Tags BMS-Dashboard
// @Produce json
// @Success 200 {object} model.BmsDashboardKpiResp
// @Router /api/v1/dashboard/kpi [get]
func (*BmsDashboardApi) GetKpi(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.BmsDashboard.GetKpi(c, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetAlarmOverview BMS Dashboard 告警概览
// @Summary BMS Dashboard 告警概览
// @Tags BMS-Dashboard
// @Produce json
// @Param days query int false "近N天(默认7)"
// @Success 200 {object} model.BmsDashboardAlarmOverviewResp
// @Router /api/v1/dashboard/alarm/overview [get]
func (*BmsDashboardApi) GetAlarmOverview(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	days := 7
	if s := c.Query("days"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 90 {
			days = v
		}
	}

	data, err := service.GroupApp.BmsDashboard.GetAlarmOverview(c, userClaims, dealerID, days)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetOnlineTrend BMS Dashboard 在线趋势
// @Summary BMS Dashboard 在线趋势
// @Tags BMS-Dashboard
// @Produce json
// @Success 200 {object} model.BmsDashboardOnlineTrendResp
// @Router /api/v1/dashboard/trend/online [get]
func (*BmsDashboardApi) GetOnlineTrend(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.BmsDashboard.GetOnlineTrend(c, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
