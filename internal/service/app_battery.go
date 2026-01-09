package service

import (
	"context"
	"strings"
	"time"

	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"gorm.io/gorm"
)

// AppBattery APP端：电池设备详情/透传（仅提供基础数据）
type AppBattery struct{}

type appBatteryDetailRow struct {
	DeviceID     string  `gorm:"column:device_id"`
	DeviceNumber string  `gorm:"column:device_number"`
	DeviceName   *string `gorm:"column:device_name"`

	BatteryModelID   *string `gorm:"column:battery_model_id"`
	BatteryModelName *string `gorm:"column:battery_model_name"`

	ItemUUID   *string  `gorm:"column:item_uuid"`
	BleMac     *string  `gorm:"column:ble_mac"`
	CommChipID *string  `gorm:"column:comm_chip_id"`

	Soc       *float64   `gorm:"column:soc"`
	Soh       *float64   `gorm:"column:soh"`
	DbUpdated *time.Time `gorm:"column:db_updated_at"`

	IsOnline      int16   `gorm:"column:is_online"`
	CurrentVer    *string `gorm:"column:current_version"`
	DeviceRemark1 *string `gorm:"column:remark1"`
}

// GetBatteryDetailForApp 获取APP端电池设备详情（要求设备已绑定到当前用户）
func (*AppBattery) GetBatteryDetailForApp(ctx context.Context, deviceID string, claims *utils.UserClaims) (*model.AppBatteryDetailResp, error) {
	if deviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device_id is required")
	}
	if claims == nil || claims.ID == "" || claims.TenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "claims is required")
	}

	// 终端用户默认要求绑定；管理员允许跨设备查看（仍受 tenant 约束）
	isAdmin := strings.Contains(strings.ToUpper(claims.Authority), "ADMIN")
	if !isAdmin {
		q := query.Use(global.DB)
		if _, err := q.DeviceUserBinding.WithContext(ctx).
			Where(
				q.DeviceUserBinding.DeviceID.Eq(deviceID),
				q.DeviceUserBinding.UserID.Eq(claims.ID),
			).First(); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errcode.NewWithMessage(errcode.CodeParamError, "device not bound to current user")
			}
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
	}

	var row appBatteryDetailRow
	err := global.DB.WithContext(ctx).
		Table("devices AS d").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			d.name AS device_name,
			d.is_online AS is_online,
			d.current_version AS current_version,
			d.remark1 AS remark1,
			dbat.battery_model_id AS battery_model_id,
			bm.name AS battery_model_name,
			dbat.item_uuid AS item_uuid,
			dbat.ble_mac AS ble_mac,
			dbat.comm_chip_id AS comm_chip_id,
			dbat.soc AS soc,
			dbat.soh AS soh,
			dbat.updated_at AS db_updated_at
		`).
		Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id`).
		Joins(`LEFT JOIN battery_models AS bm ON bm.id = dbat.battery_model_id`).
		Where("d.id = ? AND d.tenant_id = ?", deviceID, claims.TenantID).
		Scan(&row).Error
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if row.DeviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device not found")
	}

	var updatedAt *string
	if row.DbUpdated != nil {
		s := row.DbUpdated.Local().Format("2006-01-02 15:04:05")
		updatedAt = &s
	}

	return &model.AppBatteryDetailResp{
		DeviceID:          row.DeviceID,
		DeviceNumber:      row.DeviceNumber,
		DeviceName:        row.DeviceName,
		BatteryModelID:    row.BatteryModelID,
		BatteryModelName:  row.BatteryModelName,
		ItemUUID:          row.ItemUUID,
		BleMac:            row.BleMac,
		CommChipID:        row.CommChipID,
		Soc:               row.Soc,
		Soh:               row.Soh,
		UpdatedAt:         updatedAt,
		IsOnline:          row.IsOnline,
		FwVersion:         row.CurrentVer,
		Remark:            row.DeviceRemark1,
	}, nil
}
