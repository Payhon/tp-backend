package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	"project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	batteryWarrantyRecalcSourceModelChange = "MODEL_CHANGE"
	batteryWarrantyRecalcSourceManualScan  = "MANUAL_SCAN"

	batteryWarrantyRecalcStatusRunning = "RUNNING"
	batteryWarrantyRecalcStatusSuccess = "SUCCESS"
	batteryWarrantyRecalcStatusFailed  = "FAILED"
)

type batteryWarrantyRecalcJob struct {
	ID            string     `gorm:"column:id"`
	TenantID      string     `gorm:"column:tenant_id"`
	OperatorID    *string    `gorm:"column:operator_id"`
	Source        string     `gorm:"column:source"`
	ScopeModelID  *string    `gorm:"column:scope_model_id"`
	Status        string     `gorm:"column:status"`
	TotalRows     int        `gorm:"column:total_rows"`
	ProcessedRows int        `gorm:"column:processed_rows"`
	SuccessRows   int        `gorm:"column:success_rows"`
	SkippedRows   int        `gorm:"column:skipped_rows"`
	FailedRows    int        `gorm:"column:failed_rows"`
	ErrorMessage  *string    `gorm:"column:error_message"`
	StartedAt     *time.Time `gorm:"column:started_at"`
	FinishedAt    *time.Time `gorm:"column:finished_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

type batteryWarrantyRecalcJobLog struct {
	ID             int64     `gorm:"column:id"`
	JobID          string    `gorm:"column:job_id"`
	TenantID       string    `gorm:"column:tenant_id"`
	Level          string    `gorm:"column:level"`
	DeviceID       *string   `gorm:"column:device_id"`
	DeviceNumber   *string   `gorm:"column:device_number"`
	BatteryModelID *string   `gorm:"column:battery_model_id"`
	Message        string    `gorm:"column:message"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

type batteryWarrantyRecalcCandidate struct {
	DeviceID               string     `gorm:"column:device_id"`
	DeviceNumber           string     `gorm:"column:device_number"`
	BatteryModelID         *string    `gorm:"column:battery_model_id"`
	ActivationStatus       *string    `gorm:"column:activation_status"`
	ActivationDate         *time.Time `gorm:"column:activation_date"`
	WarrantyManualOverride bool       `gorm:"column:warranty_manual_override"`
	WarrantyMonths         *int32     `gorm:"column:warranty_months"`
}

func createBatteryWarrantyRecalcJobTx(
	ctx context.Context,
	db *gorm.DB,
	tenantID string,
	operatorID string,
	source string,
	scopeModelID *string,
) (string, error) {
	if db == nil {
		db = global.DB
	}
	now := time.Now()
	jobID := uuid.New()
	var operatorPtr *string
	if strings.TrimSpace(operatorID) != "" {
		operatorPtr = &operatorID
	}
	row := batteryWarrantyRecalcJob{
		ID:           jobID,
		TenantID:     tenantID,
		OperatorID:   operatorPtr,
		Source:       source,
		ScopeModelID: scopeModelID,
		Status:       batteryWarrantyRecalcStatusRunning,
		StartedAt:    &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.WithContext(ctx).Table("battery_warranty_recalc_jobs").Create(&row).Error; err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return jobID, nil
}

func startBatteryWarrantyRecalcJob(jobID string) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithField("job_id", jobID).Errorf("battery warranty recalc job panic: %v", r)
				msg := fmt.Sprintf("panic: %v", r)
				_ = global.DB.Table("battery_warranty_recalc_jobs").Where("id = ?", jobID).Updates(map[string]any{
					"status":        batteryWarrantyRecalcStatusFailed,
					"error_message": msg,
					"finished_at":   time.Now(),
					"updated_at":    time.Now(),
				}).Error
			}
		}()
		if err := runBatteryWarrantyRecalcJob(context.Background(), jobID); err != nil {
			logrus.WithError(err).WithField("job_id", jobID).Error("battery warranty recalc job failed")
			msg := err.Error()
			_ = global.DB.Table("battery_warranty_recalc_jobs").Where("id = ?", jobID).Updates(map[string]any{
				"status":        batteryWarrantyRecalcStatusFailed,
				"error_message": msg,
				"finished_at":   time.Now(),
				"updated_at":    time.Now(),
			}).Error
		}
	}()
}

func (*UserWarrantyInfo) CreateBatteryWarrantyRecalcJob(ctx context.Context, claims *utils.UserClaims) (*model.BatteryWarrantyRecalcJobCreateResp, error) {
	if claims == nil || strings.TrimSpace(claims.TenantID) == "" {
		return nil, errcode.New(errcode.CodeNoPermission)
	}
	jobID, err := createBatteryWarrantyRecalcJobTx(ctx, global.DB, claims.TenantID, claims.ID, batteryWarrantyRecalcSourceManualScan, nil)
	if err != nil {
		return nil, err
	}
	startBatteryWarrantyRecalcJob(jobID)
	return &model.BatteryWarrantyRecalcJobCreateResp{JobID: jobID}, nil
}

func (*UserWarrantyInfo) GetBatteryWarrantyRecalcJobStatus(ctx context.Context, jobID string, claims *utils.UserClaims) (*model.BatteryWarrantyRecalcJobStatusResp, error) {
	if claims == nil || strings.TrimSpace(claims.TenantID) == "" {
		return nil, errcode.New(errcode.CodeNoPermission)
	}
	var job batteryWarrantyRecalcJob
	if err := global.DB.WithContext(ctx).Table("battery_warranty_recalc_jobs").
		Where("id = ? AND tenant_id = ?", strings.TrimSpace(jobID), claims.TenantID).
		Scan(&job).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if job.ID == "" {
		return nil, errcode.New(404)
	}
	return batteryWarrantyRecalcJobStatusResp(job), nil
}

func (*UserWarrantyInfo) GetBatteryWarrantyRecalcJobLogs(ctx context.Context, jobID string, afterID int64, limit int, claims *utils.UserClaims) (*model.BatteryWarrantyRecalcJobLogListResp, error) {
	if claims == nil || strings.TrimSpace(claims.TenantID) == "" {
		return nil, errcode.New(errcode.CodeNoPermission)
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var exists int64
	if err := global.DB.WithContext(ctx).Table("battery_warranty_recalc_jobs").
		Where("id = ? AND tenant_id = ?", strings.TrimSpace(jobID), claims.TenantID).
		Count(&exists).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if exists == 0 {
		return nil, errcode.New(404)
	}

	var rows []batteryWarrantyRecalcJobLog
	q := global.DB.WithContext(ctx).Table("battery_warranty_recalc_job_logs").
		Where("job_id = ? AND tenant_id = ?", strings.TrimSpace(jobID), claims.TenantID).
		Order("id ASC").
		Limit(limit)
	if afterID > 0 {
		q = q.Where("id > ?", afterID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	nextAfterID := afterID
	list := make([]model.BatteryWarrantyRecalcJobLogItem, 0, len(rows))
	for _, r := range rows {
		if r.ID > nextAfterID {
			nextAfterID = r.ID
		}
		list = append(list, model.BatteryWarrantyRecalcJobLogItem{
			ID:             r.ID,
			Level:          r.Level,
			DeviceID:       r.DeviceID,
			DeviceNumber:   r.DeviceNumber,
			BatteryModelID: r.BatteryModelID,
			Message:        r.Message,
			CreatedAt:      r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &model.BatteryWarrantyRecalcJobLogListResp{List: list, NextAfterID: nextAfterID}, nil
}

func batteryWarrantyRecalcJobStatusResp(job batteryWarrantyRecalcJob) *model.BatteryWarrantyRecalcJobStatusResp {
	format := func(t *time.Time) *string {
		if t == nil {
			return nil
		}
		s := t.Format("2006-01-02 15:04:05")
		return &s
	}
	return &model.BatteryWarrantyRecalcJobStatusResp{
		JobID:         job.ID,
		Source:        job.Source,
		ScopeModelID:  job.ScopeModelID,
		Status:        job.Status,
		TotalRows:     job.TotalRows,
		ProcessedRows: job.ProcessedRows,
		SuccessRows:   job.SuccessRows,
		SkippedRows:   job.SkippedRows,
		FailedRows:    job.FailedRows,
		ErrorMessage:  job.ErrorMessage,
		StartedAt:     format(job.StartedAt),
		FinishedAt:    format(job.FinishedAt),
		CreatedAt:     job.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func runBatteryWarrantyRecalcJob(ctx context.Context, jobID string) error {
	db := global.DB.WithContext(ctx)
	var job batteryWarrantyRecalcJob
	if err := db.Table("battery_warranty_recalc_jobs").
		Where("id = ?", strings.TrimSpace(jobID)).
		Scan(&job).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if job.ID == "" {
		return errcode.New(404)
	}

	rows, err := loadBatteryWarrantyRecalcCandidates(ctx, job)
	if err != nil {
		return err
	}

	_ = db.Table("battery_warranty_recalc_jobs").Where("id = ? AND tenant_id = ?", job.ID, job.TenantID).Updates(map[string]any{
		"total_rows": len(rows),
		"updated_at": time.Now(),
	}).Error

	processedRows := 0
	successRows := 0
	skippedRows := 0
	failedRows := 0
	for _, row := range rows {
		processedRows++
		result := processBatteryWarrantyRecalcCandidate(ctx, job, row)
		switch result {
		case "success":
			successRows++
		case "failed":
			failedRows++
		default:
			skippedRows++
		}
		if processedRows%10 == 0 || processedRows == len(rows) {
			_ = db.Table("battery_warranty_recalc_jobs").Where("id = ? AND tenant_id = ?", job.ID, job.TenantID).Updates(map[string]any{
				"processed_rows": processedRows,
				"success_rows":   successRows,
				"skipped_rows":   skippedRows,
				"failed_rows":    failedRows,
				"updated_at":     time.Now(),
			}).Error
		}
	}

	finished := time.Now()
	if err := db.Table("battery_warranty_recalc_jobs").Where("id = ? AND tenant_id = ?", job.ID, job.TenantID).Updates(map[string]any{
		"status":         batteryWarrantyRecalcStatusSuccess,
		"processed_rows": processedRows,
		"success_rows":   successRows,
		"skipped_rows":   skippedRows,
		"failed_rows":    failedRows,
		"finished_at":    finished,
		"updated_at":     finished,
	}).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func loadBatteryWarrantyRecalcCandidates(ctx context.Context, job batteryWarrantyRecalcJob) ([]batteryWarrantyRecalcCandidate, error) {
	q := global.DB.WithContext(ctx).
		Table("device_batteries AS dbat").
		Select(`
			dbat.device_id AS device_id,
			d.device_number AS device_number,
			dbat.battery_model_id AS battery_model_id,
			dbat.activation_status AS activation_status,
			dbat.activation_date AS activation_date,
			COALESCE(dbat.warranty_manual_override, false) AS warranty_manual_override,
			bm.warranty_months AS warranty_months
		`).
		Joins(`JOIN devices AS d ON d.id = dbat.device_id AND d.tenant_id = ?`, job.TenantID).
		Joins(`LEFT JOIN battery_bms_models AS bm ON bm.id = dbat.battery_model_id AND bm.tenant_id = ?`, job.TenantID).
		Order("dbat.updated_at ASC, dbat.device_id ASC")

	switch job.Source {
	case batteryWarrantyRecalcSourceModelChange:
		if job.ScopeModelID == nil || strings.TrimSpace(*job.ScopeModelID) == "" {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "scope_model_id is required"})
		}
		q = q.Where("dbat.battery_model_id = ?", strings.TrimSpace(*job.ScopeModelID))
	case batteryWarrantyRecalcSourceManualScan:
		q = q.Where("dbat.warranty_expire_date IS NULL")
	default:
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "invalid warranty recalc source"})
	}

	var rows []batteryWarrantyRecalcCandidate
	if err := q.Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return rows, nil
}

func processBatteryWarrantyRecalcCandidate(ctx context.Context, job batteryWarrantyRecalcJob, row batteryWarrantyRecalcCandidate) string {
	deviceID := row.DeviceID
	deviceNumber := nullableTrimmedString(row.DeviceNumber)
	logSkip := func(message string) string {
		appendBatteryWarrantyRecalcJobLog(ctx, job, "WARN", &deviceID, deviceNumber, row.BatteryModelID, message)
		return "skipped"
	}

	if row.WarrantyManualOverride {
		return logSkip("已人工覆盖质保信息，跳过")
	}
	if row.ActivationStatus == nil || strings.ToUpper(strings.TrimSpace(*row.ActivationStatus)) != "ACTIVE" {
		return logSkip("电池未激活，跳过")
	}
	if row.ActivationDate == nil {
		return logSkip("缺少激活日期，跳过")
	}
	if row.BatteryModelID == nil || strings.TrimSpace(*row.BatteryModelID) == "" {
		return logSkip("未关联 BMS 型号，跳过")
	}
	if row.WarrantyMonths == nil || *row.WarrantyMonths <= 0 {
		return logSkip("BMS 型号未配置有效质保时长，跳过")
	}

	activationDate := *row.ActivationDate
	warrantyMonths := *row.WarrantyMonths
	expireDate := activationDate.AddDate(0, int(warrantyMonths), 0)
	now := time.Now()
	updates := map[string]any{
		"warranty_start_date":  activationDate,
		"warranty_months":      warrantyMonths,
		"warranty_expire_date": expireDate,
		"warranty_updated_at":  now,
		"updated_at":           now,
	}
	if job.OperatorID != nil && strings.TrimSpace(*job.OperatorID) != "" {
		updates["warranty_updated_by"] = *job.OperatorID
	}

	tx := global.DB.WithContext(ctx).Table("device_batteries").
		Where("device_id = ?", row.DeviceID).
		Where("COALESCE(warranty_manual_override, false) = false")
	if job.Source == batteryWarrantyRecalcSourceManualScan {
		tx = tx.Where("warranty_expire_date IS NULL")
	}
	if err := tx.Updates(updates).Error; err != nil {
		appendBatteryWarrantyRecalcJobLog(ctx, job, "ERROR", &deviceID, deviceNumber, row.BatteryModelID, "更新质保截止日期失败: "+err.Error())
		return "failed"
	}
	if tx.RowsAffected == 0 {
		return logSkip("质保信息已被其他操作更新，跳过")
	}

	msg := fmt.Sprintf(
		"已按激活日期 %s 和质保 %d 个月更新到期日 %s",
		activationDate.Local().Format("2006-01-02"),
		warrantyMonths,
		expireDate.Local().Format("2006-01-02"),
	)
	appendBatteryWarrantyRecalcJobLog(ctx, job, "INFO", &deviceID, deviceNumber, row.BatteryModelID, msg)
	return "success"
}

func appendBatteryWarrantyRecalcJobLog(ctx context.Context, job batteryWarrantyRecalcJob, level string, deviceID *string, deviceNumber *string, batteryModelID *string, message string) {
	now := time.Now()
	row := batteryWarrantyRecalcJobLog{
		JobID:          job.ID,
		TenantID:       job.TenantID,
		Level:          level,
		DeviceID:       deviceID,
		DeviceNumber:   deviceNumber,
		BatteryModelID: batteryModelID,
		Message:        message,
		CreatedAt:      now,
	}
	_ = global.DB.WithContext(ctx).Table("battery_warranty_recalc_job_logs").Create(&row).Error
}

func nullableTrimmedString(v string) *string {
	s := strings.TrimSpace(v)
	if s == "" {
		return nil
	}
	return &s
}
