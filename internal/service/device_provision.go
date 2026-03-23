package service

import (
	"context"
	"encoding/json"
	"strings"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// DeviceProvision 移动端设备开通（扫码/蓝牙绑定）
type DeviceProvision struct{}

type deviceProvisionRow struct {
	DeviceID     string  `gorm:"column:device_id"`
	DeviceNumber string  `gorm:"column:device_number"`
	DeviceName   *string `gorm:"column:device_name"`
	BleMac       *string `gorm:"column:ble_mac"`
	CommChipID   *string `gorm:"column:comm_chip_id"`
	BmsCommType  *int    `gorm:"column:bms_comm_type"`
	OwnerOrgID   *string `gorm:"column:owner_org_id"`
}

func normalizeMac12(input string) (string, error) {
	out, ok := normalizeBleMac12ForStore(input)
	if !ok {
		return "", errcode.NewWithMessage(errcode.CodeParamError, "invalid ble_mac, expected 12 hex chars")
	}
	return out, nil
}

func normalizeItemUUID(input string) string {
	return strings.ToUpper(strings.TrimSpace(input))
}

func getDTUDomainPortFromConfig() string {
	// 兼容两种命名，避免后续改配置造成老版本不可用
	if v := strings.TrimSpace(viper.GetString("bms.provision.dtu_domain_port")); v != "" {
		return v
	}
	if v := strings.TrimSpace(viper.GetString("bms.dtu_domain_port")); v != "" {
		return v
	}
	return ""
}

func allowLegacyAutoRegister() bool {
	return viper.GetBool("bms.provision.allow_legacy_auto_register")
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*errcode.Error)
	return ok && e.Code == errcode.CodeNotFound
}

func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}

func autoRegisterReasonPtr() *string {
	v := "legacy_ble_device_not_preset"
	return &v
}

func autoRegisterDeviceName(itemUUID string) string {
	s := strings.TrimSpace(itemUUID)
	if len(s) > 8 {
		s = s[len(s)-8:]
	}
	return "BMS-" + strings.ToUpper(s)
}

func defaultDeviceVoucher() string {
	return `{"default":"` + uuid.New() + `"}`
}

func (*DeviceProvision) findDeviceByItemUUIDWithDB(ctx context.Context, db *gorm.DB, itemUUID string, claims *utils.UserClaims) (*deviceProvisionRow, error) {
	itemUUID = normalizeItemUUID(itemUUID)
	if itemUUID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "item_uuid is required")
	}
	if claims == nil || claims.TenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "claims.tenant_id is required")
	}

	scanRow := func(where string, args ...interface{}) (*deviceProvisionRow, error) {
		var row deviceProvisionRow
		err := db.WithContext(ctx).
			Table("device_batteries AS dbat").
			Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			d.name AS device_name,
			dbat.ble_mac AS ble_mac,
			dbat.comm_chip_id AS comm_chip_id,
			dbat.bms_comm_type AS bms_comm_type,
			dbat.owner_org_id AS owner_org_id
		`).
			Joins("JOIN devices AS d ON d.id = dbat.device_id").
			Where(where, args...).
			Limit(1).
			Scan(&row).Error
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if row.DeviceID == "" {
			return nil, errcode.NewWithMessage(errcode.CodeNotFound, "device not found by item_uuid")
		}
		return &row, nil
	}

	row, err := scanRow("dbat.item_uuid = ? AND d.tenant_id = ?", itemUUID, claims.TenantID)
	if err == nil || !isNotFoundErr(err) {
		return row, err
	}

	// 兜底兼容历史错误数据：早期链路可能把小写 item_uuid 写入数据库。
	return scanRow("UPPER(dbat.item_uuid) = ? AND d.tenant_id = ?", itemUUID, claims.TenantID)
}

// GetProvisionConfig 获取移动端开通配置
func (*DeviceProvision) GetProvisionConfig(_ context.Context, _ string) (*model.DeviceProvisionConfigResp, error) {
	dtu := getDTUDomainPortFromConfig()
	if dtu == "" {
		return nil, errcode.NewWithMessage(errcode.CodeNotFound, "dtu_domain_port not configured")
	}
	return &model.DeviceProvisionConfigResp{DTUDomainPort: dtu}, nil
}

func (*DeviceProvision) findDeviceByItemUUID(ctx context.Context, itemUUID string, claims *utils.UserClaims) (*deviceProvisionRow, error) {
	return (&DeviceProvision{}).findDeviceByItemUUIDWithDB(ctx, global.DB, itemUUID, claims)
}

func (*DeviceProvision) createAutoRegisteredDevice(ctx context.Context, tx *gorm.DB, req model.DeviceProvisionBindReq, claims *utils.UserClaims) (*deviceProvisionRow, error) {
	itemUUID := normalizeItemUUID(req.ItemUUID)
	now := utils.GetUTCTime()
	deviceName := autoRegisterDeviceName(itemUUID)
	protocol := "BLE"
	accessWay := "A"
	remark1 := "移动端自注册"
	remark2 := "BLE_UUID_AUTO_REGISTER"
	description := "遗留设备由APP蓝牙读取UUID后自动补建"

	var bleMac *string
	if req.BleMac != nil && strings.TrimSpace(*req.BleMac) != "" {
		newMac, err := normalizeMac12(*req.BleMac)
		if err != nil {
			return nil, err
		}
		bleMac = &newMac
	}

	additionalInfo := map[string]interface{}{
		"auto_registered":    true,
		"register_source":    "APP_BLE",
		"register_reason":    "legacy_device_not_preset",
		"item_uuid":          itemUUID,
		"created_by_user_id": claims.ID,
		"created_at":         now.Format("2006-01-02T15:04:05Z07:00"),
	}
	if bleMac != nil {
		additionalInfo["ble_mac"] = *bleMac
	}
	additionalInfoJSON, _ := json.Marshal(additionalInfo)
	additionalInfoStr := string(additionalInfoJSON)

	device := model.Device{
		ID:             uuid.New(),
		Name:           &deviceName,
		Voucher:        defaultDeviceVoucher(),
		TenantID:       claims.TenantID,
		IsEnabled:      "enabled",
		ActivateFlag:   "inactive",
		CreatedAt:      &now,
		UpdateAt:       &now,
		DeviceNumber:   itemUUID,
		Protocol:       &protocol,
		Remark1:        &remark1,
		Remark2:        &remark2,
		AccessWay:      &accessWay,
		Description:    &description,
		AdditionalInfo: &additionalInfoStr,
	}
	if err := tx.WithContext(ctx).Create(&device).Error; err != nil {
		if isDuplicateKeyErr(err) {
			return (&DeviceProvision{}).findDeviceByItemUUIDWithDB(ctx, tx, itemUUID, claims)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO device_batteries (device_id, item_uuid, ble_mac, bms_comm_type, activation_status, transfer_status, updated_at)
		 VALUES (?, ?, ?, ?, 'INACTIVE', 'FACTORY', ?)`,
		device.ID, itemUUID, bleMac, 1, now,
	).Error; err != nil {
		if isDuplicateKeyErr(err) {
			return (&DeviceProvision{}).findDeviceByItemUUIDWithDB(ctx, tx, itemUUID, claims)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return (&DeviceProvision{}).findDeviceByItemUUIDWithDB(ctx, tx, itemUUID, claims)
}

func (*DeviceProvision) findOrCreateDeviceByItemUUID(ctx context.Context, req model.DeviceProvisionBindReq, claims *utils.UserClaims) (*deviceProvisionRow, bool, error) {
	svc := &DeviceProvision{}
	req.ItemUUID = normalizeItemUUID(req.ItemUUID)
	row, err := svc.findDeviceByItemUUID(ctx, req.ItemUUID, claims)
	if err == nil {
		return row, false, nil
	}
	if !isNotFoundErr(err) {
		return nil, false, err
	}
	if !allowLegacyAutoRegister() {
		return nil, false, err
	}

	var created bool
	err = global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var innerErr error
		row, innerErr = svc.findDeviceByItemUUIDWithDB(ctx, tx, req.ItemUUID, claims)
		if innerErr == nil {
			return nil
		}
		if !isNotFoundErr(innerErr) {
			return innerErr
		}
		row, innerErr = svc.createAutoRegisteredDevice(ctx, tx, req, claims)
		if innerErr != nil {
			return innerErr
		}
		created = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return row, created, nil
}

func (*DeviceProvision) upsertOrgAddedDeviceRecordTx(ctx context.Context, tx *gorm.DB, claims *utils.UserClaims, deviceID, source string) error {
	now := utils.GetUTCTime()
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO app_device_added_records (id, tenant_id, user_id, device_id, source, added_at, last_seen_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (tenant_id, user_id, device_id)
		 DO UPDATE SET source = EXCLUDED.source, last_seen_at = EXCLUDED.last_seen_at`,
		uuid.New(), claims.TenantID, claims.ID, deviceID, source, now, now,
	).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func (*DeviceProvision) bindEndUserDeviceTx(ctx context.Context, tx *gorm.DB, row *deviceProvisionRow, claims *utils.UserClaims) error {
	userOrgID, err := getUserOrgID(claims.ID)
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	var cnt int64
	if err := tx.WithContext(ctx).
		Table("device_user_bindings").
		Where("device_id = ? AND user_id = ?", row.DeviceID, claims.ID).
		Count(&cnt).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if cnt > 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "device already bound to current user"})
	}

	if err := tx.WithContext(ctx).
		Table("device_user_bindings").
		Where("device_id = ?", row.DeviceID).
		Count(&cnt).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	isFirstBinding := cnt == 0
	now := utils.GetUTCTime()

	if userOrgID != "" && row.OwnerOrgID != nil && strings.TrimSpace(*row.OwnerOrgID) != "" {
		if !canAccessOrg(claims.TenantID, userOrgID, strings.TrimSpace(*row.OwnerOrgID)) {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "device does not belong to current organization"})
		}
	}

	updates := map[string]interface{}{
		"activation_status": "ACTIVE",
		"transfer_status":   "USER",
		"activation_date":   now,
		"updated_at":        now,
	}
	if (row.OwnerOrgID == nil || strings.TrimSpace(*row.OwnerOrgID) == "") && userOrgID != "" {
		updates["owner_org_id"] = userOrgID
	}
	if err := tx.WithContext(ctx).
		Table("device_batteries").
		Where("device_id = ?", row.DeviceID).
		Updates(updates).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	isOwner := isFirstBinding
	binding := &model.DeviceUserBinding{
		ID:          uuid.New(),
		UserID:      claims.ID,
		DeviceID:    row.DeviceID,
		BindingTime: &now,
		IsOwner:     &isOwner,
	}
	if err := tx.WithContext(ctx).Create(binding).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if err := tx.WithContext(ctx).
		Exec(
			`UPDATE devices SET activate_flag = 'active', is_enabled = 'enabled', activate_at = ?, update_at = ? WHERE id = ? AND tenant_id = ?`,
			now, now, row.DeviceID, claims.TenantID,
		).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

// GetProvisionInfo 按 item_uuid 查询设备信息（用于“扫码 UUID”路径）
func (*DeviceProvision) GetProvisionInfo(ctx context.Context, req model.DeviceProvisionInfoReq, claims *utils.UserClaims) (*model.DeviceProvisionInfoResp, error) {
	svc := &DeviceProvision{}
	req.ItemUUID = normalizeItemUUID(req.ItemUUID)
	row, err := svc.findDeviceByItemUUID(ctx, req.ItemUUID, claims)
	if err != nil {
		if isNotFoundErr(err) && allowLegacyAutoRegister() {
			return &model.DeviceProvisionInfoResp{
				Exists:             false,
				CanAutoRegister:    true,
				AutoRegisterReason: autoRegisterReasonPtr(),
				DeviceNumber:       strings.TrimSpace(req.ItemUUID),
				IsBound:            false,
			}, nil
		}
		return nil, err
	}

	var cnt int64
	if claims != nil && claims.ID != "" {
		viewCtx, err := resolveAppDeviceViewContext(ctx, claims)
		if err != nil {
			return nil, err
		}
		tableName := "device_user_bindings"
		if viewCtx.userKind == model.UserKindOrgUser {
			tableName = "app_device_added_records"
		}
		if err := global.DB.WithContext(ctx).
			Table(tableName).
			Where("device_id = ? AND user_id = ?", row.DeviceID, claims.ID).
			Count(&cnt).Error; err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
	}

	return &model.DeviceProvisionInfoResp{
		Exists:          true,
		CanAutoRegister: false,
		DeviceID:        row.DeviceID,
		DeviceNumber:    row.DeviceNumber,
		DeviceName:      row.DeviceName,
		BleMac:          row.BleMac,
		CommChipID:      row.CommChipID,
		BmsCommType:     row.BmsCommType,
		IsBound:         cnt > 0,
	}, nil
}

// BindByItemUUID 按 item_uuid 将设备绑定到当前账号
func (*DeviceProvision) BindByItemUUID(ctx context.Context, req model.DeviceProvisionBindReq, claims *utils.UserClaims) (*model.DeviceProvisionBindResp, error) {
	svc := &DeviceProvision{}
	req.ItemUUID = normalizeItemUUID(req.ItemUUID)
	row, _, err := svc.findOrCreateDeviceByItemUUID(ctx, req, claims)
	if err != nil {
		return nil, err
	}

	// 记录 ble_mac（可选，便于后续 BLE 优先连接与排错）
	if req.BleMac != nil && strings.TrimSpace(*req.BleMac) != "" {
		newMac, err := normalizeMac12(*req.BleMac)
		if err != nil {
			return nil, err
		}

		shouldRepairMac := false
		if row.BleMac != nil && strings.TrimSpace(*row.BleMac) != "" {
			existingMac, err := normalizeMac12(*row.BleMac)
			if err == nil {
				if existingMac != newMac {
					return nil, errcode.NewWithMessage(errcode.CodeParamError, "ble_mac mismatch with existing record")
				}
				if strings.TrimSpace(*row.BleMac) != newMac {
					// 旧值可能包含分隔符/小写/尾部补零，统一修正为 12 位大写 HEX。
					shouldRepairMac = true
				}
			} else {
				// 旧值格式异常（例如尾部补零的 10-byte 展示串），用本次提交的合法 MAC 修复。
				shouldRepairMac = true
			}
		} else {
			shouldRepairMac = true
		}
		if shouldRepairMac {
			if err := global.DB.WithContext(ctx).
				Exec("UPDATE device_batteries SET ble_mac = ? WHERE device_id = ?", newMac, row.DeviceID).Error; err != nil {
				return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
		}
	}

	viewCtx, err := resolveAppDeviceViewContext(ctx, claims)
	if err != nil {
		return nil, err
	}

	if viewCtx.userKind == model.UserKindOrgUser {
		if !viewCtx.isFactory && viewCtx.orgID != "" && row.OwnerOrgID != nil && strings.TrimSpace(*row.OwnerOrgID) != "" {
			if !canAccessOrg(claims.TenantID, viewCtx.orgID, strings.TrimSpace(*row.OwnerOrgID)) {
				return nil, errcode.New(errcode.CodeNoPermission)
			}
		}
		source := "UUID_SCAN"
		if req.BleMac != nil && strings.TrimSpace(*req.BleMac) != "" {
			source = "BLE_SCAN"
		}
		if err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return svc.upsertOrgAddedDeviceRecordTx(ctx, tx, claims, row.DeviceID, source)
		}); err != nil {
			if e, ok := err.(*errcode.Error); ok && e.Code == errcode.CodeDBError {
				sqlErr := ""
				if m, ok := e.Data.(map[string]interface{}); ok {
					if v, ok := m["sql_error"].(string); ok {
						sqlErr = v
					}
				}
				if strings.Contains(sqlErr, "app_device_added_records") && strings.Contains(sqlErr, "does not exist") {
					return nil, errcode.NewWithMessage(errcode.CodeDBError, "数据库缺少表 app_device_added_records，请先执行迁移脚本 backend/sql/43.sql")
				}
			}
			return nil, err
		}
		return &model.DeviceProvisionBindResp{
			DeviceID:     row.DeviceID,
			DeviceNumber: row.DeviceNumber,
		}, nil
	}

	if err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return svc.bindEndUserDeviceTx(ctx, tx, row, claims)
	}); err != nil {
		// 针对“数据库错误”做更可读的提示（用于测试环境快速定位迁移/表缺失问题）
		if e, ok := err.(*errcode.Error); ok && e.Code == errcode.CodeDBError {
			sqlErr := ""
			if m, ok := e.Data.(map[string]interface{}); ok {
				if v, ok := m["sql_error"].(string); ok {
					sqlErr = v
				}
			}
			if strings.Contains(sqlErr, "device_user_bindings") && strings.Contains(sqlErr, "does not exist") {
				return nil, errcode.NewWithMessage(errcode.CodeDBError, "数据库缺少表 device_user_bindings，请先执行迁移脚本 backend/sql/13.sql")
			}
			if sqlErr != "" {
				msg := sqlErr
				if len(msg) > 180 {
					msg = msg[:180] + "..."
				}
				return nil, errcode.NewWithMessage(errcode.CodeDBError, "数据库错误: "+msg)
			}
		}
		return nil, err
	}

	return &model.DeviceProvisionBindResp{
		DeviceID:     row.DeviceID,
		DeviceNumber: row.DeviceNumber,
	}, nil
}
