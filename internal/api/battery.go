package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/errcode"
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

// ExportBatteryList 导出电池列表
// @Summary 导出电池列表
// @Description BMS 电池管理-导出电池列表（Excel）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param device_number query string false "设备编号(序列号)"
// @Param battery_model_id query string false "电池型号ID"
// @Param is_online query int false "在线状态(1在线/0离线)"
// @Param activation_status query string false "激活状态(ACTIVE/INACTIVE)"
// @Param dealer_id query string false "经销商ID"
// @Param production_date_start query string false "出厂日期开始(YYYY-MM-DD)"
// @Param production_date_end query string false "出厂日期结束(YYYY-MM-DD)"
// @Param warranty_status query string false "质保状态(IN在保/OVER过保)"
// @Success 200 {file} file
// @Router /api/v1/battery/export [get]
func (*BatteryApi) ExportBatteryList(c *gin.Context) {
	var req model.BatteryExportReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	filePath, err := service.GroupApp.Battery.ExportBatteryList(context.Background(), req, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}

	// 返回文件
	c.File(filePath)
}

// GetBatteryImportTemplate 获取导入模板
// @Summary 获取导入模板
// @Description BMS 电池管理-获取导入模板（Excel）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Success 200 {file} file
// @Router /api/v1/battery/import/template [get]
func (*BatteryApi) GetBatteryImportTemplate(c *gin.Context) {
	filePath, err := service.GroupApp.Battery.GetBatteryImportTemplate()
	if err != nil {
		c.Error(err)
		return
	}

	c.File(filePath)
}

// ImportBatteryList 导入电池列表
// @Summary 导入电池列表
// @Description BMS 电池管理-导入电池列表（Excel）
// @Tags 电池管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel文件"
// @Success 200 {object} model.BatteryImportResp
// @Router /api/v1/battery/import [post]
func (*BatteryApi) ImportBatteryList(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		c.Error(errcode.New(errcode.CodeFileEmpty))
		return
	}

	// 验证文件类型
	ext := filepath.Ext(file.Filename)
	if ext != ".xlsx" && ext != ".xls" {
		c.Error(errcode.WithVars(errcode.CodeFileTypeMismatch, map[string]interface{}{
			"expected_type": ".xlsx, .xls",
			"actual_type":   ext,
		}))
		return
	}

	// 保存上传的文件
	uploadDir := "./files/upload/"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		c.Error(errcode.WithVars(errcode.CodeFilePathGenError, map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	filePath := filepath.Join(uploadDir, fmt.Sprintf("battery_import_%d%s", file.Size, ext))
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.Error(errcode.WithVars(errcode.CodeFileSaveError, map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// 处理导入
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.Battery.ImportBatteryList(context.Background(), filePath, userClaims, dealerID)
	if err != nil {
		// 清理上传的文件
		os.Remove(filePath)
		c.Error(err)
		return
	}

	// 清理上传的文件
	defer os.Remove(filePath)

	c.Set("data", data)
}

// BatchAssignDealer 批量分配经销商
// @Summary 批量分配经销商
// @Description BMS 电池管理-批量分配经销商
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryBatchAssignDealerReq true "请求参数"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/battery/batch-assign-dealer [post]
func (*BatteryApi) BatchAssignDealer(c *gin.Context) {
	var req model.BatteryBatchAssignDealerReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	err := service.GroupApp.Battery.BatchAssignDealer(context.Background(), req, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", map[string]interface{}{
		"message": "批量分配成功",
	})
}
