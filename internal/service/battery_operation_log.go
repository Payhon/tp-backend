package service

import (
	"context"
	"encoding/json"
	"time"

	"project/pkg/errcode"
	global "project/pkg/global"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// BatteryOperationType 运营日志操作类型（可扩展）
const (
	BatteryOpTypeCreate              = "CREATE"
	BatteryOpTypeImport              = "IMPORT"
	BatteryOpTypeFactoryOut          = "FACTORY_OUT"
	BatteryOpTypeFactoryRestore      = "FACTORY_RESTORE"
	BatteryOpTypePackFactoryReassign = "PACK_FACTORY_REASSIGN"
	BatteryOpTypeTransfer            = "TRANSFER"
	BatteryOpTypeRollback            = "ROLLBACK"
	BatteryOpTypeActivate            = "ACTIVATE"
	BatteryOpTypeInfoComplete        = "INFO_COMPLETE"
	BatteryOpTypeEditInfo            = "EDIT_INFO"
	BatteryOpTypeDelete              = "DELETE"
	BatteryOpTypeWarrantySubmit      = "WARRANTY_SUBMIT"
	BatteryOpTypeWarrantyHandle      = "WARRANTY_HANDLE"
	BatteryOpTypeMaintenanceSubmit   = "MAINTENANCE_SUBMIT"
	BatteryOpTypeMaintenanceHandle   = "MAINTENANCE_HANDLE"
)

type batteryOperationLogRow struct {
	ID            int64     `gorm:"column:id"`
	OccurredAt    time.Time `gorm:"column:occurred_at"`
	DeviceID      string    `gorm:"column:device_id"`
	DeviceNumber  string    `gorm:"column:device_number"`
	OperationType string    `gorm:"column:operation_type"`
	OperatorID    *string   `gorm:"column:operator_id"`
	OperatorName  *string   `gorm:"column:operator_name"`
	Description   *string   `gorm:"column:description"`
}

type batteryOperationLogCreateRow struct {
	TenantID      string         `gorm:"column:tenant_id"`
	DeviceID      string         `gorm:"column:device_id"`
	DeviceNumber  string         `gorm:"column:device_number"`
	OperationType string         `gorm:"column:operation_type"`
	OperatorID    *string        `gorm:"column:operator_id"`
	Description   *string        `gorm:"column:description"`
	Extra         datatypes.JSON `gorm:"column:extra"`
	OccurredAt    time.Time      `gorm:"column:occurred_at"`
}

// CreateBatteryOperationLog 写入电池运营日志
func CreateBatteryOperationLog(ctx context.Context, tenantID, deviceID, deviceNumber, operationType string, operatorID *string, description *string, extra any) error {
	return createBatteryOperationLogWithDB(global.DB.WithContext(ctx), tenantID, deviceID, deviceNumber, operationType, operatorID, description, extra)
}

func CreateBatteryOperationLogTx(tx *gorm.DB, tenantID, deviceID, deviceNumber, operationType string, operatorID *string, description *string, extra any) error {
	if tx == nil {
		return createBatteryOperationLogWithDB(global.DB, tenantID, deviceID, deviceNumber, operationType, operatorID, description, extra)
	}
	return createBatteryOperationLogWithDB(tx, tenantID, deviceID, deviceNumber, operationType, operatorID, description, extra)
}

func createBatteryOperationLogWithDB(db *gorm.DB, tenantID, deviceID, deviceNumber, operationType string, operatorID *string, description *string, extra any) error {
	extraJSON, err := marshalBatteryOperationLogExtra(extra)
	if err != nil {
		return err
	}
	row := batteryOperationLogCreateRow{
		TenantID:      tenantID,
		DeviceID:      deviceID,
		DeviceNumber:  deviceNumber,
		OperationType: operationType,
		OperatorID:    operatorID,
		Description:   description,
		Extra:         extraJSON,
		OccurredAt:    time.Now(),
	}
	return db.Table("battery_operation_logs").Create(&row).Error
}

func marshalBatteryOperationLogExtra(extra any) (datatypes.JSON, error) {
	if extra == nil {
		return datatypes.JSON([]byte(`{}`)), nil
	}
	switch value := extra.(type) {
	case datatypes.JSON:
		if len(value) == 0 {
			return datatypes.JSON([]byte(`{}`)), nil
		}
		if !json.Valid(value) {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "battery operation log extra is not valid JSON")
		}
		return value, nil
	case json.RawMessage:
		if len(value) == 0 {
			return datatypes.JSON([]byte(`{}`)), nil
		}
		if !json.Valid(value) {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "battery operation log extra is not valid JSON")
		}
		return datatypes.JSON(value), nil
	}

	raw, err := json.Marshal(extra)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "battery operation log extra is not JSON serializable"})
	}
	return datatypes.JSON(raw), nil
}

// GetBatteryOperationLogList 查询运营日志列表（分页）
func GetBatteryOperationLogList(ctx context.Context, tenantID string, orgID string, page, pageSize int, deviceID *string, deviceNumberLike *string, operationType *string, startTime, endTime *time.Time) (total int64, list []batteryOperationLogRow, err error) {
	db := global.DB.WithContext(ctx)

	q := db.Table("battery_operation_logs AS bol").
		Select(`
			bol.id,
			bol.occurred_at,
			bol.device_id,
			bol.device_number,
			bol.operation_type,
			bol.operator_id,
			u.name AS operator_name,
			bol.description
		`).
		Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = bol.device_id`).
		Joins(`LEFT JOIN users AS u ON u.id = bol.operator_id`).
		Where("bol.tenant_id = ?", tenantID)

	// 组织数据隔离：orgID 不为空时只看子树内设备
	if orgID != "" {
		q = q.Where(`dbat.owner_org_id IN (
			SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
		)`, tenantID, orgID)
	}

	if deviceID != nil && *deviceID != "" {
		q = q.Where("bol.device_id = ?", *deviceID)
	}
	if deviceNumberLike != nil && *deviceNumberLike != "" {
		q = q.Where("bol.device_number ILIKE ?", "%"+*deviceNumberLike+"%")
	}
	if operationType != nil && *operationType != "" {
		q = q.Where("bol.operation_type = ?", *operationType)
	}
	if startTime != nil {
		q = q.Where("bol.occurred_at >= ?", *startTime)
	}
	if endTime != nil {
		q = q.Where("bol.occurred_at <= ?", *endTime)
	}

	if err := q.Count(&total).Error; err != nil {
		return 0, nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (page - 1) * pageSize
	rows := make([]batteryOperationLogRow, 0, pageSize)
	if err := q.Order("bol.occurred_at DESC, bol.id DESC").Offset(offset).Limit(pageSize).Scan(&rows).Error; err != nil {
		return 0, nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return total, rows, nil
}
