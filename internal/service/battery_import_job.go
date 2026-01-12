package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type batteryImportJob struct {
	ID            string     `gorm:"column:id"`
	TenantID      string     `gorm:"column:tenant_id"`
	OperatorID    string     `gorm:"column:operator_id"`
	Status        string     `gorm:"column:status"`
	TotalRows     int        `gorm:"column:total_rows"`
	ProcessedRows int        `gorm:"column:processed_rows"`
	SuccessRows   int        `gorm:"column:success_rows"`
	FailedRows    int        `gorm:"column:failed_rows"`
	ErrorMessage  *string    `gorm:"column:error_message"`
	StartedAt     *time.Time `gorm:"column:started_at"`
	FinishedAt    *time.Time `gorm:"column:finished_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

type batteryImportJobLog struct {
	ID           int64     `gorm:"column:id"`
	JobID        string    `gorm:"column:job_id"`
	TenantID     string    `gorm:"column:tenant_id"`
	RowNumber    *int      `gorm:"column:row_number"`
	Level        string    `gorm:"column:level"`
	DeviceNumber *string   `gorm:"column:device_number"`
	Message      string    `gorm:"column:message"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func normalizeHeader(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "*")
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func parseDateYYYYMMDD(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (b *Battery) CreateBatteryImportJob(ctx context.Context, filePath string, claims *utils.UserClaims) (*model.BatteryImportJobCreateResp, error) {
	jobID := uuid.New()
	now := time.Now()

	db := global.DB.WithContext(ctx)
	if err := db.Table("battery_import_jobs").Create(map[string]any{
		"id":          jobID,
		"tenant_id":   claims.TenantID,
		"operator_id": claims.ID,
		"status":      "RUNNING",
		"started_at":  now,
	}).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	go func() {
		defer func() {
			_ = os.Remove(filePath)
			if r := recover(); r != nil {
				logrus.WithField("job_id", jobID).Errorf("battery import job panic: %v", r)
				msg := fmt.Sprintf("panic: %v", r)
				_ = global.DB.Table("battery_import_jobs").Where("id = ? AND tenant_id = ?", jobID, claims.TenantID).Updates(map[string]any{
					"status":        "FAILED",
					"error_message": msg,
					"finished_at":   time.Now(),
				}).Error
			}
		}()

		if err := runBatteryImportJob(context.Background(), jobID, filePath, claims); err != nil {
			logrus.WithError(err).WithField("job_id", jobID).Error("battery import job failed")
			msg := err.Error()
			_ = global.DB.Table("battery_import_jobs").Where("id = ? AND tenant_id = ?", jobID, claims.TenantID).Updates(map[string]any{
				"status":        "FAILED",
				"error_message": msg,
				"finished_at":   time.Now(),
			}).Error
		}
	}()

	return &model.BatteryImportJobCreateResp{JobID: jobID}, nil
}

func (*Battery) GetBatteryImportJobStatus(ctx context.Context, jobID string, claims *utils.UserClaims) (*model.BatteryImportJobStatusResp, error) {
	db := global.DB.WithContext(ctx)
	var job batteryImportJob
	if err := db.Table("battery_import_jobs").
		Where("id = ? AND tenant_id = ?", jobID, claims.TenantID).
		Scan(&job).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if job.ID == "" {
		return nil, errcode.New(404)
	}

	format := func(t *time.Time) *string {
		if t == nil {
			return nil
		}
		s := t.Format("2006-01-02 15:04:05")
		return &s
	}

	createdAt := job.CreatedAt.Format("2006-01-02 15:04:05")
	return &model.BatteryImportJobStatusResp{
		JobID:         job.ID,
		Status:        job.Status,
		TotalRows:     job.TotalRows,
		ProcessedRows: job.ProcessedRows,
		SuccessRows:   job.SuccessRows,
		FailedRows:    job.FailedRows,
		ErrorMessage:  job.ErrorMessage,
		StartedAt:     format(job.StartedAt),
		FinishedAt:    format(job.FinishedAt),
		CreatedAt:     createdAt,
	}, nil
}

func (*Battery) GetBatteryImportJobLogs(ctx context.Context, jobID string, afterID int64, limit int, claims *utils.UserClaims) (*model.BatteryImportJobLogListResp, error) {
	db := global.DB.WithContext(ctx)
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	// 先校验任务是否属于当前租户
	var exists int64
	if err := db.Table("battery_import_jobs").Where("id = ? AND tenant_id = ?", jobID, claims.TenantID).Count(&exists).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if exists == 0 {
		return nil, errcode.New(404)
	}

	var rows []batteryImportJobLog
	q := db.Table("battery_import_job_logs").
		Where("job_id = ? AND tenant_id = ?", jobID, claims.TenantID).
		Order("id ASC").
		Limit(limit)
	if afterID > 0 {
		q = q.Where("id > ?", afterID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	nextAfterID := afterID
	list := make([]model.BatteryImportJobLogItem, 0, len(rows))
	for _, r := range rows {
		if r.ID > nextAfterID {
			nextAfterID = r.ID
		}
		var rowNumber *int
		if r.RowNumber != nil {
			rowNumber = r.RowNumber
		}
		list = append(list, model.BatteryImportJobLogItem{
			ID:           r.ID,
			RowNumber:    rowNumber,
			Level:        r.Level,
			DeviceNumber: r.DeviceNumber,
			Message:      r.Message,
			CreatedAt:    r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &model.BatteryImportJobLogListResp{
		List:        list,
		NextAfterID: nextAfterID,
	}, nil
}

func runBatteryImportJob(ctx context.Context, jobID string, filePath string, claims *utils.UserClaims) error {
	db := global.DB.WithContext(ctx)

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return errcode.WithVars(errcode.CodeFileSaveError, map[string]interface{}{"error": err.Error()})
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("close excel file error")
		}
	}()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return errcode.WithVars(errcode.CodeFileSaveError, map[string]interface{}{"error": err.Error()})
	}
	if len(rows) < 2 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "导入文件无数据"})
	}

	headers := rows[0]
	headerIndex := make(map[string]int)
	for i, h := range headers {
		k := normalizeHeader(h)
		if k != "" {
			headerIndex[k] = i
		}
	}

	colItemUUID, okItem := pickHeader(headerIndex, []string{"电池序列号id", "item_uuid", "序列号", "device_number"})
	colBatch, okBatch := pickHeader(headerIndex, []string{"批号", "batch_number", "批次号"})
	colProductSpec, okProductSpec := pickHeader(headerIndex, []string{"产品规格", "product_spec", "规格"})
	colOrderNumber, okOrderNumber := pickHeader(headerIndex, []string{"订单编号", "order_number", "订单号"})
	colBle, _ := pickHeader(headerIndex, []string{"蓝牙mac", "ble_mac"})
	colComm, _ := pickHeader(headerIndex, []string{"4g通讯卡id", "comm_chip_id", "4g卡id"})
	colModel, _ := pickHeader(headerIndex, []string{"电池型号", "battery_model"})
	colProd, _ := pickHeader(headerIndex, []string{"出厂日期", "production_date"})
	colWarranty, _ := pickHeader(headerIndex, []string{"质保到期", "warranty_expire_date"})

	if !okItem || !okBatch || !okProductSpec || !okOrderNumber {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "模板表头缺少必填列：电池序列号ID / 批号 / 产品规格 / 订单编号",
		})
	}

	totalRows := len(rows) - 1
	_ = db.Table("battery_import_jobs").Where("id = ? AND tenant_id = ?", jobID, claims.TenantID).Updates(map[string]any{
		"total_rows": totalRows,
	}).Error

	successRows := 0
	failedRows := 0
	processedRows := 0

	for idx := 1; idx < len(rows); idx++ {
		rowNumber := idx + 1
		row := rows[idx]
		processedRows++

		itemUUID := strings.TrimSpace(cellAt(row, colItemUUID))
		batchNumber := strings.TrimSpace(cellAt(row, colBatch))
		productSpec := strings.TrimSpace(cellAt(row, colProductSpec))
		orderNumber := strings.TrimSpace(cellAt(row, colOrderNumber))
		if itemUUID == "" {
			failedRows++
			appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", nil, "电池序列号ID不能为空")
			continue
		}
		if batchNumber == "" {
			failedRows++
			appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", &itemUUID, "批号不能为空")
			continue
		}
		if productSpec == "" {
			failedRows++
			appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", &itemUUID, "产品规格不能为空")
			continue
		}
		if orderNumber == "" {
			failedRows++
			appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", &itemUUID, "订单编号不能为空")
			continue
		}

		bleMac := strings.TrimSpace(cellAt(row, colBle))
		commChipID := strings.TrimSpace(cellAt(row, colComm))
		modelName := strings.TrimSpace(cellAt(row, colModel))
		productionDateStr := strings.TrimSpace(cellAt(row, colProd))
		warrantyExpireDateStr := strings.TrimSpace(cellAt(row, colWarranty))

		// 查找设备（item_uuid -> devices.device_number）
		device, err := query.Device.WithContext(ctx).Where(
			query.Device.DeviceNumber.Eq(itemUUID),
			query.Device.TenantID.Eq(claims.TenantID),
		).First()
		if err != nil {
			failedRows++
			if err == gorm.ErrRecordNotFound {
				appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", &itemUUID, "设备不存在（devices.device_number 未找到）")
				continue
			}
			appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", &itemUUID, "查询设备失败: "+err.Error())
			continue
		}

		// 电池型号（按 name 查找 id）
		var batteryModelID *string
		if modelName != "" {
			bm, err := query.BatteryModel.WithContext(ctx).Where(
				query.BatteryModel.TenantID.Eq(claims.TenantID),
				query.BatteryModel.Name.Eq(modelName),
			).First()
			if err != nil {
				failedRows++
				if err == gorm.ErrRecordNotFound {
					appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", &itemUUID, "电池型号不存在: "+modelName)
					continue
				}
				appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", &itemUUID, "查询电池型号失败: "+err.Error())
				continue
			}
			batteryModelID = &bm.ID
		}

		productionDate, err := parseDateYYYYMMDD(productionDateStr)
		if err != nil {
			failedRows++
			appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", &itemUUID, "出厂日期格式错误，应为 YYYY-MM-DD")
			continue
		}
		warrantyExpireDate, err := parseDateYYYYMMDD(warrantyExpireDateStr)
		if err != nil {
			failedRows++
			appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", &itemUUID, "质保到期格式错误，应为 YYYY-MM-DD")
			continue
		}

		var blePtr *string
		if bleMac != "" {
			blePtr = &bleMac
		}
		var commPtr *string
		if commChipID != "" {
			commPtr = &commChipID
		}

		// Upsert device_batteries（空值不覆盖）
		if err := upsertDeviceBattery(
			ctx,
			device.ID,
			itemUUID,
			&batchNumber,
			&productSpec,
			&orderNumber,
			batteryModelID,
			blePtr,
			commPtr,
			productionDate,
			warrantyExpireDate,
			nil,
		); err != nil {
			failedRows++
			appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "ERROR", &itemUUID, "写入电池信息失败: "+err.Error())
			continue
		}

		// 运营日志：IMPORT
		desc := fmt.Sprintf(
			"导入电池信息（批号=%s，规格=%s，订单=%s%s%s）",
			batchNumber,
			productSpec,
			orderNumber,
			formatMaybe("，蓝牙=", blePtr),
			formatMaybe("，4G卡=", commPtr),
		)
		_ = CreateBatteryOperationLog(ctx, claims.TenantID, device.ID, itemUUID, BatteryOpTypeImport, &claims.ID, &desc, map[string]any{
			"job_id":           jobID,
			"battery_model_id": batteryModelID,
		})

		successRows++
		appendImportJobLog(ctx, jobID, claims.TenantID, &rowNumber, "INFO", &itemUUID, "导入成功")

		// 更新进度（降低写频）
		if processedRows%10 == 0 || processedRows == totalRows {
			_ = db.Table("battery_import_jobs").Where("id = ? AND tenant_id = ?", jobID, claims.TenantID).Updates(map[string]any{
				"processed_rows": processedRows,
				"success_rows":   successRows,
				"failed_rows":    failedRows,
			}).Error
		}
	}

	finished := time.Now()
	if err := db.Table("battery_import_jobs").Where("id = ? AND tenant_id = ?", jobID, claims.TenantID).Updates(map[string]any{
		"status":         "SUCCESS",
		"processed_rows": processedRows,
		"success_rows":   successRows,
		"failed_rows":    failedRows,
		"finished_at":    finished,
	}).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func pickHeader(headerIndex map[string]int, candidates []string) (int, bool) {
	for _, c := range candidates {
		if idx, ok := headerIndex[normalizeHeader(c)]; ok {
			return idx, true
		}
	}
	return -1, false
}

func cellAt(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return row[col]
}

func appendImportJobLog(ctx context.Context, jobID, tenantID string, rowNumber *int, level string, deviceNumber *string, message string) {
	_ = global.DB.WithContext(ctx).Table("battery_import_job_logs").Create(map[string]any{
		"job_id":        jobID,
		"tenant_id":     tenantID,
		"row_number":    rowNumber,
		"level":         level,
		"device_number": deviceNumber,
		"message":       message,
	}).Error
}

func formatMaybe(prefix string, v *string) string {
	if v == nil || *v == "" {
		return ""
	}
	return prefix + *v
}

// upsertDeviceBattery Upsert device_batteries with "empty does not overwrite" semantics.
// ownerOrgID: optional, only used when existing owner_org_id is NULL.
func upsertDeviceBattery(
	ctx context.Context,
	deviceID string,
	itemUUID string,
	batchNumber *string,
	productSpec *string,
	orderNumber *string,
	batteryModelID *string,
	bleMac *string,
	commChipID *string,
	productionDate *time.Time,
	warrantyExpireDate *time.Time,
	ownerOrgID *string,
) error {
	db := global.DB.WithContext(ctx)
	now := time.Now()

	sql := `
		INSERT INTO device_batteries (
			device_id, battery_model_id, batch_number, product_spec, order_number, ble_mac, comm_chip_id, item_uuid,
			production_date, warranty_expire_date, activation_status, transfer_status, owner_org_id, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, 'INACTIVE', 'FACTORY', ?, ?
		)
		ON CONFLICT (device_id) DO UPDATE SET
			battery_model_id = COALESCE(EXCLUDED.battery_model_id, device_batteries.battery_model_id),
			batch_number = EXCLUDED.batch_number,
			product_spec = EXCLUDED.product_spec,
			order_number = EXCLUDED.order_number,
			ble_mac = COALESCE(EXCLUDED.ble_mac, device_batteries.ble_mac),
			comm_chip_id = COALESCE(EXCLUDED.comm_chip_id, device_batteries.comm_chip_id),
			item_uuid = EXCLUDED.item_uuid,
			production_date = COALESCE(EXCLUDED.production_date, device_batteries.production_date),
			warranty_expire_date = COALESCE(EXCLUDED.warranty_expire_date, device_batteries.warranty_expire_date),
			owner_org_id = COALESCE(device_batteries.owner_org_id, EXCLUDED.owner_org_id),
			updated_at = ?
	`
	if err := db.Exec(
		sql,
		deviceID,
		batteryModelID,
		batchNumber,
		productSpec,
		orderNumber,
		bleMac,
		commChipID,
		itemUUID,
		productionDate,
		warrantyExpireDate,
		ownerOrgID,
		now,
		now,
	).Error; err != nil {
		return err
	}
	return nil
}
