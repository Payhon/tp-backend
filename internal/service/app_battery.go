package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"project/internal/bms/protocol"
	"project/internal/dal"
	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	ItemUUID           *string    `gorm:"column:item_uuid"`
	BatchNumber        *string    `gorm:"column:batch_number"`
	ProductSpec        *string    `gorm:"column:product_spec"`
	OrderNumber        *string    `gorm:"column:order_number"`
	BleMac             *string    `gorm:"column:ble_mac"`
	IdentityBleMac     *string    `gorm:"column:identity_ble_mac"`
	CommChipID         *string    `gorm:"column:comm_chip_id"`
	ProductionDate     *time.Time `gorm:"column:production_date"`
	WarrantyExpireDate *time.Time `gorm:"column:warranty_expire_date"`

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
	TenantID         *string `gorm:"column:tenant_id"`
	BatteryModelID   *string `gorm:"column:battery_model_id"`
	BatteryModelName *string `gorm:"column:battery_model_name"`
	BatchNumber      *string `gorm:"column:batch_number"`
	ItemUUID         *string `gorm:"column:item_uuid"`
}

const (
	appBatterySnapshotKey      = "bms.snapshot"
	appBatterySnapshotMaxBytes = 64 * 1024
	appBatteryStatusOnline     = int16(1)
	appBatteryStatusOffline    = int16(0)
)

var appBatteryCoreKeyWhitelist = map[string]struct{}{
	"soc":                   {},
	"soh":                   {},
	"packCellSumVoltageV":   {},
	"vPackV":                {},
	"currentA":              {},
	"avgCellVoltageMv":      {},
	"highestCellVoltageMv":  {},
	"lowestCellVoltageMv":   {},
	"maxCellVoltageDiffMv":  {},
	"chargeMosC":            {},
	"dischargeMosC":         {},
	"ambientC":              {},
	"cycleCount":            {},
	"chargeRemainingMin":    {},
	"dischargeRemainingMin": {},
	"chargeFetOn":           {},
	"dischargeFetOn":        {},
	"charging":              {},
	"discharging":           {},
	"balancingOn":           {},
	"protectOn":             {},
	"alarmCount":            {},
	"protectCount":          {},
	"faultCount":            {},
	"seriesCount":           {},
}

var appBatteryCurrentTelemetryKeys = []string{
	"soc",
	"soh",
	"packCellSumVoltageV",
	"vPackV",
	"currentA",
	"avgCellVoltageMv",
	"highestCellVoltageMv",
	"lowestCellVoltageMv",
	"maxCellVoltageDiffMv",
	"chargeMosC",
	"dischargeMosC",
	"ambientC",
	"cycleCount",
	"chargeRemainingMin",
	"dischargeRemainingMin",
	"chargeFetOn",
	"dischargeFetOn",
	"charging",
	"discharging",
	"balancingOn",
	"protectOn",
	"alarmCount",
	"protectCount",
	"faultCount",
	"seriesCount",
	"meta.seriesCount",
	"meta.cellTempCount",
	"electrical.packCellSumVoltageV",
	"electrical.vPackV",
	"electrical.currentA",
	"electrical.avgCellVoltageMv",
	"electrical.highestCellVoltageMv",
	"electrical.lowestCellVoltageMv",
	"electrical.maxCellVoltageDiffMv",
	"electrical.cellVoltageIndex.highest",
	"electrical.cellVoltageIndex.lowest",
	"temperature.chargeMosC",
	"temperature.dischargeMosC",
	"temperature.ambientC",
	"temperature.cellTempsC",
	"cell.voltagesMv",
	"cell.balancing",
	appBatterySnapshotKey,
}

func canAccessOrgDevice(ctx context.Context, tenantID, userID, deviceID string) (bool, error) {
	orgID, err := getUserOrgID(userID)
	if err != nil {
		return false, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_user_org_id",
			"user_id":   userID,
			"error":     err.Error(),
		})
	}

	orgType := ""
	if userID != "" {
		if t, ok, err := GroupApp.OrgTypePermission.GetUserOrgType(ctx, tenantID, userID); err != nil {
			return false, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_user_org_type",
				"user_id":   userID,
				"error":     err.Error(),
			})
		} else if ok {
			orgType = strings.TrimSpace(t)
		}
	}

	isFactory := orgType == model.OrgTypeBMSFactory || strings.TrimSpace(orgID) == ""
	if isFactory {
		return true, nil
	}

	var ownerOrgID *string
	if err := global.DB.WithContext(ctx).
		Table("device_batteries AS dbat").
		Select("dbat.owner_org_id").
		Joins("JOIN devices AS d ON d.id = dbat.device_id").
		Where("d.id = ? AND d.tenant_id = ?", deviceID, tenantID).
		Limit(1).
		Scan(&ownerOrgID).Error; err != nil {
		return false, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_device_owner_org",
			"device_id": deviceID,
			"error":     err.Error(),
		})
	}

	if ownerOrgID == nil || strings.TrimSpace(*ownerOrgID) == "" {
		return false, nil
	}

	return canAccessOrg(tenantID, orgID, strings.TrimSpace(*ownerOrgID)), nil
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
		userKind, err := GroupApp.OrgTypePermission.GetUserKind(ctx, claims.TenantID, claims.ID)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_user_kind",
				"user_id":   claims.ID,
				"error":     err.Error(),
			})
		}
		if userKind == model.UserKindOrgUser {
			ok, err := canAccessOrgDevice(ctx, claims.TenantID, claims.ID, deviceID)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, errcode.New(errcode.CodeNoPermission)
			}
		} else {
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
			dbat.batch_number AS batch_number,
			dbat.product_spec AS product_spec,
			dbat.order_number AS order_number,
			dbat.ble_mac AS ble_mac,
			dbat.identity_ble_mac AS identity_ble_mac,
			dbat.comm_chip_id AS comm_chip_id,
			dbat.production_date AS production_date,
			dbat.warranty_expire_date AS warranty_expire_date,
			dbat.soc AS soc,
			dbat.soh AS soh,
			dbat.updated_at AS db_updated_at
		`).
		Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id`).
		Joins(`LEFT JOIN `+model.TableNameBatteryModel+` AS bm ON bm.id = dbat.battery_model_id`).
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
	var productionDate *string
	if row.ProductionDate != nil {
		s := row.ProductionDate.Local().Format("2006-01-02")
		productionDate = &s
	}
	var warrantyExpireDate *string
	if row.WarrantyExpireDate != nil {
		s := row.WarrantyExpireDate.Local().Format("2006-01-02")
		warrantyExpireDate = &s
	}

	return &model.AppBatteryDetailResp{
		DeviceID:           row.DeviceID,
		DeviceNumber:       row.DeviceNumber,
		DeviceName:         row.DeviceName,
		BmsCommType:        row.BmsCommType,
		BatteryModelID:     row.BatteryModelID,
		BatteryModelName:   row.BatteryModelName,
		ItemUUID:           row.ItemUUID,
		BatchNumber:        row.BatchNumber,
		ProductSpec:        row.ProductSpec,
		OrderNumber:        row.OrderNumber,
		BleMac:             row.BleMac,
		IdentityBleMac:     row.IdentityBleMac,
		CommChipID:         row.CommChipID,
		ProductionDate:     productionDate,
		WarrantyExpireDate: warrantyExpireDate,
		Soc:                row.Soc,
		Soh:                row.Soh,
		UpdatedAt:          updatedAt,
		IsOnline:           row.IsOnline,
		FwVersion:          row.CurrentVer,
		Remark:             row.DeviceRemark1,
	}, nil
}

// GetBatteryCurrentTelemetryForApp 获取APP端电池当前遥测（要求设备已绑定到当前用户）。
func (*AppBattery) GetBatteryCurrentTelemetryForApp(ctx context.Context, deviceID string, claims *utils.UserClaims) (*model.AppBatteryCurrentTelemetryResp, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device_id is required")
	}

	detail, err := new(AppBattery).GetBatteryDetailForApp(ctx, deviceID, claims)
	if err != nil {
		return nil, err
	}

	rows, err := dal.GetCurrentTelemetryDataEvolutionByKeys(deviceID, appBatteryCurrentTelemetryKeys)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_app_battery_current_telemetry",
			"device_id": deviceID,
			"error":     err.Error(),
		})
	}

	current := make(map[string]model.AppBatteryCurrentTelemetryValue, len(rows))
	var snapshot map[string]interface{}
	var lastReportTs int64
	for _, row := range rows {
		if row == nil {
			continue
		}
		value := telemetryCurrentValue(row)
		ts := row.T.UnixMilli()
		if ts > lastReportTs {
			lastReportTs = ts
		}
		current[row.Key] = model.AppBatteryCurrentTelemetryValue{
			Value: value,
			Ts:    ts,
		}
		if row.Key == appBatterySnapshotKey {
			if raw, ok := value.(string); ok && strings.TrimSpace(raw) != "" {
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
					snapshot = parsed
				}
			}
		}
	}
	snapshot = mergeCurrentTelemetryIntoAppBatterySnapshot(snapshot, current)

	return &model.AppBatteryCurrentTelemetryResp{
		DeviceID:     deviceID,
		IsOnline:     detail.IsOnline,
		LastReportTs: lastReportTs,
		Current:      current,
		Snapshot:     snapshot,
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

// CheckBatteryOtaForApp APP端OTA升级检查（按租户与升级约束匹配升级包）
func (*AppBattery) CheckBatteryOtaForApp(ctx context.Context, req model.AppBatteryOtaCheckReq, claims *utils.UserClaims) (*model.AppBatteryOtaCheckResp, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	if claims == nil || claims.TenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "claims is required")
	}

	var row appBatteryOtaCheckRow

	if deviceID != "" {
		err := global.DB.WithContext(ctx).
			Table("devices AS d").
			Select(`
				d.id AS device_id,
				d.current_version AS current_version,
				d.tenant_id AS tenant_id,
				dbat.battery_model_id AS battery_model_id,
				bm.name AS battery_model_name,
				dbat.batch_number AS batch_number,
				dbat.item_uuid AS item_uuid
			`).
			Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id`).
			Joins(`LEFT JOIN `+model.TableNameBatteryModel+` AS bm ON bm.id = dbat.battery_model_id`).
			Where("d.id = ? AND d.tenant_id = ?", deviceID, claims.TenantID).
			Scan(&row).Error
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if row.DeviceID == "" {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "device not found")
		}
	}

	resp := &model.AppBatteryOtaCheckResp{
		DeviceID:       row.DeviceID,
		NeedUpgrade:    false,
		CurrentVersion: row.CurrentVersion,
	}

	var packages []model.OtaUpgradePackage
	pkgQuery := global.DB.WithContext(ctx).Table(model.TableNameOtaUpgradePackage).
		Where("device_kind = ?", model.OTADeviceKindBMS)
	// 允许租户级或公共包（tenant_id 为 NULL）
	pkgQuery = pkgQuery.Where("tenant_id = ? OR tenant_id IS NULL", claims.TenantID)
	if err := pkgQuery.Order("created_at DESC").Find(&packages).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if len(packages) == 0 {
		return resp, nil
	}

	reportedVer := ""
	if req.Version != nil {
		reportedVer = strings.TrimSpace(*req.Version)
	}
	if reportedVer == "" && row.CurrentVersion != nil {
		reportedVer = strings.TrimSpace(*row.CurrentVersion)
	}
	resp.CurrentVersion = stringPtrOrNil(reportedVer)

	criteria := appBatteryOtaMatchCriteria{
		BatteryModelID: firstTrimmed(req.BatteryModelID, row.BatteryModelID),
		BatchNumber:    firstTrimmed(req.BatchNumber, row.BatchNumber),
		ItemUUID:       firstTrimmed(req.ItemUUID, row.ItemUUID),
	}
	selected := selectAppBatteryOtaPackage(packages, reportedVer, criteria)
	if selected == nil {
		return resp, nil
	}
	resp.NeedUpgrade = true

	firmwareURL := buildOtaDownloadURL(selected.PackageURL)
	resp.Version = &selected.Version
	resp.TargetVersion = selected.TargetVersion
	resp.FirmwareURL = firmwareURL
	resp.PackageID = &selected.ID
	resp.PackageType = &selected.PackageType
	resp.SignatureType = selected.SignatureType
	resp.Signature = selected.Signature
	resp.Module = selected.Module
	resp.AdditionalInfo = selected.AdditionalInfo
	resp.Remark = selected.Remark

	return resp, nil
}

func resolveAppBatteryTenantID(ctx context.Context, claims *utils.UserClaims, tenantHeader string) (string, error) {
	tid := strings.TrimSpace(tenantHeader)
	claimsTenantID := ""
	if claims != nil {
		claimsTenantID = strings.TrimSpace(claims.TenantID)
	}
	if tid != "" && claimsTenantID != "" && tid != claimsTenantID {
		logrus.WithFields(logrus.Fields{
			"header_tenant": tid,
			"claims_tenant": claimsTenantID,
			"operation":     "resolve_app_battery_tenant_id",
		}).Warn("app battery tenant header mismatched with claims, prefer claims tenant")
	}
	if claimsTenantID != "" {
		return claimsTenantID, nil
	}
	if tid != "" {
		return tid, nil
	}
	return resolveTenantID(ctx, "")
}

func (*AppBattery) GetMeterOtaPackagesForApp(ctx context.Context, claims *utils.UserClaims, tenantHeader string) ([]model.AppBatteryMeterOtaPackageResp, error) {
	resolvedTenantID, err := resolveAppBatteryTenantID(ctx, claims, tenantHeader)
	if err != nil {
		return nil, err
	}

	var packages []model.OtaUpgradePackage
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameOtaUpgradePackage).
		Where("(tenant_id = ? OR tenant_id IS NULL)", resolvedTenantID).
		Where("device_kind = ?", model.OTADeviceKindMeter).
		Order("created_at DESC").
		Find(&packages).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	resp := make([]model.AppBatteryMeterOtaPackageResp, 0, len(packages))
	for _, pkg := range packages {
		resp = append(resp, model.AppBatteryMeterOtaPackageResp{
			ID:          pkg.ID,
			Name:        pkg.Name,
			Description: pkg.Description,
			PackageURL:  pkg.PackageURL,
		})
	}

	logrus.WithFields(logrus.Fields{
		"tenant_id":     resolvedTenantID,
		"header_tenant": strings.TrimSpace(tenantHeader),
		"claims_tenant": func() string {
			if claims == nil {
				return ""
			}
			return strings.TrimSpace(claims.TenantID)
		}(),
		"package_count":   len(resp),
		"device_kind":     model.OTADeviceKindMeter,
		"query_operation": "app_meter_ota_packages",
	}).Info("app battery meter ota packages fetched")

	return resp, nil
}

// ReportBatteryDataForApp APP端BMS上报（BLE 经 App 上云）
func (*AppBattery) ReportBatteryDataForApp(ctx context.Context, req model.AppBatteryReportReq, claims *utils.UserClaims) (*model.AppBatteryReportResp, error) {
	if claims == nil || claims.ID == "" || claims.TenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "claims is required")
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device_id is required")
	}
	if req.Ts <= 0 {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "ts must be a valid millisecond timestamp")
	}
	if len(req.Core) == 0 {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "core is required")
	}

	connType := strings.ToLower(strings.TrimSpace(req.ConnType))
	if connType == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "conn_type is required")
	}
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if platform == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "platform is required")
	}

	enabled := true
	if viper.IsSet("bms.app_report.enabled") {
		enabled = viper.GetBool("bms.app_report.enabled")
	}
	if !enabled {
		return &model.AppBatteryReportResp{
			DeviceID:      deviceID,
			Ts:            req.Ts,
			Accepted:      false,
			IgnoredReason: "feature_disabled",
		}, nil
	}

	bluetoothOnly := true
	if viper.IsSet("bms.app_report.bluetooth_only") {
		bluetoothOnly = viper.GetBool("bms.app_report.bluetooth_only")
	}
	if bluetoothOnly && connType != "bluetooth" {
		return &model.AppBatteryReportResp{
			DeviceID:      deviceID,
			Ts:            req.Ts,
			Accepted:      false,
			IgnoredReason: "bluetooth_only",
		}, nil
	}

	// 复用现有APP设备权限校验（绑定/组织/租户）
	batteryDetail, err := new(AppBattery).GetBatteryDetailForApp(ctx, deviceID, claims)
	if err != nil {
		return nil, err
	}

	coreValues, err := normalizeAppBatteryCore(req.Core)
	if err != nil {
		return nil, err
	}

	var snapshotRaw *string
	if req.Snapshot != nil {
		s, err := normalizeAppBatterySnapshot(req.Snapshot)
		if err != nil {
			return nil, err
		}
		snapshotRaw = &s
	}

	reportTs := req.Ts
	reportAt := time.UnixMilli(reportTs).UTC()
	tenantID := claims.TenantID

	historyRows := make([]model.TelemetryData, 0, len(coreValues)+1)
	currentRows := make([]model.TelemetryCurrentData, 0, len(coreValues)+1)
	wsPayload := make(map[string]interface{}, len(coreValues)+1)

	for key, value := range coreValues {
		boolV, numberV, stringV := toTelemetryColumns(value)
		historyRows = append(historyRows, model.TelemetryData{
			DeviceID: deviceID,
			Key:      key,
			T:        reportTs,
			BoolV:    boolV,
			NumberV:  numberV,
			StringV:  stringV,
			TenantID: &tenantID,
		})
		currentRows = append(currentRows, model.TelemetryCurrentData{
			DeviceID: deviceID,
			Key:      key,
			T:        reportAt,
			BoolV:    boolV,
			NumberV:  numberV,
			StringV:  stringV,
			TenantID: &tenantID,
		})
		wsPayload[key] = value
	}

	if snapshotRaw != nil {
		historyRows = append(historyRows, model.TelemetryData{
			DeviceID: deviceID,
			Key:      appBatterySnapshotKey,
			T:        reportTs,
			StringV:  snapshotRaw,
			TenantID: &tenantID,
		})
		currentRows = append(currentRows, model.TelemetryCurrentData{
			DeviceID: deviceID,
			Key:      appBatterySnapshotKey,
			T:        reportAt,
			StringV:  snapshotRaw,
			TenantID: &tenantID,
		})
		wsPayload[appBatterySnapshotKey] = *snapshotRaw
	}

	err = global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(historyRows) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "device_id"}, {Name: "key"}, {Name: "ts"}},
				DoNothing: true,
			}).Create(&historyRows).Error; err != nil {
				return err
			}
		}

		if len(currentRows) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "device_id"}, {Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"ts", "bool_v", "number_v", "string_v", "tenant_id",
				}),
				Where: clause.Where{
					Exprs: []clause.Expression{clause.Expr{SQL: "telemetry_current_datas.ts <= EXCLUDED.ts"}},
				},
			}).Create(&currentRows).Error; err != nil {
				return err
			}
		}

		var soc *float64
		var soh *float64
		var identityBleMac *string
		if v, ok := coreValues["soc"]; ok {
			if n, ok := toFloat64(v); ok {
				soc = &n
			}
		}
		if v, ok := coreValues["soh"]; ok {
			if n, ok := toFloat64(v); ok {
				soh = &n
			}
		}
		if req.Snapshot != nil {
			if identity, ok := req.Snapshot["identity"].(map[string]interface{}); ok {
				if s, ok := toNonEmptyString(identity["bluetoothMac"]); ok {
					if normalized, ok := normalizeBleMac12ForStore(s); ok {
						identityBleMac = &normalized
					}
				}
				if identityBleMac == nil {
					if s, ok := toNonEmptyString(identity["bluetooth_mac"]); ok {
						if normalized, ok := normalizeBleMac12ForStore(s); ok {
							identityBleMac = &normalized
						}
					}
				}
			}
		}

		if soc != nil || soh != nil || identityBleMac != nil {
			if err := tx.Exec(
				`INSERT INTO device_batteries (device_id, soc, soh, identity_ble_mac, updated_at)
				 VALUES (?, ?, ?, ?, NOW())
				 ON CONFLICT (device_id) DO UPDATE
				 SET soc = COALESCE(EXCLUDED.soc, device_batteries.soc),
				     soh = COALESCE(EXCLUDED.soh, device_batteries.soh),
				     identity_ble_mac = COALESCE(NULLIF(EXCLUDED.identity_ble_mac, ''), device_batteries.identity_ble_mac),
				     updated_at = NOW()`,
				deviceID, soc, soh, identityBleMac,
			).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if shouldSyncDeviceOnlineByApp(batteryDetail) {
		if _, err := syncDeviceStatusFromApp(ctx, deviceID, tenantID, appBatteryStatusOnline, "app_report"); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"device_id": deviceID,
				"tenant_id": tenantID,
			}).Warn("sync online status by app report failed")
		}
	}

	publishAppBatteryWSEvent(deviceID, tenantID, wsPayload)

	logrus.WithFields(logrus.Fields{
		"device_id": deviceID,
		"tenant_id": tenantID,
		"conn_type": connType,
		"platform":  platform,
		"core_keys": len(coreValues),
		"snapshot":  snapshotRaw != nil,
	}).Debug("app battery telemetry report accepted")

	return &model.AppBatteryReportResp{
		DeviceID: deviceID,
		Ts:       reportTs,
		Accepted: true,
	}, nil
}

// ReportBatteryConnectionStatusForApp APP端蓝牙连接状态同步
func (*AppBattery) ReportBatteryConnectionStatusForApp(ctx context.Context, req model.AppBatteryConnectionStatusReq, claims *utils.UserClaims) (*model.AppBatteryConnectionStatusResp, error) {
	if claims == nil || claims.ID == "" || claims.TenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "claims is required")
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device_id is required")
	}

	ts := req.Ts
	if ts <= 0 {
		ts = time.Now().UnixMilli()
	}

	connType := strings.ToLower(strings.TrimSpace(req.ConnType))
	if connType == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "conn_type is required")
	}

	if !isAppReportEnabled() {
		return &model.AppBatteryConnectionStatusResp{
			DeviceID:      deviceID,
			Ts:            ts,
			BleConnected:  req.BleConnected,
			Accepted:      false,
			IgnoredReason: "feature_disabled",
		}, nil
	}

	if connType != "bluetooth" {
		return &model.AppBatteryConnectionStatusResp{
			DeviceID:      deviceID,
			Ts:            ts,
			BleConnected:  req.BleConnected,
			Accepted:      false,
			IgnoredReason: "non_bluetooth",
		}, nil
	}

	batteryDetail, err := new(AppBattery).GetBatteryDetailForApp(ctx, deviceID, claims)
	if err != nil {
		return nil, err
	}

	if !shouldSyncDeviceOnlineByApp(batteryDetail) {
		return &model.AppBatteryConnectionStatusResp{
			DeviceID:      deviceID,
			Ts:            ts,
			BleConnected:  req.BleConnected,
			Accepted:      false,
			IgnoredReason: "not_ble_relay_device",
		}, nil
	}

	targetStatus := appBatteryStatusOffline
	source := "app_connection_disconnected"
	if req.BleConnected {
		targetStatus = appBatteryStatusOnline
		source = "app_connection_connected"
	}

	changed := false
	if targetStatus == appBatteryStatusOnline {
		var err error
		changed, err = syncDeviceStatusFromApp(ctx, deviceID, claims.TenantID, targetStatus, source)
		if err != nil {
			return nil, err
		}
	} else {
		before, err := query.Device.WithContext(ctx).
			Select(query.Device.IsOnline).
			Where(query.Device.ID.Eq(deviceID), query.Device.TenantID.Eq(claims.TenantID)).
			First()
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if err := syncDeviceStatusOfflineIfNoRelayOwner(ctx, deviceID, claims.TenantID, source); err != nil {
			return nil, err
		}
		after, err := query.Device.WithContext(ctx).
			Select(query.Device.IsOnline).
			Where(query.Device.ID.Eq(deviceID), query.Device.TenantID.Eq(claims.TenantID)).
			First()
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		changed = before.IsOnline != after.IsOnline
	}

	return &model.AppBatteryConnectionStatusResp{
		DeviceID:      deviceID,
		Ts:            ts,
		BleConnected:  req.BleConnected,
		Accepted:      true,
		StatusChanged: changed,
	}, nil
}

func isAppReportEnabled() bool {
	enabled := true
	if viper.IsSet("bms.app_report.enabled") {
		enabled = viper.GetBool("bms.app_report.enabled")
	}
	return enabled
}

func isAppReportOnlineSyncEnabled() bool {
	enabled := true
	if viper.IsSet("bms.app_report.sync_device_online") {
		enabled = viper.GetBool("bms.app_report.sync_device_online")
	}
	return enabled
}

func isAppReportOfflineSyncEnabled() bool {
	enabled := true
	if viper.IsSet("bms.app_report.offline_on_ble_disconnect") {
		enabled = viper.GetBool("bms.app_report.offline_on_ble_disconnect")
	}
	return enabled
}

func appReportOnlineTTL() time.Duration {
	ttlSec := 45
	if viper.IsSet("bms.app_report.online_ttl_sec") {
		ttlSec = viper.GetInt("bms.app_report.online_ttl_sec")
	}
	if ttlSec < 5 {
		ttlSec = 5
	}
	return time.Duration(ttlSec) * time.Second
}

func shouldSyncDeviceOnlineByApp(detail *model.AppBatteryDetailResp) bool {
	if detail == nil {
		return true
	}
	if detail.BmsCommType != nil && *detail.BmsCommType == 1 {
		return true
	}
	commChipID := strings.TrimSpace(derefString(detail.CommChipID))
	if commChipID == "" {
		return true
	}
	return false
}

func syncDeviceStatusFromApp(ctx context.Context, deviceID, tenantID string, status int16, source string) (bool, error) {
	if status == appBatteryStatusOnline && !isAppReportOnlineSyncEnabled() {
		return false, nil
	}
	if status == appBatteryStatusOffline && !isAppReportOfflineSyncEnabled() {
		return false, nil
	}

	changed, err := dal.UpdateDeviceStatus(deviceID, status)
	if err != nil {
		return false, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if status == appBatteryStatusOnline {
		refreshAppReportOnlineKey(ctx, deviceID)
	} else {
		clearAppReportOnlineKey(ctx, deviceID)
	}

	if changed {
		publishAppDeviceStatus(ctx, deviceID, status)
	}

	logrus.WithFields(logrus.Fields{
		"device_id":      deviceID,
		"tenant_id":      tenantID,
		"status":         status,
		"status_changed": changed,
		"source":         source,
	}).Debug("app sync device online status")

	return changed, nil
}

func refreshAppReportOnlineKey(ctx context.Context, deviceID string) {
	if global.REDIS == nil {
		return
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	key := "device:" + deviceID + ":heartbeat"
	if err := global.REDIS.Set(ctx, key, 1, appReportOnlineTTL()).Err(); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"device_id": deviceID,
			"key":       key,
		}).Warn("refresh app report online key failed")
	}
}

func clearAppReportOnlineKey(ctx context.Context, deviceID string) {
	if global.REDIS == nil {
		return
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	heartbeatKey := "device:" + deviceID + ":heartbeat"
	timeoutKey := "device:" + deviceID + ":timeout"
	if err := global.REDIS.Del(ctx, heartbeatKey, timeoutKey).Err(); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"device_id": deviceID,
		}).Warn("clear app report online key failed")
	}
}

func publishAppDeviceStatus(ctx context.Context, deviceID string, status int16) {
	if global.REDIS == nil {
		return
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	raw, err := json.Marshal(map[string]interface{}{
		"is_online": int(status),
	})
	if err != nil {
		return
	}
	channel := "device:" + deviceID + ":status"
	if err := global.REDIS.Publish(ctx, channel, string(raw)).Err(); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"device_id": deviceID,
			"channel":   channel,
		}).Warn("publish app device status failed")
	}
}

func syncDeviceStatusOfflineIfNoRelayOwner(ctx context.Context, deviceID, tenantID, source string) error {
	if !isAppReportOfflineSyncEnabled() {
		return nil
	}
	owner, found, err := loadRelayOwnerState(ctx, deviceID)
	if err != nil {
		return err
	}
	if found && owner != nil && owner.TenantID == tenantID {
		return nil
	}
	canSync, err := canSyncDeviceOnlineByAppDevice(ctx, deviceID, tenantID)
	if err != nil {
		return err
	}
	if !canSync {
		return nil
	}
	_, err = syncDeviceStatusFromApp(ctx, deviceID, tenantID, appBatteryStatusOffline, source)
	return err
}

func canSyncDeviceOnlineByAppDevice(ctx context.Context, deviceID, tenantID string) (bool, error) {
	type row struct {
		BmsCommType *int    `gorm:"column:bms_comm_type"`
		CommChipID  *string `gorm:"column:comm_chip_id"`
	}
	var r row
	err := global.DB.WithContext(ctx).
		Table("devices AS d").
		Select("dbat.bms_comm_type AS bms_comm_type, dbat.comm_chip_id AS comm_chip_id").
		Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
		Where("d.id = ? AND d.tenant_id = ?", strings.TrimSpace(deviceID), strings.TrimSpace(tenantID)).
		Scan(&r).Error
	if err != nil {
		return false, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return shouldSyncDeviceOnlineByApp(&model.AppBatteryDetailResp{
		BmsCommType: r.BmsCommType,
		CommChipID:  r.CommChipID,
	}), nil
}

func normalizeAppBatteryCore(core map[string]interface{}) (map[string]interface{}, error) {
	out := make(map[string]interface{}, len(core))
	for rawKey, rawVal := range core {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "core key is empty")
		}
		if _, ok := appBatteryCoreKeyWhitelist[key]; !ok {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "core key not allowed: "+key)
		}
		val, ok := normalizeCoreValue(rawVal)
		if !ok {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "invalid core value type for key: "+key)
		}
		out[key] = val
	}
	return out, nil
}

func normalizeCoreValue(v interface{}) (interface{}, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int8:
		return float64(t), true
	case int16:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint:
		return float64(t), true
	case uint8:
		return float64(t), true
	case uint16:
		return float64(t), true
	case uint32:
		return float64(t), true
	case uint64:
		return float64(t), true
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f, true
		}
		return nil, false
	case string:
		return strings.TrimSpace(t), true
	default:
		return nil, false
	}
}

func normalizeAppBatterySnapshot(snapshot map[string]interface{}) (string, error) {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return "", errcode.NewWithMessage(errcode.CodeParamError, "snapshot is not valid json object")
	}
	if len(raw) > appBatterySnapshotMaxBytes {
		return "", errcode.NewWithMessage(errcode.CodeParamError, "snapshot payload exceeds 64KB")
	}
	return string(raw), nil
}

func mergeCurrentTelemetryIntoAppBatterySnapshot(snapshot map[string]interface{}, current map[string]model.AppBatteryCurrentTelemetryValue) map[string]interface{} {
	if snapshot == nil {
		return nil
	}

	setNumberPathFromCurrent(snapshot, current, []string{"meta", "seriesCount"}, "meta.seriesCount", "seriesCount")
	setNumberPathFromCurrent(snapshot, current, []string{"meta", "cellTempCount"}, "meta.cellTempCount")
	setNumberPathFromCurrent(snapshot, current, []string{"energy", "socPct"}, "soc")
	setNumberPathFromCurrent(snapshot, current, []string{"energy", "sohPct"}, "soh")
	setNumberPathFromCurrent(snapshot, current, []string{"energy", "cycleCount"}, "cycleCount")
	setNumberPathFromCurrent(snapshot, current, []string{"timing", "chargeRemainingMin"}, "chargeRemainingMin")
	setNumberPathFromCurrent(snapshot, current, []string{"timing", "dischargeRemainingMin"}, "dischargeRemainingMin")
	setNumberPathFromCurrent(snapshot, current, []string{"electrical", "packCellSumVoltageV"}, "electrical.packCellSumVoltageV", "packCellSumVoltageV")
	setNumberPathFromCurrent(snapshot, current, []string{"electrical", "vPackV"}, "electrical.vPackV", "vPackV")
	setNumberPathFromCurrent(snapshot, current, []string{"electrical", "currentA"}, "electrical.currentA", "currentA")
	setNumberPathFromCurrent(snapshot, current, []string{"electrical", "avgCellVoltageMv"}, "electrical.avgCellVoltageMv", "avgCellVoltageMv")
	setNumberPathFromCurrent(snapshot, current, []string{"electrical", "highestCellVoltageMv"}, "electrical.highestCellVoltageMv", "highestCellVoltageMv")
	setNumberPathFromCurrent(snapshot, current, []string{"electrical", "lowestCellVoltageMv"}, "electrical.lowestCellVoltageMv", "lowestCellVoltageMv")
	setNumberPathFromCurrent(snapshot, current, []string{"electrical", "maxCellVoltageDiffMv"}, "electrical.maxCellVoltageDiffMv", "maxCellVoltageDiffMv")
	setNumberPathFromCurrent(snapshot, current, []string{"electrical", "cellVoltageIndex", "highest"}, "electrical.cellVoltageIndex.highest", "cellVoltageHighestIndex")
	setNumberPathFromCurrent(snapshot, current, []string{"electrical", "cellVoltageIndex", "lowest"}, "electrical.cellVoltageIndex.lowest", "cellVoltageLowestIndex")
	setNumberPathFromCurrent(snapshot, current, []string{"temperature", "chargeMosC"}, "temperature.chargeMosC", "chargeMosC")
	setNumberPathFromCurrent(snapshot, current, []string{"temperature", "dischargeMosC"}, "temperature.dischargeMosC", "dischargeMosC")
	setNumberPathFromCurrent(snapshot, current, []string{"temperature", "ambientC"}, "temperature.ambientC", "ambientC")
	setBoolPathFromCurrent(snapshot, current, []string{"status", "indicatorStatus", "chargeFetOn"}, "chargeFetOn")
	setBoolPathFromCurrent(snapshot, current, []string{"status", "indicatorStatus", "dischargeFetOn"}, "dischargeFetOn")
	setBoolPathFromCurrent(snapshot, current, []string{"status", "indicatorStatus", "charging"}, "charging")
	setBoolPathFromCurrent(snapshot, current, []string{"status", "indicatorStatus", "discharging"}, "discharging")

	if values, ok := currentNumberArray(current, "cell.voltagesMv", "cellVoltagesMv"); ok && hasValidBmsCellVoltageItems(values) {
		setPath(snapshot, []string{"cell", "voltagesMv"}, values)
	} else if existing, ok := snapshotNumberArray(snapshot, []string{"cell", "voltagesMv"}); ok && !hasValidBmsCellVoltageMv(existing) {
		setPath(snapshot, []string{"cell", "voltagesMv"}, []interface{}{})
	}
	if values, ok := currentNumberArray(current, "temperature.cellTempsC", "cellTempsC"); ok {
		setPath(snapshot, []string{"temperature", "cellTempsC"}, values)
	}

	return snapshot
}

func setNumberPathFromCurrent(snapshot map[string]interface{}, current map[string]model.AppBatteryCurrentTelemetryValue, path []string, keys ...string) {
	value, ok := currentValueByKeys(current, keys...)
	if !ok {
		return
	}
	n, ok := toFloat64(value)
	if !ok {
		return
	}
	setPath(snapshot, path, n)
}

func setBoolPathFromCurrent(snapshot map[string]interface{}, current map[string]model.AppBatteryCurrentTelemetryValue, path []string, keys ...string) {
	value, ok := currentValueByKeys(current, keys...)
	if !ok {
		return
	}
	b, ok := toBool(value)
	if !ok {
		return
	}
	setPath(snapshot, path, b)
}

func currentValueByKeys(current map[string]model.AppBatteryCurrentTelemetryValue, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		if item, ok := current[key]; ok {
			return item.Value, true
		}
	}
	return nil, false
}

func setPath(root map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}
	m := root
	for _, segment := range path[:len(path)-1] {
		next, _ := m[segment].(map[string]interface{})
		if next == nil {
			next = make(map[string]interface{})
			m[segment] = next
		}
		m = next
	}
	m[path[len(path)-1]] = value
}

func currentNumberArray(current map[string]model.AppBatteryCurrentTelemetryValue, keys ...string) ([]interface{}, bool) {
	value, ok := currentValueByKeys(current, keys...)
	if !ok {
		return nil, false
	}
	values, ok := numberArrayValue(value)
	if !ok {
		return nil, false
	}
	out := make([]interface{}, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}
	return out, true
}

func snapshotNumberArray(snapshot map[string]interface{}, path []string) ([]float64, bool) {
	var value interface{} = snapshot
	for _, segment := range path {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil, false
		}
		value, ok = m[segment]
		if !ok {
			return nil, false
		}
	}
	return numberArrayValue(value)
}

func numberArrayValue(value interface{}) ([]float64, bool) {
	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, false
		}
		var parsed interface{}
		if err := json.Unmarshal([]byte(text), &parsed); err != nil {
			return nil, false
		}
		value = parsed
	}
	raw, ok := value.([]interface{})
	if !ok {
		return nil, false
	}
	out := make([]float64, 0, len(raw))
	for _, item := range raw {
		n, ok := toFloat64(item)
		if !ok {
			continue
		}
		out = append(out, n)
	}
	return out, len(out) > 0
}

func hasValidBmsCellVoltageMv(values []float64) bool {
	for _, v := range values {
		if v > 0 && v < 10000 && v != 0xffff {
			return true
		}
	}
	return false
}

func hasValidBmsCellVoltageItems(values []interface{}) bool {
	for _, item := range values {
		n, ok := toFloat64(item)
		if ok && n > 0 && n < 10000 && n != 0xffff {
			return true
		}
	}
	return false
}

func toBool(v interface{}) (bool, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true", "1", "on":
			return true, true
		case "false", "0", "off":
			return false, true
		default:
			return false, false
		}
	case int:
		return t != 0, true
	case int8:
		return t != 0, true
	case int16:
		return t != 0, true
	case int32:
		return t != 0, true
	case int64:
		return t != 0, true
	case uint:
		return t != 0, true
	case uint8:
		return t != 0, true
	case uint16:
		return t != 0, true
	case uint32:
		return t != 0, true
	case uint64:
		return t != 0, true
	case float32:
		return t != 0, true
	case float64:
		return t != 0, true
	case json.Number:
		f, err := t.Float64()
		return f != 0, err == nil
	default:
		return false, false
	}
}

func toTelemetryColumns(v interface{}) (*bool, *float64, *string) {
	switch t := v.(type) {
	case bool:
		return &t, nil, nil
	case float64:
		return nil, &t, nil
	case string:
		return nil, nil, &t
	default:
		s := strings.TrimSpace(toString(v))
		return nil, nil, &s
	}
}

func telemetryCurrentValue(row *model.TelemetryCurrentData) interface{} {
	if row == nil {
		return nil
	}
	if row.BoolV != nil {
		return *row.BoolV
	}
	if row.NumberV != nil {
		return *row.NumberV
	}
	if row.StringV != nil {
		return *row.StringV
	}
	return nil
}

func toFloat64(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int8:
		return float64(t), true
	case int16:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint:
		return float64(t), true
	case uint8:
		return float64(t), true
	case uint16:
		return float64(t), true
	case uint32:
		return float64(t), true
	case uint64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func toNonEmptyString(v interface{}) (string, bool) {
	s := strings.TrimSpace(toString(v))
	if s == "" {
		return "", false
	}
	return s, true
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func publishAppBatteryWSEvent(deviceID, tenantID string, data map[string]interface{}) {
	if len(data) == 0 || global.REDIS == nil {
		return
	}
	ctx := context.Background()
	exists, err := global.REDIS.Exists(ctx, "ws:sub:"+deviceID).Result()
	if err != nil {
		logrus.WithError(err).WithField("device_id", deviceID).Debug("app report ws exists check failed")
		return
	}
	if exists == 0 {
		return
	}

	event := global.WSEvent{
		DeviceID:  deviceID,
		TenantID:  tenantID,
		Timestamp: time.Now().UnixMilli(),
		Data:      data,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		logrus.WithError(err).WithField("device_id", deviceID).Error("marshal app report ws event failed")
		return
	}
	if err := global.REDIS.Publish(ctx, "ws:device:"+deviceID, payload).Err(); err != nil {
		logrus.WithError(err).WithField("device_id", deviceID).Error("publish app report ws event failed")
	}
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

type appBatteryOtaMatchCriteria struct {
	BatteryModelID string
	BatchNumber    string
	ItemUUID       string
}

func firstTrimmed(values ...*string) string {
	for _, value := range values {
		if value == nil {
			continue
		}
		if s := strings.TrimSpace(*value); s != "" {
			return s
		}
	}
	return ""
}

func packageConstraintValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func matchOptionalConstraint(pkgValue, actualValue string) bool {
	if pkgValue == "" {
		return true
	}
	return actualValue != "" && strings.EqualFold(pkgValue, actualValue)
}

func appBatteryOtaPackageMatchScore(pkg model.OtaUpgradePackage, criteria appBatteryOtaMatchCriteria) (int, bool) {
	itemUUID := packageConstraintValue(pkg.ItemUUID)
	batteryModelID := packageConstraintValue(pkg.BatteryModelID)
	batchNumber := packageConstraintValue(pkg.BatchNumber)
	constraintCount := 0
	if itemUUID != "" {
		constraintCount++
	}
	if batteryModelID != "" {
		constraintCount++
	}
	if batchNumber != "" {
		constraintCount++
	}

	if !matchOptionalConstraint(itemUUID, criteria.ItemUUID) ||
		!matchOptionalConstraint(batteryModelID, criteria.BatteryModelID) ||
		!matchOptionalConstraint(batchNumber, criteria.BatchNumber) {
		return 0, false
	}

	switch constraintCount {
	case 0:
		return 0, true
	case 1:
		if itemUUID != "" {
			return 13, true
		}
		if batteryModelID != "" {
			return 12, true
		}
		return 11, true
	case 2:
		return 20, true
	case 3:
		return 30, true
	default:
		return 0, false
	}
}

func appBatteryOtaPackageTargetVersionMatches(pkg model.OtaUpgradePackage, currentVersion string) bool {
	targetVersion := packageConstraintValue(pkg.TargetVersion)
	if targetVersion == "" {
		return true
	}
	return strings.EqualFold(targetVersion, strings.TrimSpace(currentVersion))
}

func selectAppBatteryOtaPackage(packages []model.OtaUpgradePackage, currentVersion string, criteria appBatteryOtaMatchCriteria) *model.OtaUpgradePackage {
	var selected *model.OtaUpgradePackage
	selectedScore := -1
	for i := range packages {
		pkg := &packages[i]
		if !appBatteryOtaPackageTargetVersionMatches(*pkg, currentVersion) {
			continue
		}
		if compareVersion(pkg.Version, currentVersion) <= 0 {
			continue
		}
		score, ok := appBatteryOtaPackageMatchScore(*pkg, criteria)
		if !ok {
			continue
		}
		if selected == nil ||
			score > selectedScore ||
			(score == selectedScore && compareVersion(pkg.Version, selected.Version) > 0) ||
			(score == selectedScore && compareVersion(pkg.Version, selected.Version) == 0 && pkg.CreatedAt.After(selected.CreatedAt)) {
			selected = pkg
			selectedScore = score
		}
	}
	return selected
}

func compareVersion(a, b string) int {
	aa := normalizeVersion(a)
	bb := normalizeVersion(b)
	if aa == "" && bb == "" {
		return 0
	}
	if aa == "" {
		return -1
	}
	if bb == "" {
		return 1
	}
	if isNumeric(aa) && isNumeric(bb) {
		return compareNumericStrings(aa, bb)
	}
	if isSemverLike(aa) && isSemverLike(bb) {
		return compareSemver(aa, bb)
	}
	return strings.Compare(strings.ToLower(aa), strings.ToLower(bb))
}

func normalizeVersion(s string) string {
	out := strings.TrimSpace(s)
	out = strings.TrimPrefix(strings.ToLower(out), "v")
	return out
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isSemverLike(s string) bool {
	if s == "" {
		return false
	}
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if !isNumeric(p) {
			return false
		}
	}
	return true
}

func compareSemver(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		ai := 0
		if i < len(ap) && ap[i] != "" {
			ai = atoiSafe(ap[i])
		}
		bi := 0
		if i < len(bp) && bp[i] != "" {
			bi = atoiSafe(bp[i])
		}
		if ai > bi {
			return 1
		}
		if ai < bi {
			return -1
		}
	}
	return 0
}

func compareNumericStrings(a, b string) int {
	aa := strings.TrimLeft(a, "0")
	bb := strings.TrimLeft(b, "0")
	if aa == "" {
		aa = "0"
	}
	if bb == "" {
		bb = "0"
	}
	if len(aa) > len(bb) {
		return 1
	}
	if len(aa) < len(bb) {
		return -1
	}
	return strings.Compare(aa, bb)
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func stringPtrOrNil(s string) *string {
	out := strings.TrimSpace(s)
	if out == "" {
		return nil
	}
	return &out
}

const (
	appBatteryRelayOwnerTTL   = 45 * time.Second
	appBatteryRelaySessionTTL = 60 * time.Second
	appBatteryRelayCommandTTL = 10 * time.Minute

	appBatteryRelayStatusPending = "PENDING"
	appBatteryRelayStatusSent    = "SENT"
	appBatteryRelayStatusSuccess = "SUCCESS"
	appBatteryRelayStatusFailed  = "FAILED"
	appBatteryRelayStatusTimeout = "TIMEOUT"
)

type appBatteryRelayOwnerState struct {
	DeviceID    string `json:"device_id"`
	SessionID   string `json:"session_id"`
	UserID      string `json:"user_id"`
	TenantID    string `json:"tenant_id"`
	Platform    string `json:"platform"`
	ConnType    string `json:"conn_type"`
	LastSeenTs  int64  `json:"last_seen_ts"`
	ExpiresAtTs int64  `json:"expires_at_ts"`
}

type appBatteryRelaySessionState struct {
	SessionID    string `json:"session_id"`
	DeviceID     string `json:"device_id"`
	UserID       string `json:"user_id"`
	TenantID     string `json:"tenant_id"`
	Platform     string `json:"platform"`
	ConnType     string `json:"conn_type"`
	BleConnected bool   `json:"ble_connected"`
	LastSeenTs   int64  `json:"last_seen_ts"`
	CreatedAtTs  int64  `json:"created_at_ts"`
}

type appBatteryRelayCommandState struct {
	CommandID      string      `json:"command_id"`
	DeviceID       string      `json:"device_id"`
	SessionID      string      `json:"session_id"`
	TenantID       string      `json:"tenant_id"`
	RequestUserID  string      `json:"request_user_id"`
	CommandType    string      `json:"command_type"`
	ParamKey       *string     `json:"param_key,omitempty"`
	Value          interface{} `json:"value,omitempty"`
	StartAddress   *int        `json:"start_address,omitempty"`
	RegisterValues []int       `json:"register_values,omitempty"`
	Status         string      `json:"status"`
	ErrorMessage   *string     `json:"error_message,omitempty"`
	Result         interface{} `json:"result,omitempty"`
	CreatedAtTs    int64       `json:"created_at_ts"`
	UpdatedAtTs    int64       `json:"updated_at_ts"`
	FinishedAtTs   *int64      `json:"finished_at_ts,omitempty"`
}

type appBatteryRelayCommandEnvelope struct {
	Type           string      `json:"type"`
	CommandID      string      `json:"cmd_id"`
	DeviceID       string      `json:"device_id"`
	CommandType    string      `json:"command_type"`
	ParamKey       *string     `json:"param_key,omitempty"`
	Value          interface{} `json:"value,omitempty"`
	StartAddress   *int        `json:"start_address,omitempty"`
	RegisterValues []int       `json:"register_values,omitempty"`
	Ts             int64       `json:"ts"`
}

func appBatteryRelayOwnerKey(deviceID string) string {
	return "bms:relay:owner:" + strings.TrimSpace(deviceID)
}

func appBatteryRelaySessionKey(sessionID string) string {
	return "bms:relay:session:" + strings.TrimSpace(sessionID)
}

func appBatteryRelayCommandKey(commandID string) string {
	return "bms:relay:cmd:" + strings.TrimSpace(commandID)
}

// AppBatteryRelayCommandChannel APP Relay session 指令通道
func AppBatteryRelayCommandChannel(sessionID string) string {
	return "bms:relay:cmd:session:" + strings.TrimSpace(sessionID)
}

func ensureWebBatteryDeviceAccess(ctx context.Context, deviceID string, claims *utils.UserClaims, orgID string) error {
	if claims == nil || claims.ID == "" || claims.TenantID == "" {
		return errcode.NewWithMessage(errcode.CodeParamError, "claims is required")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errcode.NewWithMessage(errcode.CodeParamError, "device_id is required")
	}

	_, err := query.Device.WithContext(ctx).
		Where(query.Device.ID.Eq(deviceID), query.Device.TenantID.Eq(claims.TenantID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "device not found"})
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if err := checkDeviceOrgAccess(ctx, deviceID, claims.TenantID, strings.TrimSpace(orgID)); err != nil {
		return err
	}

	return nil
}

func (*AppBattery) OpenRelaySessionForApp(ctx context.Context, deviceID, platform, connType string, bleConnected bool, claims *utils.UserClaims) (*appBatteryRelaySessionState, error) {
	if global.REDIS == nil {
		return nil, errcode.NewWithMessage(errcode.CodeSystemError, "redis unavailable")
	}

	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device_id is required")
	}
	platform = strings.TrimSpace(strings.ToLower(platform))
	if platform == "" {
		platform = "unknown"
	}
	connType = strings.TrimSpace(strings.ToLower(connType))
	if connType == "" {
		connType = "bluetooth"
	}

	batteryDetail, err := new(AppBattery).GetBatteryDetailForApp(ctx, deviceID, claims)
	if err != nil {
		return nil, err
	}

	nowMs := time.Now().UnixMilli()
	session := &appBatteryRelaySessionState{
		SessionID:    uuid.New(),
		DeviceID:     deviceID,
		UserID:       claims.ID,
		TenantID:     claims.TenantID,
		Platform:     platform,
		ConnType:     connType,
		BleConnected: bleConnected,
		LastSeenTs:   nowMs,
		CreatedAtTs:  nowMs,
	}
	if err := saveRelaySessionState(ctx, session); err != nil {
		return nil, err
	}
	if bleConnected {
		if err := setRelayOwnerState(ctx, session); err != nil {
			return nil, err
		}
		if shouldSyncDeviceOnlineByApp(batteryDetail) {
			if _, err := syncDeviceStatusFromApp(ctx, session.DeviceID, session.TenantID, appBatteryStatusOnline, "relay_session_open"); err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"device_id":  session.DeviceID,
					"session_id": session.SessionID,
				}).Warn("sync online status on relay open failed")
			}
		}
	}
	return session, nil
}

func (*AppBattery) RefreshRelaySessionForApp(ctx context.Context, sessionID string, bleConnected bool) (*appBatteryRelaySessionState, error) {
	if global.REDIS == nil {
		return nil, errcode.NewWithMessage(errcode.CodeSystemError, "redis unavailable")
	}
	session, found, err := loadRelaySessionState(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "relay session not found")
	}

	session.BleConnected = bleConnected
	session.LastSeenTs = time.Now().UnixMilli()
	if err := saveRelaySessionState(ctx, session); err != nil {
		return nil, err
	}

	if bleConnected {
		if err := setRelayOwnerState(ctx, session); err != nil {
			return nil, err
		}
	} else {
		if err := clearRelayOwnerIfMatched(ctx, session.DeviceID, session.SessionID); err != nil {
			return nil, err
		}
		if err := syncDeviceStatusOfflineIfNoRelayOwner(ctx, session.DeviceID, session.TenantID, "relay_heartbeat_disconnected"); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"device_id":  session.DeviceID,
				"session_id": session.SessionID,
			}).Warn("sync offline status on relay heartbeat failed")
		}
	}

	return session, nil
}

func (*AppBattery) CloseRelaySessionForApp(ctx context.Context, sessionID string) {
	if global.REDIS == nil {
		return
	}
	session, found, err := loadRelaySessionState(ctx, sessionID)
	if err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Warn("close relay session load failed")
		return
	}
	if !found {
		return
	}
	if err := clearRelayOwnerIfMatched(ctx, session.DeviceID, session.SessionID); err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Warn("close relay session clear owner failed")
	}
	if err := global.REDIS.Del(ctx, appBatteryRelaySessionKey(sessionID)).Err(); err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Warn("close relay session delete session key failed")
	}
	if err := syncDeviceStatusOfflineIfNoRelayOwner(ctx, session.DeviceID, session.TenantID, "relay_session_closed"); err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Warn("sync offline status on relay close failed")
	}
}

func (*AppBattery) GetRelayStatusForWeb(ctx context.Context, deviceID string, claims *utils.UserClaims, orgID string) (*model.AppBatteryRelayStatusResp, error) {
	if global.REDIS == nil {
		return nil, errcode.NewWithMessage(errcode.CodeSystemError, "redis unavailable")
	}
	if err := ensureWebBatteryDeviceAccess(ctx, deviceID, claims, orgID); err != nil {
		return nil, err
	}

	owner, found, err := loadRelayOwnerState(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	resp := &model.AppBatteryRelayStatusResp{
		DeviceID:    strings.TrimSpace(deviceID),
		OwnerOnline: false,
	}
	if !found {
		return resp, nil
	}
	if owner.TenantID != claims.TenantID {
		return resp, nil
	}

	resp.OwnerOnline = true
	resp.SessionID = stringPtrOrNil(owner.SessionID)
	resp.Platform = stringPtrOrNil(owner.Platform)
	resp.ConnType = stringPtrOrNil(owner.ConnType)
	resp.OwnerUserID = stringPtrOrNil(owner.UserID)
	resp.OwnerTenantID = stringPtrOrNil(owner.TenantID)
	resp.LastSeenTs = &owner.LastSeenTs
	resp.ExpiresAtTs = &owner.ExpiresAtTs
	return resp, nil
}

func (*AppBattery) SendRelayCommandForWeb(ctx context.Context, req model.AppBatteryRelayCommandReq, claims *utils.UserClaims, orgID string) (*model.AppBatteryRelayCommandResp, error) {
	if global.REDIS == nil {
		return nil, errcode.NewWithMessage(errcode.CodeSystemError, "redis unavailable")
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	commandType := strings.ToLower(strings.TrimSpace(req.CommandType))
	if err := ensureWebBatteryDeviceAccess(ctx, deviceID, claims, orgID); err != nil {
		return nil, err
	}

	owner, found, err := loadRelayOwnerState(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if !found || owner.TenantID != claims.TenantID {
		return nil, errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
			"message": "无可用 BLE 中继连接，请先在APP中蓝牙连接设备",
		})
	}

	var paramKey *string
	var startAddress *int
	var registerValues []int
	value := req.Value

	switch commandType {
	case "read_param":
		if req.ParamKey == nil || strings.TrimSpace(*req.ParamKey) == "" {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "param_key is required")
		}
		normalized := strings.TrimSpace(*req.ParamKey)
		if canonical, ok := protocol.NormalizeRWParamKey(normalized); ok {
			normalized = canonical
		}
		paramKey = &normalized
	case "write_param":
		if req.ParamKey == nil || strings.TrimSpace(*req.ParamKey) == "" {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "param_key is required")
		}
		if req.Value == nil {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "value is required")
		}
		normalized := strings.TrimSpace(*req.ParamKey)
		if canonical, ok := protocol.NormalizeRWParamKey(normalized); ok {
			normalized = canonical
		}
		paramKey = &normalized
	case "write_registers":
		if req.StartAddress == nil {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "start_address is required")
		}
		if *req.StartAddress < 0 || *req.StartAddress > 0xFFFF {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "start_address out of range")
		}
		if len(req.RegisterValues) == 0 {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "register_values is required")
		}
		if len(req.RegisterValues) > 120 {
			return nil, errcode.NewWithMessage(errcode.CodeParamError, "register_values exceeds 120")
		}
		registerValues = make([]int, 0, len(req.RegisterValues))
		for _, v := range req.RegisterValues {
			if v < 0 || v > 0xFFFF {
				return nil, errcode.NewWithMessage(errcode.CodeParamError, "register value out of range")
			}
			registerValues = append(registerValues, v)
		}
		sa := *req.StartAddress
		startAddress = &sa
	default:
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "command_type not supported")
	}

	waitMs := int64(15000)
	if req.WaitMs != nil {
		waitMs = *req.WaitMs
	}
	if waitMs < 0 {
		waitMs = 0
	}
	if waitMs > 30000 {
		waitMs = 30000
	}

	nowMs := time.Now().UnixMilli()
	state := &appBatteryRelayCommandState{
		CommandID:      uuid.New(),
		DeviceID:       deviceID,
		SessionID:      owner.SessionID,
		TenantID:       claims.TenantID,
		RequestUserID:  claims.ID,
		CommandType:    commandType,
		ParamKey:       paramKey,
		Value:          value,
		StartAddress:   startAddress,
		RegisterValues: registerValues,
		Status:         appBatteryRelayStatusPending,
		CreatedAtTs:    nowMs,
		UpdatedAtTs:    nowMs,
	}
	if err := saveRelayCommandState(ctx, state); err != nil {
		return nil, err
	}

	envelope := appBatteryRelayCommandEnvelope{
		Type:           "relay_command",
		CommandID:      state.CommandID,
		DeviceID:       state.DeviceID,
		CommandType:    state.CommandType,
		ParamKey:       state.ParamKey,
		Value:          state.Value,
		StartAddress:   state.StartAddress,
		RegisterValues: state.RegisterValues,
		Ts:             nowMs,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": err.Error()})
	}

	if err := global.REDIS.Publish(ctx, AppBatteryRelayCommandChannel(owner.SessionID), payload).Err(); err != nil {
		msg := "relay command publish failed"
		state.Status = appBatteryRelayStatusFailed
		state.ErrorMessage = &msg
		finish := time.Now().UnixMilli()
		state.UpdatedAtTs = finish
		state.FinishedAtTs = &finish
		_ = saveRelayCommandState(ctx, state)
		return toRelayCommandResp(state), nil
	}

	state.Status = appBatteryRelayStatusSent
	state.UpdatedAtTs = time.Now().UnixMilli()
	if err := saveRelayCommandState(ctx, state); err != nil {
		return nil, err
	}

	if waitMs == 0 {
		return toRelayCommandResp(state), nil
	}

	deadline := time.Now().Add(time.Duration(waitMs) * time.Millisecond)
	for {
		cur, found, err := loadRelayCommandState(ctx, state.CommandID)
		if err != nil {
			return nil, err
		}
		if !found {
			msg := "relay command state not found"
			state.Status = appBatteryRelayStatusFailed
			state.ErrorMessage = &msg
			finish := time.Now().UnixMilli()
			state.UpdatedAtTs = finish
			state.FinishedAtTs = &finish
			return toRelayCommandResp(state), nil
		}
		if cur.Status == appBatteryRelayStatusSuccess || cur.Status == appBatteryRelayStatusFailed || cur.Status == appBatteryRelayStatusTimeout {
			return toRelayCommandResp(cur), nil
		}
		if time.Now().After(deadline) {
			timeoutMsg := "relay command timeout"
			cur.Status = appBatteryRelayStatusTimeout
			cur.ErrorMessage = &timeoutMsg
			finish := time.Now().UnixMilli()
			cur.UpdatedAtTs = finish
			cur.FinishedAtTs = &finish
			_ = saveRelayCommandState(ctx, cur)
			return toRelayCommandResp(cur), nil
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (*AppBattery) GetRelayCommandForWeb(ctx context.Context, commandID string, claims *utils.UserClaims, orgID string) (*model.AppBatteryRelayCommandResp, error) {
	if global.REDIS == nil {
		return nil, errcode.NewWithMessage(errcode.CodeSystemError, "redis unavailable")
	}

	commandID = strings.TrimSpace(commandID)
	if commandID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "command_id is required")
	}

	state, found, err := loadRelayCommandState(ctx, commandID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "relay command not found"})
	}
	if state.TenantID != claims.TenantID {
		return nil, errcode.New(errcode.CodeNoPermission)
	}
	if err := ensureWebBatteryDeviceAccess(ctx, state.DeviceID, claims, orgID); err != nil {
		return nil, err
	}
	return toRelayCommandResp(state), nil
}

func (*AppBattery) AcceptRelayCommandResultForApp(ctx context.Context, sessionID, commandID string, ok bool, result interface{}, errorMessage *string) (*model.AppBatteryRelayCommandResp, error) {
	if global.REDIS == nil {
		return nil, errcode.NewWithMessage(errcode.CodeSystemError, "redis unavailable")
	}
	session, found, err := loadRelaySessionState(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "relay session not found")
	}

	commandID = strings.TrimSpace(commandID)
	if commandID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "cmd_id is required")
	}

	state, found, err := loadRelayCommandState(ctx, commandID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "relay command not found"})
	}
	if state.SessionID != session.SessionID || state.DeviceID != session.DeviceID {
		return nil, errcode.New(errcode.CodeNoPermission)
	}
	if state.Status == appBatteryRelayStatusSuccess || state.Status == appBatteryRelayStatusFailed || state.Status == appBatteryRelayStatusTimeout {
		return toRelayCommandResp(state), nil
	}

	finish := time.Now().UnixMilli()
	state.UpdatedAtTs = finish
	state.FinishedAtTs = &finish
	if ok {
		state.Status = appBatteryRelayStatusSuccess
		state.Result = result
		state.ErrorMessage = nil
	} else {
		state.Status = appBatteryRelayStatusFailed
		state.Result = nil
		msg := strings.TrimSpace(derefString(errorMessage))
		if msg == "" {
			msg = "relay command execution failed"
		}
		state.ErrorMessage = &msg
	}
	if err := saveRelayCommandState(ctx, state); err != nil {
		return nil, err
	}
	return toRelayCommandResp(state), nil
}

func saveRelaySessionState(ctx context.Context, session *appBatteryRelaySessionState) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": err.Error()})
	}
	if err := global.REDIS.Set(ctx, appBatteryRelaySessionKey(session.SessionID), raw, appBatteryRelaySessionTTL).Err(); err != nil {
		return errcode.WithData(errcode.CodeCacheError, map[string]interface{}{"error": err.Error()})
	}
	return nil
}

func loadRelaySessionState(ctx context.Context, sessionID string) (*appBatteryRelaySessionState, bool, error) {
	raw, err := global.REDIS.Get(ctx, appBatteryRelaySessionKey(sessionID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, errcode.WithData(errcode.CodeCacheError, map[string]interface{}{"error": err.Error()})
	}
	var out appBatteryRelaySessionState
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false, errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": err.Error()})
	}
	return &out, true, nil
}

func setRelayOwnerState(ctx context.Context, session *appBatteryRelaySessionState) error {
	nowMs := time.Now().UnixMilli()
	owner := appBatteryRelayOwnerState{
		DeviceID:    session.DeviceID,
		SessionID:   session.SessionID,
		UserID:      session.UserID,
		TenantID:    session.TenantID,
		Platform:    session.Platform,
		ConnType:    session.ConnType,
		LastSeenTs:  nowMs,
		ExpiresAtTs: nowMs + appBatteryRelayOwnerTTL.Milliseconds(),
	}
	raw, err := json.Marshal(owner)
	if err != nil {
		return errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": err.Error()})
	}
	if err := global.REDIS.Set(ctx, appBatteryRelayOwnerKey(session.DeviceID), raw, appBatteryRelayOwnerTTL).Err(); err != nil {
		return errcode.WithData(errcode.CodeCacheError, map[string]interface{}{"error": err.Error()})
	}
	return nil
}

func clearRelayOwnerIfMatched(ctx context.Context, deviceID, sessionID string) error {
	owner, found, err := loadRelayOwnerState(ctx, deviceID)
	if err != nil {
		return err
	}
	if !found || owner.SessionID != sessionID {
		return nil
	}
	if err := global.REDIS.Del(ctx, appBatteryRelayOwnerKey(deviceID)).Err(); err != nil {
		return errcode.WithData(errcode.CodeCacheError, map[string]interface{}{"error": err.Error()})
	}
	return nil
}

func loadRelayOwnerState(ctx context.Context, deviceID string) (*appBatteryRelayOwnerState, bool, error) {
	raw, err := global.REDIS.Get(ctx, appBatteryRelayOwnerKey(deviceID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, errcode.WithData(errcode.CodeCacheError, map[string]interface{}{"error": err.Error()})
	}
	var owner appBatteryRelayOwnerState
	if err := json.Unmarshal([]byte(raw), &owner); err != nil {
		return nil, false, errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": err.Error()})
	}
	return &owner, true, nil
}

func saveRelayCommandState(ctx context.Context, state *appBatteryRelayCommandState) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": err.Error()})
	}
	if err := global.REDIS.Set(ctx, appBatteryRelayCommandKey(state.CommandID), raw, appBatteryRelayCommandTTL).Err(); err != nil {
		return errcode.WithData(errcode.CodeCacheError, map[string]interface{}{"error": err.Error()})
	}
	return nil
}

func loadRelayCommandState(ctx context.Context, commandID string) (*appBatteryRelayCommandState, bool, error) {
	raw, err := global.REDIS.Get(ctx, appBatteryRelayCommandKey(commandID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, errcode.WithData(errcode.CodeCacheError, map[string]interface{}{"error": err.Error()})
	}
	var out appBatteryRelayCommandState
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false, errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"error": err.Error()})
	}
	return &out, true, nil
}

func toRelayCommandResp(state *appBatteryRelayCommandState) *model.AppBatteryRelayCommandResp {
	return &model.AppBatteryRelayCommandResp{
		CommandID:    state.CommandID,
		DeviceID:     state.DeviceID,
		CommandType:  state.CommandType,
		Status:       state.Status,
		ErrorMessage: state.ErrorMessage,
		Result:       state.Result,
		CreatedAtTs:  state.CreatedAtTs,
		UpdatedAtTs:  state.UpdatedAtTs,
		FinishedAtTs: state.FinishedAtTs,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
