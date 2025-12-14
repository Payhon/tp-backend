package api

import (
	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/errcode"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BatteryMaintenanceApi 电池维保记录
type BatteryMaintenanceApi struct{}

// CreateBatteryMaintenance 创建电池维保记录
// @Router /api/v1/battery_maintenance [post]
func (*BatteryMaintenanceApi) CreateBatteryMaintenance(c *gin.Context) {
	var req model.BatteryMaintenanceCreateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerScopeID, _ := dealerIDVal.(string)

	if err := service.GroupApp.BatteryMaintenance.Create(c, req, userClaims, dealerScopeID); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// ListBatteryMaintenance 电池维保记录列表
// @Router /api/v1/battery_maintenance [get]
func (*BatteryMaintenanceApi) ListBatteryMaintenance(c *gin.Context) {
	var req model.BatteryMaintenanceListReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerScopeID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.BatteryMaintenance.List(c, req, userClaims, dealerScopeID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBatteryMaintenanceDetail 电池维保记录详情
// @Router /api/v1/battery_maintenance/{id} [get]
func (*BatteryMaintenanceApi) GetBatteryMaintenanceDetail(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerScopeID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.BatteryMaintenance.Detail(c, id, userClaims, dealerScopeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Error(errcode.New(404))
			return
		}
		c.Error(err)
		return
	}
	c.Set("data", data)
}

