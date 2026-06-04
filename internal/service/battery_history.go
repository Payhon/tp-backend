package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"project/internal/model"
	query "project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

const (
	bmsHistoryMaxRangeMillis  int64 = 31 * 24 * 60 * 60 * 1000
	bmsHistoryExportRetention       = 7 * 24 * time.Hour
)

type bmsHistoryMergedRow struct {
	TSMs           int64  `gorm:"column:ts_ms"`
	DataType       string `gorm:"column:data_type"`
	DataIdentifier string `gorm:"column:data_identifier"`
	DataName       string `gorm:"column:data_name"`
	Value          string `gorm:"column:value"`
}

type bmsHistoryWideCellRow struct {
	TSMs           int64  `gorm:"column:ts_ms"`
	DataType       string `gorm:"column:data_type"`
	DataIdentifier string `gorm:"column:data_identifier"`
	Value          string `gorm:"column:value"`
}

type bmsHistoryTSRow struct {
	TSMs int64 `gorm:"column:ts_ms"`
}

type bmsHistoryWideColumnRow struct {
	DataType       string `gorm:"column:data_type"`
	DataIdentifier string `gorm:"column:data_identifier"`
	DataName       string `gorm:"column:data_name"`
}

type bmsHistoryWideValueRow struct {
	DataType       string `gorm:"column:data_type"`
	DataIdentifier string `gorm:"column:data_identifier"`
	Value          string `gorm:"column:value"`
}

type bmsHistoryDeviceOptionRow struct {
	DeviceID     string  `gorm:"column:device_id"`
	DeviceNumber string  `gorm:"column:device_number"`
	DeviceName   *string `gorm:"column:device_name"`
	ItemUUID     *string `gorm:"column:item_uuid"`
	BmsCommType  *int    `gorm:"column:bms_comm_type"`
	BleMac       *string `gorm:"column:ble_mac"`
	CommChipID   *string `gorm:"column:comm_chip_id"`
}

type bmsHistoryExportJob struct {
	ID           string     `gorm:"column:id"`
	TenantID     string     `gorm:"column:tenant_id"`
	CreatorUser  string     `gorm:"column:creator_user_id"`
	CreatorOrgID *string    `gorm:"column:creator_org_id"`
	DeviceID     string     `gorm:"column:device_id"`
	ViewMode     string     `gorm:"column:view_mode"`
	StartTimeMs  int64      `gorm:"column:start_time_ms"`
	EndTimeMs    int64      `gorm:"column:end_time_ms"`
	Status       string     `gorm:"column:status"`
	FilePath     *string    `gorm:"column:file_path"`
	FileName     *string    `gorm:"column:file_name"`
	FileSize     *int64     `gorm:"column:file_size"`
	FileExpireAt *time.Time `gorm:"column:file_expire_at"`
	ErrorMessage *string    `gorm:"column:error_message"`
	DownloadedAt *time.Time `gorm:"column:downloaded_at"`
	StartedAt    *time.Time `gorm:"column:started_at"`
	FinishedAt   *time.Time `gorm:"column:finished_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

type bmsHistoryExportPendingRow struct {
	TaskID       string    `gorm:"column:task_id"`
	DeviceID     string    `gorm:"column:device_id"`
	DeviceNumber string    `gorm:"column:device_number"`
	ViewMode     string    `gorm:"column:view_mode"`
	StartTimeMs  int64     `gorm:"column:start_time_ms"`
	EndTimeMs    int64     `gorm:"column:end_time_ms"`
	FileName     string    `gorm:"column:file_name"`
	FinishedAt   time.Time `gorm:"column:finished_at"`
}

func validateBMSHistoryRange(startTime, endTime int64) error {
	if startTime <= 0 || endTime <= 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "时间范围参数无效"})
	}
	if endTime < startTime {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "结束时间不能早于开始时间"})
	}
	if endTime-startTime > bmsHistoryMaxRangeMillis {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "时间范围不能超过31天"})
	}
	return nil
}

func applyBMSHistoryOrgScope(queryBuilder *gorm.DB, tenantID, orgID string) *gorm.DB {
	if orgID == "" {
		return queryBuilder
	}

	var orgType string
	_ = global.DB.Table("orgs").
		Select("org_type").
		Where("id = ? AND tenant_id = ?", orgID, tenantID).
		Scan(&orgType).Error
	orgType = strings.TrimSpace(orgType)
	if orgType == model.OrgTypeBMSFactory {
		return queryBuilder.Where(`(
			dbat.owner_org_id IS NULL OR dbat.owner_org_id IN (
				SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
			)
		)`, tenantID, orgID)
	}

	return queryBuilder.Where(`dbat.owner_org_id IN (
		SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
	)`, tenantID, orgID)
}

func ensureBMSHistoryDeviceAccess(ctx context.Context, deviceID string, claims *utils.UserClaims, orgID string) error {
	_, err := query.Device.WithContext(ctx).
		Where(query.Device.ID.Eq(deviceID), query.Device.TenantID.Eq(claims.TenantID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "设备不存在"})
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return checkDeviceOrgAccess(ctx, deviceID, claims.TenantID, orgID)
}

func getBMSHistoryDeviceTemplateID(ctx context.Context, deviceID, tenantID string) (string, error) {
	var row struct {
		DeviceTemplateID *string `gorm:"column:device_template_id"`
	}
	if err := global.DB.WithContext(ctx).Table("devices AS d").
		Select("dc.device_template_id AS device_template_id").
		Joins("LEFT JOIN device_configs AS dc ON dc.id = d.device_config_id").
		Where("d.id = ? AND d.tenant_id = ?", deviceID, tenantID).
		Limit(1).
		Scan(&row).Error; err != nil {
		return "", err
	}
	if row.DeviceTemplateID == nil {
		return "", nil
	}
	return strings.TrimSpace(*row.DeviceTemplateID), nil
}

func buildBMSHistoryMergedSQL() string {
	return `
		SELECT td.ts AS ts_ms,
		       'telemetry' AS data_type,
		       td.key AS data_identifier,
		       COALESCE(dmt.data_name, td.key) AS data_name,
		       COALESCE(td.string_v, td.number_v::text, td.bool_v::text, '') AS value
		  FROM telemetry_datas td
		  LEFT JOIN device_model_telemetry dmt
		    ON dmt.device_template_id = ? AND dmt.data_identifier = td.key
		 WHERE td.device_id = ?
		   AND (td.tenant_id = ? OR td.tenant_id IS NULL)
		   AND td.ts BETWEEN ? AND ?
		UNION ALL
		SELECT (EXTRACT(EPOCH FROM ah.ts) * 1000)::bigint AS ts_ms,
		       'attribute' AS data_type,
		       ah.key AS data_identifier,
		       COALESCE(dma.data_name, ah.key) AS data_name,
		       COALESCE(ah.string_v, ah.number_v::text, ah.bool_v::text, '') AS value
		  FROM attribute_history_datas ah
		  LEFT JOIN device_model_attributes dma
		    ON dma.device_template_id = ? AND dma.data_identifier = ah.key
		 WHERE ah.device_id = ?
		   AND (ah.tenant_id = ? OR ah.tenant_id IS NULL)
		   AND ah.ts BETWEEN to_timestamp(?::double precision / 1000.0)
		                 AND to_timestamp(?::double precision / 1000.0)
	`
}

func buildBMSHistoryMergedArgs(templateID, deviceID, tenantID string, startTime, endTime int64) []interface{} {
	return []interface{}{
		templateID, deviceID, tenantID, startTime, endTime,
		templateID, deviceID, tenantID, startTime, endTime,
	}
}

func buildBMSHistoryWideColumnKey(dataType, identifier string) string {
	return dataType + "__" + identifier
}

var bmsHistoryWideExcludedIdentifiers = map[string]struct{}{
	"balancingOn":  {},
	"bms.snapshot": {},
	"protectCount": {},
	"protectOn":    {},
	"vPackV":       {},
}

var bmsHistoryWideKnownJSONIdentifiers = map[string]struct{}{
	"bms.snapshot":           {},
	"cell.balancing":         {},
	"cell.voltagesMv":        {},
	"customParams":           {},
	"temperature.cellTempsC": {},
}

func isBMSHistoryWideJSONValue(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return false
	}
	if !((strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}")) ||
		(strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]"))) {
		return false
	}
	return json.Valid([]byte(value))
}

func shouldExcludeBMSHistoryWideIdentifier(identifier string) bool {
	identifier = strings.TrimSpace(identifier)
	if _, ok := bmsHistoryWideExcludedIdentifiers[identifier]; ok {
		return true
	}
	if _, ok := bmsHistoryWideKnownJSONIdentifiers[identifier]; ok {
		return true
	}
	return false
}

func filterBMSHistoryWideColumnRows(columnRows []bmsHistoryWideColumnRow, valueRows []bmsHistoryWideValueRow) []bmsHistoryWideColumnRow {
	jsonColumnKeys := make(map[string]struct{})
	for _, row := range valueRows {
		if shouldExcludeBMSHistoryWideIdentifier(row.DataIdentifier) || isBMSHistoryWideJSONValue(row.Value) {
			jsonColumnKeys[buildBMSHistoryWideColumnKey(row.DataType, row.DataIdentifier)] = struct{}{}
		}
	}

	filtered := make([]bmsHistoryWideColumnRow, 0, len(columnRows))
	for _, row := range columnRows {
		if shouldExcludeBMSHistoryWideIdentifier(row.DataIdentifier) {
			continue
		}
		if _, ok := jsonColumnKeys[buildBMSHistoryWideColumnKey(row.DataType, row.DataIdentifier)]; ok {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func formatBMSHistoryTime(tsMs int64) string {
	return time.UnixMilli(tsMs).Format("2006-01-02 15:04:05")
}

func (*Battery) ListBMSHistoryDevices(ctx context.Context, req model.BMSHistoryDeviceListReq, claims *utils.UserClaims, orgID string) (*model.BMSHistoryDeviceListResp, error) {
	db := global.DB.WithContext(ctx)
	queryBuilder := db.Table("devices AS d").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			d.name AS device_name,
			dbat.item_uuid AS item_uuid,
			dbat.bms_comm_type AS bms_comm_type,
			dbat.ble_mac AS ble_mac,
			dbat.comm_chip_id AS comm_chip_id
		`).
		Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
		Where("d.tenant_id = ?", claims.TenantID)

	queryBuilder = applyBMSHistoryOrgScope(queryBuilder, claims.TenantID, orgID)

	if req.Keyword != nil {
		keyword := strings.TrimSpace(*req.Keyword)
		if keyword != "" {
			like := "%" + keyword + "%"
			queryBuilder = queryBuilder.Where(`(
				d.device_number ILIKE ? OR
				d.name ILIKE ? OR
				dbat.item_uuid ILIKE ? OR
				dbat.ble_mac ILIKE ? OR
				dbat.comm_chip_id ILIKE ?
			)`, like, like, like, like, like)
		}
	}

	var total int64
	if err := queryBuilder.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	rows := make([]bmsHistoryDeviceOptionRow, 0)
	if err := queryBuilder.
		Order("d.update_at DESC NULLS LAST").
		Order("d.created_at DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.BMSHistoryDeviceItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, model.BMSHistoryDeviceItem{
			DeviceID:     row.DeviceID,
			DeviceNumber: row.DeviceNumber,
			DeviceName:   row.DeviceName,
			ItemUUID:     row.ItemUUID,
			BmsCommType:  row.BmsCommType,
			BleMac:       row.BleMac,
			CommChipID:   row.CommChipID,
		})
	}

	return &model.BMSHistoryDeviceListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (*Battery) GetBMSHistory(ctx context.Context, req model.BMSHistoryQueryReq, claims *utils.UserClaims, orgID string) (*model.BMSHistoryQueryResp, error) {
	if err := validateBMSHistoryRange(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if err := ensureBMSHistoryDeviceAccess(ctx, req.DeviceID, claims, orgID); err != nil {
		return nil, err
	}

	templateID, err := getBMSHistoryDeviceTemplateID(ctx, req.DeviceID, claims.TenantID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	switch req.ViewMode {
	case "long":
		return queryBMSLongHistory(ctx, req, claims, templateID)
	case "wide":
		return queryBMSWideHistory(ctx, req, claims, templateID)
	default:
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "不支持的视图模式"})
	}
}

func queryBMSLongHistory(ctx context.Context, req model.BMSHistoryQueryReq, claims *utils.UserClaims, templateID string) (*model.BMSHistoryQueryResp, error) {
	mergedSQL := buildBMSHistoryMergedSQL()
	mergedArgs := buildBMSHistoryMergedArgs(templateID, req.DeviceID, claims.TenantID, req.StartTime, req.EndTime)
	countSQL := "SELECT COUNT(1) AS total FROM (" + mergedSQL + ") merged"

	var countRow struct {
		Total int64 `gorm:"column:total"`
	}
	if err := global.DB.WithContext(ctx).Raw(countSQL, mergedArgs...).Scan(&countRow).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	querySQL := `
		SELECT ts_ms, data_type, data_identifier, data_name, value
		  FROM (` + mergedSQL + `) merged
		 ORDER BY ts_ms DESC, data_type ASC, data_identifier ASC
		 LIMIT ? OFFSET ?
	`

	offset := (req.Page - 1) * req.PageSize
	args := append([]interface{}{}, mergedArgs...)
	args = append(args, req.PageSize, offset)

	rows := make([]bmsHistoryMergedRow, 0)
	if err := global.DB.WithContext(ctx).Raw(querySQL, args...).Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		list = append(list, map[string]any{
			"ts":         row.TSMs,
			"time":       formatBMSHistoryTime(row.TSMs),
			"data_type":  row.DataType,
			"identifier": row.DataIdentifier,
			"data_name":  row.DataName,
			"value":      row.Value,
		})
	}

	return &model.BMSHistoryQueryResp{
		ViewMode: "long",
		List:     list,
		Total:    countRow.Total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func queryBMSWideHistory(ctx context.Context, req model.BMSHistoryQueryReq, claims *utils.UserClaims, templateID string) (*model.BMSHistoryQueryResp, error) {
	mergedSQL := buildBMSHistoryMergedSQL()
	mergedArgs := buildBMSHistoryMergedArgs(templateID, req.DeviceID, claims.TenantID, req.StartTime, req.EndTime)

	columnsSQL := `
		SELECT DISTINCT data_type, data_identifier, data_name
		  FROM (` + mergedSQL + `) merged
		 ORDER BY data_type ASC, data_identifier ASC
	`
	columnRows := make([]bmsHistoryWideColumnRow, 0)
	if err := global.DB.WithContext(ctx).Raw(columnsSQL, mergedArgs...).Scan(&columnRows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	valuesSQL := `
		SELECT DISTINCT data_type, data_identifier, value
		  FROM (` + mergedSQL + `) merged
		 WHERE btrim(value) LIKE '{%' OR btrim(value) LIKE '[%'
	`
	valueRows := make([]bmsHistoryWideValueRow, 0)
	if err := global.DB.WithContext(ctx).Raw(valuesSQL, mergedArgs...).Scan(&valueRows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	columnRows = filterBMSHistoryWideColumnRows(columnRows, valueRows)

	columns := make([]model.BMSHistoryWideColumn, 0, len(columnRows))
	for _, row := range columnRows {
		columns = append(columns, model.BMSHistoryWideColumn{
			Key:        buildBMSHistoryWideColumnKey(row.DataType, row.DataIdentifier),
			DataType:   row.DataType,
			Identifier: row.DataIdentifier,
			DataName:   row.DataName,
		})
	}

	countSQL := `
		SELECT COUNT(1) AS total
		  FROM (
		        SELECT ts_ms
		          FROM (` + mergedSQL + `) merged
		         GROUP BY ts_ms
		       ) ts_rows
	`
	var countRow struct {
		Total int64 `gorm:"column:total"`
	}
	if err := global.DB.WithContext(ctx).Raw(countSQL, mergedArgs...).Scan(&countRow).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	tsSQL := `
		SELECT ts_ms
		  FROM (` + mergedSQL + `) merged
		 GROUP BY ts_ms
		 ORDER BY ts_ms DESC
		 LIMIT ? OFFSET ?
	`
	offset := (req.Page - 1) * req.PageSize
	tsArgs := append([]interface{}{}, mergedArgs...)
	tsArgs = append(tsArgs, req.PageSize, offset)
	tsRows := make([]bmsHistoryTSRow, 0)
	if err := global.DB.WithContext(ctx).Raw(tsSQL, tsArgs...).Scan(&tsRows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if len(tsRows) == 0 {
		return &model.BMSHistoryQueryResp{
			ViewMode: "wide",
			Columns:  columns,
			List:     []map[string]any{},
			Total:    countRow.Total,
			Page:     req.Page,
			PageSize: req.PageSize,
		}, nil
	}

	tsList := make([]int64, 0, len(tsRows))
	for _, ts := range tsRows {
		tsList = append(tsList, ts.TSMs)
	}

	cellSQL := `
		SELECT ts_ms, data_type, data_identifier, value
		  FROM (` + mergedSQL + `) merged
		 WHERE ts_ms IN ?
		 ORDER BY ts_ms DESC, data_type ASC, data_identifier ASC
	`
	cellArgs := append([]interface{}{}, mergedArgs...)
	cellArgs = append(cellArgs, tsList)

	cells := make([]bmsHistoryWideCellRow, 0)
	if err := global.DB.WithContext(ctx).Raw(cellSQL, cellArgs...).Scan(&cells).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	valueMap := make(map[int64]map[string]string, len(tsList))
	for _, ts := range tsList {
		valueMap[ts] = make(map[string]string)
	}
	for _, cell := range cells {
		colKey := buildBMSHistoryWideColumnKey(cell.DataType, cell.DataIdentifier)
		valueMap[cell.TSMs][colKey] = cell.Value
	}

	list := make([]map[string]any, 0, len(tsList))
	for _, ts := range tsList {
		row := map[string]any{
			"ts":   ts,
			"time": formatBMSHistoryTime(ts),
		}
		for _, col := range columns {
			row[col.Key] = valueMap[ts][col.Key]
		}
		list = append(list, row)
	}

	return &model.BMSHistoryQueryResp{
		ViewMode: "wide",
		Columns:  columns,
		List:     list,
		Total:    countRow.Total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (b *Battery) CreateBMSHistoryExportJob(ctx context.Context, req model.BMSHistoryExportCreateReq, claims *utils.UserClaims, orgID string) (*model.BMSHistoryExportCreateResp, error) {
	if err := validateBMSHistoryRange(req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if err := ensureBMSHistoryDeviceAccess(ctx, req.DeviceID, claims, orgID); err != nil {
		return nil, err
	}

	jobID := uuid.New()
	now := time.Now()

	var creatorOrgID *string
	orgID = strings.TrimSpace(orgID)
	if orgID != "" {
		creatorOrgID = &orgID
	}

	job := bmsHistoryExportJob{
		ID:           jobID,
		TenantID:     claims.TenantID,
		CreatorUser:  claims.ID,
		CreatorOrgID: creatorOrgID,
		DeviceID:     req.DeviceID,
		ViewMode:     req.ViewMode,
		StartTimeMs:  req.StartTime,
		EndTimeMs:    req.EndTime,
		Status:       "PENDING",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := global.DB.WithContext(ctx).Table("bms_history_export_jobs").Create(&job).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	go func() {
		if err := b.runBMSHistoryExportJob(jobID); err != nil {
			logrus.WithError(err).WithField("job_id", jobID).Error("run bms history export job failed")
		}
	}()

	return &model.BMSHistoryExportCreateResp{JobID: jobID}, nil
}

func (b *Battery) runBMSHistoryExportJob(jobID string) error {
	ctx := context.Background()
	db := global.DB.WithContext(ctx)

	var job bmsHistoryExportJob
	if err := db.Table("bms_history_export_jobs").Where("id = ?", jobID).Scan(&job).Error; err != nil {
		return err
	}
	if job.ID == "" {
		return fmt.Errorf("job not found: %s", jobID)
	}

	now := time.Now()
	if err := db.Table("bms_history_export_jobs").
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":     "RUNNING",
			"started_at": now,
			"updated_at": now,
		}).Error; err != nil {
		return err
	}

	filePath, fileName, err := b.executeBMSHistoryExport(ctx, &job)
	if err != nil {
		msg := err.Error()
		finishedAt := time.Now()
		_ = db.Table("bms_history_export_jobs").
			Where("id = ?", jobID).
			Updates(map[string]any{
				"status":        "FAILED",
				"error_message": msg,
				"finished_at":   finishedAt,
				"updated_at":    finishedAt,
			}).Error
		return err
	}

	stat, statErr := os.Stat(filePath)
	var fileSize int64 = 0
	if statErr == nil {
		fileSize = stat.Size()
	}

	finishedAt := time.Now()
	expireAt := finishedAt.Add(bmsHistoryExportRetention)
	if err := db.Table("bms_history_export_jobs").
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":         "SUCCESS",
			"file_path":      filePath,
			"file_name":      fileName,
			"file_size":      fileSize,
			"file_expire_at": expireAt,
			"finished_at":    finishedAt,
			"updated_at":     finishedAt,
		}).Error; err != nil {
		return err
	}

	if global.BMSHistoryExportWSManager != nil {
		wsMsg := model.BMSHistoryExportWSMessage{
			Type:        "bms_history_export_complete",
			TaskID:      job.ID,
			DeviceID:    job.DeviceID,
			FileName:    fileName,
			DownloadURL: fmt.Sprintf("/api/v1/battery/history/export/download?task_id=%s", job.ID),
			FinishedAt:  finishedAt.UnixMilli(),
		}
		if err := global.BMSHistoryExportWSManager.PublishToUser(job.CreatorUser, wsMsg); err != nil {
			logrus.WithError(err).WithField("job_id", job.ID).Warn("push bms history export ws notify failed")
		}
	}

	return nil
}

func (b *Battery) executeBMSHistoryExport(ctx context.Context, job *bmsHistoryExportJob) (string, string, error) {
	templateID, err := getBMSHistoryDeviceTemplateID(ctx, job.DeviceID, job.TenantID)
	if err != nil {
		return "", "", err
	}

	exportDir := filepath.Join(".", "files", "bms-history-export")
	if err := os.MkdirAll(exportDir, os.ModePerm); err != nil {
		return "", "", err
	}

	fileName := fmt.Sprintf("bms_history_%s_%s.xlsx", job.ViewMode, time.Now().Format("20060102150405"))
	filePath := filepath.ToSlash(filepath.Join(exportDir, fileName))

	switch job.ViewMode {
	case "long":
		if err := exportBMSHistoryLongExcel(ctx, filePath, templateID, job); err != nil {
			return "", "", err
		}
	case "wide":
		if err := exportBMSHistoryWideExcel(ctx, filePath, templateID, job); err != nil {
			return "", "", err
		}
	default:
		return "", "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "不支持的导出视图"})
	}

	return filePath, fileName, nil
}

func exportBMSHistoryLongExcel(ctx context.Context, filePath, templateID string, job *bmsHistoryExportJob) error {
	mergedSQL := buildBMSHistoryMergedSQL()
	mergedArgs := buildBMSHistoryMergedArgs(templateID, job.DeviceID, job.TenantID, job.StartTimeMs, job.EndTimeMs)
	querySQL := `
		SELECT ts_ms, data_type, data_identifier, data_name, value
		  FROM (` + mergedSQL + `) merged
		 ORDER BY ts_ms DESC, data_type ASC, data_identifier ASC
	`

	rows := make([]bmsHistoryMergedRow, 0)
	if err := global.DB.WithContext(ctx).Raw(querySQL, mergedArgs...).Scan(&rows).Error; err != nil {
		return err
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetCellValue(sheet, "A1", "时间")
	f.SetCellValue(sheet, "B1", "数据类型")
	f.SetCellValue(sheet, "C1", "标识符")
	f.SetCellValue(sheet, "D1", "数据名称")
	f.SetCellValue(sheet, "E1", "数值")

	for i, row := range rows {
		idx := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", idx), formatBMSHistoryTime(row.TSMs))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", idx), row.DataType)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", idx), row.DataIdentifier)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", idx), row.DataName)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", idx), row.Value)
	}

	return f.SaveAs(filePath)
}

func exportBMSHistoryWideExcel(ctx context.Context, filePath, templateID string, job *bmsHistoryExportJob) error {
	mergedSQL := buildBMSHistoryMergedSQL()
	mergedArgs := buildBMSHistoryMergedArgs(templateID, job.DeviceID, job.TenantID, job.StartTimeMs, job.EndTimeMs)

	columnsSQL := `
		SELECT DISTINCT data_type, data_identifier, data_name
		  FROM (` + mergedSQL + `) merged
		 ORDER BY data_type ASC, data_identifier ASC
	`
	columnRows := make([]bmsHistoryWideColumnRow, 0)
	if err := global.DB.WithContext(ctx).Raw(columnsSQL, mergedArgs...).Scan(&columnRows).Error; err != nil {
		return err
	}

	cellSQL := `
		SELECT ts_ms, data_type, data_identifier, value
		  FROM (` + mergedSQL + `) merged
		 ORDER BY ts_ms DESC, data_type ASC, data_identifier ASC
	`
	cells := make([]bmsHistoryWideCellRow, 0)
	if err := global.DB.WithContext(ctx).Raw(cellSQL, mergedArgs...).Scan(&cells).Error; err != nil {
		return err
	}

	valueRows := make([]bmsHistoryWideValueRow, 0, len(cells))
	for _, cell := range cells {
		valueRows = append(valueRows, bmsHistoryWideValueRow{
			DataType:       cell.DataType,
			DataIdentifier: cell.DataIdentifier,
			Value:          cell.Value,
		})
	}
	columnRows = filterBMSHistoryWideColumnRows(columnRows, valueRows)

	timeMap := make(map[int64]map[string]string)
	timeOrder := make([]int64, 0)
	seen := make(map[int64]struct{})
	for _, cell := range cells {
		if _, ok := seen[cell.TSMs]; !ok {
			seen[cell.TSMs] = struct{}{}
			timeOrder = append(timeOrder, cell.TSMs)
			timeMap[cell.TSMs] = make(map[string]string)
		}
		if shouldExcludeBMSHistoryWideIdentifier(cell.DataIdentifier) || isBMSHistoryWideJSONValue(cell.Value) {
			continue
		}
		colKey := buildBMSHistoryWideColumnKey(cell.DataType, cell.DataIdentifier)
		timeMap[cell.TSMs][colKey] = cell.Value
	}
	sort.SliceStable(timeOrder, func(i, j int) bool { return timeOrder[i] > timeOrder[j] })

	f := excelize.NewFile()
	sheet := "Sheet1"
	f.SetCellValue(sheet, "A1", "时间")
	for i, col := range columnRows {
		colIndex := i + 2
		cellName, _ := excelize.CoordinatesToCellName(colIndex, 1)
		header := col.DataIdentifier
		if strings.TrimSpace(col.DataName) != "" {
			header = fmt.Sprintf("%s(%s)", col.DataName, col.DataIdentifier)
		}
		f.SetCellValue(sheet, cellName, header)
	}

	for rowIdx, ts := range timeOrder {
		excelRow := rowIdx + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", excelRow), formatBMSHistoryTime(ts))
		for colIdx, col := range columnRows {
			colKey := buildBMSHistoryWideColumnKey(col.DataType, col.DataIdentifier)
			cellName, _ := excelize.CoordinatesToCellName(colIdx+2, excelRow)
			f.SetCellValue(sheet, cellName, timeMap[ts][colKey])
		}
	}

	return f.SaveAs(filePath)
}

func (*Battery) ListBMSHistoryPendingExportJobs(ctx context.Context, req model.BMSHistoryExportPendingReq, claims *utils.UserClaims) (*model.BMSHistoryExportPendingResp, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	rows := make([]bmsHistoryExportPendingRow, 0)
	if err := global.DB.WithContext(ctx).Table("bms_history_export_jobs AS j").
		Select(`
			j.id AS task_id,
			j.device_id AS device_id,
			COALESCE(d.device_number, '') AS device_number,
			j.view_mode AS view_mode,
			j.start_time_ms AS start_time_ms,
			j.end_time_ms AS end_time_ms,
			COALESCE(j.file_name, '') AS file_name,
			j.finished_at AS finished_at
		`).
		Joins("LEFT JOIN devices AS d ON d.id = j.device_id").
		Where("j.tenant_id = ? AND j.creator_user_id = ?", claims.TenantID, claims.ID).
		Where("j.status = 'SUCCESS'").
		Where("j.downloaded_at IS NULL").
		Where("j.file_path IS NOT NULL AND j.file_path <> ''").
		Where("(j.file_expire_at IS NULL OR j.file_expire_at > ?)", time.Now()).
		Order("j.finished_at DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.BMSHistoryExportJobItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, model.BMSHistoryExportJobItem{
			TaskID:       row.TaskID,
			DeviceID:     row.DeviceID,
			DeviceNumber: row.DeviceNumber,
			ViewMode:     row.ViewMode,
			StartTime:    row.StartTimeMs,
			EndTime:      row.EndTimeMs,
			FileName:     row.FileName,
			DownloadURL:  fmt.Sprintf("/api/v1/battery/history/export/download?task_id=%s", row.TaskID),
			FinishedAt:   row.FinishedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &model.BMSHistoryExportPendingResp{List: list}, nil
}

func (*Battery) GetBMSHistoryExportDownloadInfo(ctx context.Context, taskID string, claims *utils.UserClaims) (string, string, error) {
	var job bmsHistoryExportJob
	if err := global.DB.WithContext(ctx).Table("bms_history_export_jobs").
		Where("id = ? AND tenant_id = ? AND creator_user_id = ?", taskID, claims.TenantID, claims.ID).
		Scan(&job).Error; err != nil {
		return "", "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if job.ID == "" {
		return "", "", errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "导出任务不存在"})
	}
	if job.Status != "SUCCESS" {
		return "", "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "导出任务尚未完成"})
	}
	if job.DownloadedAt != nil {
		return "", "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "导出任务已下载"})
	}
	if job.FileExpireAt != nil && job.FileExpireAt.Before(time.Now()) {
		return "", "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "导出文件已过期"})
	}
	if job.FilePath == nil || strings.TrimSpace(*job.FilePath) == "" {
		return "", "", errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "导出文件不存在"})
	}

	filePath := filepath.Clean(*job.FilePath)
	if _, err := os.Stat(filePath); err != nil {
		return "", "", errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "导出文件不存在"})
	}

	fileName := "bms_history_export.xlsx"
	if job.FileName != nil && strings.TrimSpace(*job.FileName) != "" {
		fileName = *job.FileName
	}
	return filePath, fileName, nil
}

func (*Battery) MarkBMSHistoryExportDownloaded(ctx context.Context, taskID string, claims *utils.UserClaims) error {
	now := time.Now()
	return global.DB.WithContext(ctx).Table("bms_history_export_jobs").
		Where("id = ? AND tenant_id = ? AND creator_user_id = ?", taskID, claims.TenantID, claims.ID).
		Where("downloaded_at IS NULL").
		Updates(map[string]any{
			"downloaded_at": now,
			"updated_at":    now,
		}).Error
}
