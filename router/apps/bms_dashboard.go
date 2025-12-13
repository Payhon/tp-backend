package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

// BmsDashboard BMS: Dashboard 路由
type BmsDashboard struct{}

func (*BmsDashboard) InitBmsDashboard(Router *gin.RouterGroup) {
	dashboardApi := Router.Group("dashboard")
	{
		dashboardApi.GET("/kpi", api.Controllers.BmsDashboardApi.GetKpi)
		dashboardApi.GET("/alarm/overview", api.Controllers.BmsDashboardApi.GetAlarmOverview)
		dashboardApi.GET("/trend/online", api.Controllers.BmsDashboardApi.GetOnlineTrend)
	}
}
