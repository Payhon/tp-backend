package api

import (
	"context"
	"strconv"

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

// CreateBatteryWarrantyRecalcJob 创建质保截止日期补偿任务
// @Summary 创建质保截止日期补偿任务
// @Description BMS 电池管理-扫描当前租户空质保截止日期并异步补偿
// @Tags 电池管理
// @Produce json
// @Success 200 {object} model.BatteryWarrantyRecalcJobCreateResp
// @Router /api/v1/battery/warranty/recalculate-jobs [post]
func (*UserWarrantyInfoApi) CreateBatteryWarrantyRecalcJob(c *gin.Context) {
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.UserWarrantyInfo.CreateBatteryWarrantyRecalcJob(context.Background(), claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBatteryWarrantyRecalcJobStatus 获取质保补偿任务状态
// @Summary 获取质保补偿任务状态
// @Description BMS 电池管理-质保截止日期补偿任务状态
// @Tags 电池管理
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} model.BatteryWarrantyRecalcJobStatusResp
// @Router /api/v1/battery/warranty/recalculate-jobs/{id} [get]
func (*UserWarrantyInfoApi) GetBatteryWarrantyRecalcJobStatus(c *gin.Context) {
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.UserWarrantyInfo.GetBatteryWarrantyRecalcJobStatus(context.Background(), c.Param("id"), claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBatteryWarrantyRecalcJobLogs 获取质保补偿任务日志
// @Summary 获取质保补偿任务日志
// @Description BMS 电池管理-质保截止日期补偿任务日志（增量拉取）
// @Tags 电池管理
// @Produce json
// @Param id path string true "任务ID"
// @Param after_id query int false "从该日志ID之后开始拉取"
// @Param limit query int false "单次拉取条数(<=500)"
// @Success 200 {object} model.BatteryWarrantyRecalcJobLogListResp
// @Router /api/v1/battery/warranty/recalculate-jobs/{id}/logs [get]
func (*UserWarrantyInfoApi) GetBatteryWarrantyRecalcJobLogs(c *gin.Context) {
	var afterID int64
	if afterIDStr := c.Query("after_id"); afterIDStr != "" {
		if v, err := strconv.ParseInt(afterIDStr, 10, 64); err == nil {
			afterID = v
		}
	}
	limit := 200
	if limitStr := c.Query("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = v
		}
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.UserWarrantyInfo.GetBatteryWarrantyRecalcJobLogs(context.Background(), c.Param("id"), afterID, limit, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
