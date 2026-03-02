package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"project/internal/dal"
	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

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

const (
	appBatterySnapshotKey      = "bms.snapshot"
	appBatterySnapshotMaxBytes = 64 * 1024
)

var appBatteryCoreKeyWhitelist = map[string]struct{}{
	"soc":                   {},
	"soh":                   {},
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
func (*AppBattery) CheckBatteryOtaForApp(ctx context.Context, req model.AppBatteryOtaCheckReq, claims *utils.UserClaims) (*model.AppBatteryOtaCheckResp, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
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

	deviceConfigID := row.DeviceConfigID
	if deviceConfigID == nil || strings.TrimSpace(*deviceConfigID) == "" {
		modelName := ""
		if req.Model != nil {
			modelName = strings.TrimSpace(*req.Model)
		}
		if modelName == "" && row.BatteryModelName != nil {
			modelName = strings.TrimSpace(*row.BatteryModelName)
		}
		if modelName != "" {
			var bm model.BatteryModel
			if err := global.DB.WithContext(ctx).
				Where("name = ? AND tenant_id = ?", modelName, claims.TenantID).
				Limit(1).
				First(&bm).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
				}
			} else if bm.DeviceConfigID != nil && strings.TrimSpace(*bm.DeviceConfigID) != "" {
				deviceConfigID = bm.DeviceConfigID
			}
		}
	}

	if deviceConfigID == nil || strings.TrimSpace(*deviceConfigID) == "" {
		return resp, nil
	}

	var packages []model.OtaUpgradePackage
	pkgQuery := global.DB.WithContext(ctx).Table(model.TableNameOtaUpgradePackage).
		Where("device_config_id = ?", strings.TrimSpace(*deviceConfigID))
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

	// 优先匹配 target_version == reportedVer 的升级包
	var selected *model.OtaUpgradePackage
	for i := range packages {
		p := &packages[i]
		if p.TargetVersion == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(*p.TargetVersion), reportedVer) {
			selected = p
			break
		}
	}

	// 没有匹配 target_version 时，按版本号大小选择最新包
	if selected == nil {
		for i := range packages {
			p := &packages[i]
			if p.TargetVersion != nil && strings.TrimSpace(*p.TargetVersion) != "" {
				// target_version 不匹配当前版本，忽略
				continue
			}
			if selected == nil {
				selected = p
				continue
			}
			if compareVersion(p.Version, selected.Version) > 0 {
				selected = p
			}
		}
	}

	if selected == nil {
		return resp, nil
	}

	cmp := compareVersion(selected.Version, reportedVer)
	resp.NeedUpgrade = cmp > 0

	// 若无需升级，返回版本信息但不携带下载地址
	if !resp.NeedUpgrade {
		resp.Version = &selected.Version
		resp.TargetVersion = selected.TargetVersion
		resp.PackageID = &selected.ID
		resp.PackageType = &selected.PackageType
		return resp, nil
	}

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
	if _, err := new(AppBattery).GetBatteryDetailForApp(ctx, deviceID, claims); err != nil {
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
		var bleMac *string
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
					bleMac = &s
				}
				if bleMac == nil {
					if s, ok := toNonEmptyString(identity["bluetooth_mac"]); ok {
						bleMac = &s
					}
				}
			}
		}

		if soc != nil || soh != nil || bleMac != nil {
			if err := tx.Exec(
				`INSERT INTO device_batteries (device_id, soc, soh, ble_mac, updated_at)
				 VALUES (?, ?, ?, ?, NOW())
				 ON CONFLICT (device_id) DO UPDATE
				 SET soc = COALESCE(EXCLUDED.soc, device_batteries.soc),
				     soh = COALESCE(EXCLUDED.soh, device_batteries.soh),
				     ble_mac = COALESCE(NULLIF(EXCLUDED.ble_mac, ''), device_batteries.ble_mac),
				     updated_at = NOW()`,
				deviceID, soc, soh, bleMac,
			).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
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
