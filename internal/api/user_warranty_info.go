package api

import (
	"context"

	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type UserWarrantyInfoApi struct{}

// GetAppWarrantyProfile 获取 APP 端质保信息
// @Summary 获取 APP 端质保信息
// @Tags APP-Warranty
// @Produce json
// @Param appid query string false "微信小程序 AppID"
// @Success 200 {object} model.AppWarrantyProfileResp
// @Router /api/v1/app/warranty/profile [get]
func (*UserWarrantyInfoApi) GetAppWarrantyProfile(c *gin.Context) {
	var req model.AppWarrantyProfileQueryReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	appid := ""
	if req.AppID != nil {
		appid = *req.AppID
	}
	data, err := service.GroupApp.UserWarrantyInfo.GetProfile(c.Request.Context(), claims, appid)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// SaveAppWarrantyProfile 保存 APP 端质保信息
// @Summary 保存 APP 端质保信息
// @Tags APP-Warranty
// @Accept json
// @Produce json
// @Param appid query string false "微信小程序 AppID"
// @Param body body model.AppWarrantyProfileSaveReq true "质保联系人"
// @Success 200 {object} model.AppWarrantyProfileResp
// @Router /api/v1/app/warranty/profile [post]
func (*UserWarrantyInfoApi) SaveAppWarrantyProfile(c *gin.Context) {
	var req model.AppWarrantyProfileSaveReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.UserWarrantyInfo.SaveProfile(c.Request.Context(), claims, c.Query("appid"), &req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBatteryWarranty 获取后台电池质保信息
// @Summary 获取后台电池质保信息
// @Tags 电池管理
// @Produce json
// @Param device_id path string true "设备ID"
// @Success 200 {object} model.BatteryWarrantyResp
// @Router /api/v1/battery/{device_id}/warranty [get]
func (*UserWarrantyInfoApi) GetBatteryWarranty(c *gin.Context) {
	claims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)
	data, err := service.GroupApp.UserWarrantyInfo.GetBatteryWarranty(context.Background(), c.Param("device_id"), claims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// UpdateBatteryWarranty 更新后台电池质保信息
// @Summary 更新后台电池质保信息
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID"
// @Param body body model.BatteryWarrantyUpdateReq true "质保信息"
// @Success 200 {object} model.BatteryWarrantyResp
// @Router /api/v1/battery/{device_id}/warranty [put]
func (*UserWarrantyInfoApi) UpdateBatteryWarranty(c *gin.Context) {
	var req model.BatteryWarrantyUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)
	data, err := service.GroupApp.UserWarrantyInfo.UpdateBatteryWarranty(context.Background(), c.Param("device_id"), &req, claims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
