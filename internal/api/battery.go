package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/errcode"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// BatteryApi BMS: 电池管理
type BatteryApi struct{}

// CreateSingleBattery 添加单个电池
// @Summary 添加单个电池
// @Description BMS 电池管理-添加单个电池（item_uuid 对应 devices.device_number）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryCreateReq true "电池信息"
// @Success 200 {object} model.BatteryCreateResp
// @Router /api/v1/battery/single [post]
func (*BatteryApi) CreateSingleBattery(c *gin.Context) {
	var req model.BatteryCreateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgIDVal, _ := c.Get(middleware.DealerIDContextKey)
	orgID, _ := orgIDVal.(string)

	data, err := service.GroupApp.Battery.CreateSingleBattery(context.Background(), req, userClaims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// UpdateSingleBattery 编辑单个电池
// @Summary 编辑单个电池
// @Description BMS 电池管理-编辑 BMS 信息（对应新增 BMS 表单字段）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param body body model.BatteryCreateReq true "电池信息"
// @Success 200 {object} model.BatteryCreateResp
// @Router /api/v1/battery/single/{id} [put]
func (*BatteryApi) UpdateSingleBattery(c *gin.Context) {
	var req model.BatteryCreateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)

	data, err := service.GroupApp.Battery.UpdateSingleBattery(context.Background(), c.Param("id"), req, userClaims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// DeleteBattery 删除电池
// @Summary 删除电池
// @Description BMS 电池管理-删除电池及关联业务数据（不可恢复）
// @Tags 电池管理
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} map[string]any
// @Router /api/v1/battery/{id} [delete]
func (*BatteryApi) DeleteBattery(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)

	if err := service.GroupApp.Battery.DeleteBattery(context.Background(), c.Param("id"), userClaims, orgID); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", gin.H{"success": true})
}

// GetBatteryList 获取电池列表
// @Summary 获取电池列表
// @Description BMS 电池管理-电池列表（支持厂家/经销商视角数据隔离）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param search_field query string false "文本搜索字段(device_number/batch_number/battery_model_name/product_spec/ble_mac/comm_chip_id)"
// @Param search_value query string false "文本搜索值"
// @Param device_number query string false "设备编号(序列号)"
// @Param battery_model_id query string false "电池型号ID"
// @Param is_online query int false "在线状态(1在线/0离线)"
// @Param activation_status query string false "激活状态(ACTIVE/INACTIVE)"
// @Param owner_org_id query string false "归属组织ID"
// @Param owner_org_type query string false "归属组织类型"
// @Param dealer_id query string false "经销商ID(已废弃)"
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

	orgID := middleware.GetOrgID(c)

	data, err := service.GroupApp.Battery.GetBatteryList(context.Background(), req, userClaims, orgID)
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
// @Param search_field query string false "文本搜索字段(device_number/batch_number/battery_model_name/product_spec/ble_mac/comm_chip_id)"
// @Param search_value query string false "文本搜索值"
// @Param device_number query string false "设备编号(序列号)"
// @Param battery_model_id query string false "电池型号ID"
// @Param is_online query int false "在线状态(1在线/0离线)"
// @Param activation_status query string false "激活状态(ACTIVE/INACTIVE)"
// @Param owner_org_id query string false "归属组织ID"
// @Param owner_org_type query string false "归属组织类型"
// @Param dealer_id query string false "经销商ID(已废弃)"
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
	orgID := middleware.GetOrgID(c)

	filePath, err := service.GroupApp.Battery.ExportBatteryList(context.Background(), req, userClaims, orgID)
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
// @Description BMS 电池管理-导入电池列表（Excel，异步任务：返回 job_id 用于查询进度/日志）
// @Tags 电池管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel文件"
// @Success 200 {object} model.BatteryImportJobCreateResp
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
	data, err := service.GroupApp.Battery.CreateBatteryImportJob(context.Background(), filePath, userClaims)
	if err != nil {
		_ = os.Remove(filePath)
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// GetBatteryImportJobStatus 获取导入任务状态
// @Summary 获取导入任务状态
// @Description BMS 电池管理-导入任务状态
// @Tags 电池管理
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} model.BatteryImportJobStatusResp
// @Router /api/v1/battery/import/jobs/{id} [get]
func (*BatteryApi) GetBatteryImportJobStatus(c *gin.Context) {
	jobID := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.Battery.GetBatteryImportJobStatus(context.Background(), jobID, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBatteryImportJobLogs 获取导入任务日志
// @Summary 获取导入任务日志
// @Description BMS 电池管理-导入任务日志（增量拉取）
// @Tags 电池管理
// @Produce json
// @Param id path string true "任务ID"
// @Param after_id query int false "从该日志ID之后开始拉取"
// @Param limit query int false "单次拉取条数(<=500)"
// @Success 200 {object} model.BatteryImportJobLogListResp
// @Router /api/v1/battery/import/jobs/{id}/logs [get]
func (*BatteryApi) GetBatteryImportJobLogs(c *gin.Context) {
	jobID := c.Param("id")
	afterIDStr := c.Query("after_id")
	limitStr := c.Query("limit")
	var afterID int64
	if afterIDStr != "" {
		if v, err := strconv.ParseInt(afterIDStr, 10, 64); err == nil {
			afterID = v
		}
	}
	limit := 200
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = v
		}
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.Battery.GetBatteryImportJobLogs(context.Background(), jobID, afterID, limit, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBatteryOperationLogList 运营日志列表
// @Summary 运营日志列表
// @Description BMS 运营管理-运营日志（按电池编号查询）
// @Tags 运营管理
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param device_id query string false "设备ID(电池详情精确过滤)"
// @Param device_number query string false "电池编号(模糊)"
// @Param operation_type query string false "操作类型"
// @Param start_time query string false "开始时间(RFC3339 或 YYYY-MM-DD)"
// @Param end_time query string false "结束时间(RFC3339 或 YYYY-MM-DD)"
// @Success 200 {object} model.BatteryOperationLogListResp
// @Router /api/v1/battery/operation_logs [get]
func (*BatteryApi) GetBatteryOperationLogList(c *gin.Context) {
	var req model.BatteryOperationLogListReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgIDVal, _ := c.Get(middleware.DealerIDContextKey)
	orgID, _ := orgIDVal.(string)
	data, err := service.GroupApp.Battery.GetBatteryOperationLogList(context.Background(), req, userClaims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// FactoryOutBattery 电池出厂
// @Summary 电池出厂
// @Description BMS 电池管理-出厂（厂家 -> PACK/经销商）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryFactoryOutReq true "请求参数"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/battery/factory_out [post]
func (*BatteryApi) FactoryOutBattery(c *gin.Context) {
	var req model.BatteryFactoryOutReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)

	if err := service.GroupApp.Battery.FactoryOutBattery(context.Background(), req, userClaims, orgID); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", map[string]interface{}{"message": "出厂成功"})
}

// BatchFactoryOutBattery 批量电池出厂
// @Summary 批量电池出厂
// @Description BMS 电池管理-批量出厂（厂家 -> PACK/经销商）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryBatchFactoryOutReq true "请求参数"
// @Success 200 {object} model.BatteryBatchFactoryOutResp
// @Router /api/v1/battery/batch-factory-out [post]
func (*BatteryApi) BatchFactoryOutBattery(c *gin.Context) {
	var req model.BatteryBatchFactoryOutReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)

	data, err := service.GroupApp.Battery.BatchFactoryOutBattery(context.Background(), req, userClaims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// FactoryRestoreBattery 电池恢复出厂
// @Summary 电池恢复出厂
// @Description BMS 电池管理-恢复出厂（退回厂家库存）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryFactoryRestoreReq true "请求参数"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/battery/factory_restore [post]
func (*BatteryApi) FactoryRestoreBattery(c *gin.Context) {
	var req model.BatteryFactoryRestoreReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)

	if err := service.GroupApp.Battery.FactoryRestoreBattery(context.Background(), req, userClaims, orgID); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", map[string]interface{}{"message": "恢复出厂成功"})
}

// TransferBattery 电池调拨
// @Summary 电池调拨
// @Description BMS 电池管理-调拨（组织转移）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryTransferReq true "请求参数"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/battery/transfer [post]
func (*BatteryApi) TransferBattery(c *gin.Context) {
	var req model.BatteryTransferReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)

	if err := service.GroupApp.Battery.TransferBattery(context.Background(), req, userClaims, orgID); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", map[string]interface{}{"message": "调拨成功"})
}

// PreviewRollbackBattery 电池回退预览
// @Summary 电池回退预览
// @Description BMS 电池管理-回退前查询目标机构
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param device_id query string true "设备ID"
// @Success 200 {object} model.BatteryRollbackPreviewResp
// @Router /api/v1/battery/rollback/preview [get]
func (*BatteryApi) PreviewRollbackBattery(c *gin.Context) {
	var req model.BatteryRollbackPreviewReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)

	data, err := service.GroupApp.Battery.PreviewRollbackBattery(context.Background(), req.DeviceID, userClaims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// RollbackBattery 电池回退
// @Summary 电池回退
// @Description BMS 电池管理-按最近一次入库来源回退
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryRollbackReq true "请求参数"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/battery/rollback [post]
func (*BatteryApi) RollbackBattery(c *gin.Context) {
	var req model.BatteryRollbackReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)

	if err := service.GroupApp.Battery.RollbackBattery(context.Background(), req, userClaims, orgID); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", map[string]interface{}{"message": "回退成功"})
}

// ActivateBattery 电池激活
// @Summary 电池激活
// @Description BMS 电池管理-激活（绑定 APP 用户）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryActivateReq true "请求参数"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/battery/activate [post]
func (*BatteryApi) ActivateBattery(c *gin.Context) {
	var req model.BatteryActivateReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)

	if err := service.GroupApp.Battery.ActivateBattery(context.Background(), req, userClaims, orgID); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", map[string]interface{}{"message": "激活成功"})
}

// CompleteBatteryInfo 电池信息补全
// @Summary 电池信息补全
// @Description BMS 电池管理-批量补全电芯品牌和电池型号
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryCompleteInfoReq true "请求参数"
// @Success 200 {object} model.BatteryCompleteInfoResp
// @Router /api/v1/battery/complete-info [post]
func (*BatteryApi) CompleteBatteryInfo(c *gin.Context) {
	var req model.BatteryCompleteInfoReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)

	data, err := service.GroupApp.Battery.CompleteBatteryInfo(context.Background(), req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
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

// BatchSendCommand 批量下发指令
// @Summary 批量下发指令
// @Description BMS 电池管理-批量下发指令（仅在线设备）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryBatchCommandReq true "请求参数"
// @Success 200 {object} model.BatteryBatchCommandResp
// @Router /api/v1/battery/batch-command [post]
func (*BatteryApi) BatchSendCommand(c *gin.Context) {
	var req model.BatteryBatchCommandReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.Battery.BatchSendCommand(context.Background(), req, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Set("data", data)
}

// BatchPushOTA 批量 OTA 推送
// @Summary 批量 OTA 推送
// @Description BMS 电池管理-批量 OTA 推送（创建升级任务并触发推送）
// @Tags 电池管理
// @Accept json
// @Produce json
// @Param body body model.BatteryBatchOtaPushReq true "请求参数"
// @Success 200 {object} model.BatteryBatchOtaPushResp
// @Router /api/v1/battery/batch-ota [post]
func (*BatteryApi) BatchPushOTA(c *gin.Context) {
	var req model.BatteryBatchOtaPushReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	dealerIDVal, _ := c.Get(middleware.DealerIDContextKey)
	dealerID, _ := dealerIDVal.(string)

	data, err := service.GroupApp.Battery.BatchPushOTA(context.Background(), req, userClaims, dealerID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
