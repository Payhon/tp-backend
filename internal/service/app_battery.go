package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"project/internal/dal"
	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// AppBattery APP端：电池设备详情/透传（仅提供基础数据）
type AppBattery struct{}

type appBatteryDetailRow struct {
	DeviceID     string  `gorm:"column:device_id"`
	DeviceNumber string  `gorm:"column:device_number"`
	DeviceName   *string `gorm:"column:device_name"`

	BmsCommType *int `gorm:"column:bms_comm_type"`

	BatteryModelID   *string `gorm:"column:battery_model_id"`
	BatteryModelName *string `gorm:"column:battery_model_name"`

	ItemUUID   *string `gorm:"column:item_uuid"`
	BleMac     *string `gorm:"column:ble_mac"`
	CommChipID *string `gorm:"column:comm_chip_id"`

	Soc       *float64   `gorm:"column:soc"`
	Soh       *float64   `gorm:"column:soh"`
	DbUpdated *time.Time `gorm:"column:db_updated_at"`

	IsOnline      int16   `gorm:"column:is_online"`
	CurrentVer    *string `gorm:"column:current_version"`
	DeviceRemark1 *string `gorm:"column:remark1"`
}

type appBatteryOtaCheckRow struct {
	DeviceID         string  `gorm:"column:device_id"`
	CurrentVersion   *string `gorm:"column:current_version"`
	DeviceConfigID   *string `gorm:"column:device_config_id"`
	TenantID         *string `gorm:"column:tenant_id"`
	BatteryModelID   *string `gorm:"column:battery_model_id"`
	BatteryModelName *string `gorm:"column:battery_model_name"`
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
			dbat.bms_comm_type AS bms_comm_type,
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
		DeviceID:         row.DeviceID,
		DeviceNumber:     row.DeviceNumber,
		DeviceName:       row.DeviceName,
		BmsCommType:      row.BmsCommType,
		BatteryModelID:   row.BatteryModelID,
		BatteryModelName: row.BatteryModelName,
		ItemUUID:         row.ItemUUID,
		BleMac:           row.BleMac,
		CommChipID:       row.CommChipID,
		Soc:              row.Soc,
		Soh:              row.Soh,
		UpdatedAt:        updatedAt,
		IsOnline:         row.IsOnline,
		FwVersion:        row.CurrentVer,
		Remark:           row.DeviceRemark1,
	}, nil
}

// GetBatteryMqttCredentialForApp 获取APP端直连MQTT所需凭证（要求设备已绑定）
func (*AppBattery) GetBatteryMqttCredentialForApp(ctx context.Context, deviceID string, claims *utils.UserClaims) (*model.AppBatteryMqttCredentialResp, error) {
	if deviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device_id is required")
	}
	if claims == nil || claims.ID == "" || claims.TenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "claims is required")
	}

	// 复用绑定校验逻辑（管理员允许跨设备查看，仍受 tenant 约束）
	if _, err := new(AppBattery).GetBatteryDetailForApp(ctx, deviceID, claims); err != nil {
		return nil, err
	}

	device, err := dal.GetDeviceByID(deviceID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if device.TenantID != claims.TenantID {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device not in current tenant")
	}

	voucher := strings.TrimSpace(device.Voucher)
	if voucher == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device voucher is empty")
	}
	var dv model.DeviceVoucher
	if err := json.Unmarshal([]byte(voucher), &dv); err != nil {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device voucher is not valid json")
	}
	if strings.TrimSpace(dv.Username) == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device voucher username is empty")
	}

	wsURL, err := GetDictValueByConfigKey("mqtt.ws_address", claims.TenantID)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"err": err.Error()})
	}
	wsURL = strings.TrimSpace(wsURL)
	if wsURL == "" {
		wsURL = strings.TrimSpace(viper.GetString("mqtt.ws_address"))
	}
	if wsURL == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "mqtt.ws_address is not configured")
	}

	writeTopic := "device/socket/rx/" + deviceID
	readTopic := "device/socket/tx/" + deviceID

	return &model.AppBatteryMqttCredentialResp{
		DeviceID:   deviceID,
		WsURL:      wsURL,
		Username:   dv.Username,
		Password:   dv.Password,
		WriteTopic: writeTopic,
		ReadTopic:  readTopic,
	}, nil
}

// CheckBatteryOtaForApp APP端OTA升级检查（根据设备配置匹配升级包）
func (*AppBattery) CheckBatteryOtaForApp(ctx context.Context, deviceID string, claims *utils.UserClaims) (*model.AppBatteryOtaCheckResp, error) {
	if deviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device_id is required")
	}
	if claims == nil || claims.ID == "" || claims.TenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "claims is required")
	}

	// 复用绑定校验逻辑（管理员允许跨设备查看，仍受 tenant 约束）
	if _, err := new(AppBattery).GetBatteryDetailForApp(ctx, deviceID, claims); err != nil {
		return nil, err
	}

	var row appBatteryOtaCheckRow
	err := global.DB.WithContext(ctx).
		Table("devices AS d").
		Select(`
			d.id AS device_id,
			d.current_version AS current_version,
			d.tenant_id AS tenant_id,
			dbat.battery_model_id AS battery_model_id,
			bm.name AS battery_model_name,
			bm.device_config_id AS device_config_id
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

	resp := &model.AppBatteryOtaCheckResp{
		DeviceID:       row.DeviceID,
		NeedUpgrade:    false,
		CurrentVersion: row.CurrentVersion,
	}

	if row.DeviceConfigID == nil || strings.TrimSpace(*row.DeviceConfigID) == "" {
		return resp, nil
	}

	var pkg model.OtaUpgradePackage
	pkgQuery := global.DB.WithContext(ctx).Table(model.TableNameOtaUpgradePackage).
		Where("device_config_id = ?", strings.TrimSpace(*row.DeviceConfigID))
	// 允许租户级或公共包（tenant_id 为 NULL）
	pkgQuery = pkgQuery.Where("tenant_id = ? OR tenant_id IS NULL", claims.TenantID)
	if err := pkgQuery.Order("created_at DESC").Limit(1).Take(&pkg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return resp, nil
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	currentVer := ""
	if row.CurrentVersion != nil {
		currentVer = strings.TrimSpace(*row.CurrentVersion)
	}
	targetVer := ""
	if pkg.TargetVersion != nil {
		targetVer = strings.TrimSpace(*pkg.TargetVersion)
	}
	if targetVer != "" {
		resp.NeedUpgrade = !strings.EqualFold(currentVer, targetVer)
	} else {
		resp.NeedUpgrade = !strings.EqualFold(currentVer, pkg.Version)
	}

	// 若无需升级，返回版本信息但不携带下载地址
	if !resp.NeedUpgrade {
		resp.Version = &pkg.Version
		resp.TargetVersion = pkg.TargetVersion
		resp.PackageID = &pkg.ID
		resp.PackageType = &pkg.PackageType
		return resp, nil
	}

	firmwareURL := buildOtaDownloadURL(pkg.PackageURL)
	resp.Version = &pkg.Version
	resp.TargetVersion = pkg.TargetVersion
	resp.FirmwareURL = firmwareURL
	resp.PackageID = &pkg.ID
	resp.PackageType = &pkg.PackageType
	resp.SignatureType = pkg.SignatureType
	resp.Signature = pkg.Signature
	resp.Module = pkg.Module
	resp.AdditionalInfo = pkg.AdditionalInfo
	resp.Remark = pkg.Remark

	return resp, nil
}

func buildOtaDownloadURL(packageURL *string) *string {
	if packageURL == nil || strings.TrimSpace(*packageURL) == "" {
		return nil
	}
	raw := strings.TrimSpace(*packageURL)
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return &raw
	}
	base := strings.TrimSpace(global.OtaAddress)
	if base == "" {
		return &raw
	}
	url := base + strings.TrimPrefix(raw, ".")
	return &url
}
