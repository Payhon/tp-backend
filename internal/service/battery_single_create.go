package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"project/initialize"
	"project/internal/dal"
	"project/internal/model"
	"project/internal/query"
	protocolplugin "project/internal/service/protocol_plugin"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func findUniquePackFactoryByName(ctx context.Context, tenantID, rawName string) (*model.Org, bool, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return nil, false, nil
	}

	orgs, err := query.Org.WithContext(ctx).
		Where(
			query.Org.TenantID.Eq(tenantID),
			query.Org.OrgType.Eq(model.OrgTypePACKFactory),
			query.Org.Name.Eq(name),
		).
		Find()
	if err != nil {
		return nil, false, err
	}
	if len(orgs) != 1 {
		return nil, false, nil
	}
	return orgs[0], true, nil
}

// CreateSingleBattery BMS：添加/更新单个电池信息（device_batteries）
func (*Battery) CreateSingleBattery(ctx context.Context, req model.BatteryCreateReq, claims *utils.UserClaims, orgID string) (*model.BatteryCreateResp, error) {
	// 电池型号：兼容历史 PACK 型号与当前 BMS 型号
	var batteryModelID *string
	var batteryModelName *string
	var warrantyMonths *int32
	var deviceConfigID *string

	resolveBmsModelMeta := func(packModelID, packModelName string) error {
		if bmsModel, err := getBmsBatteryModelByID(ctx, claims.TenantID, packModelID); err == nil {
			warrantyMonths = bmsModel.WarrantyMonth
			deviceConfigID = bmsModel.DeviceConfigID
			return nil
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		if bmsModel, err := getBmsBatteryModelByName(ctx, claims.TenantID, packModelName); err == nil {
			warrantyMonths = bmsModel.WarrantyMonth
			deviceConfigID = bmsModel.DeviceConfigID
			return nil
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
		return nil
	}

	if req.BatteryModelID != nil && *req.BatteryModelID != "" {
		if bmsModel, err := getBmsBatteryModelByID(ctx, claims.TenantID, *req.BatteryModelID); err == nil {
			batteryModelID = &bmsModel.ID
			batteryModelName = &bmsModel.Name
			warrantyMonths = bmsModel.WarrantyMonth
			deviceConfigID = bmsModel.DeviceConfigID
		} else if err != gorm.ErrRecordNotFound {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		} else {
			bm, err := getPackBatteryModelByID(ctx, claims.TenantID, *req.BatteryModelID)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "BMS型号不存在"})
				}
				return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
			batteryModelID = &bm.ID
			batteryModelName = &bm.Name
			if err := resolveBmsModelMeta(bm.ID, bm.Name); err != nil {
				return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
		}
	} else if req.BatteryModelName != nil && *req.BatteryModelName != "" {
		if bmsModel, err := getBmsBatteryModelByName(ctx, claims.TenantID, *req.BatteryModelName); err == nil {
			batteryModelID = &bmsModel.ID
			batteryModelName = &bmsModel.Name
			warrantyMonths = bmsModel.WarrantyMonth
			deviceConfigID = bmsModel.DeviceConfigID
		} else if err != gorm.ErrRecordNotFound {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		} else {
			bm, err := getPackBatteryModelByName(ctx, claims.TenantID, *req.BatteryModelName)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "BMS型号不存在"})
				}
				return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
			batteryModelID = &bm.ID
			batteryModelName = &bm.Name
			if err := resolveBmsModelMeta(bm.ID, bm.Name); err != nil {
				return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
		}
	}

	// item_uuid -> devices.device_number
	device, createdDevice, err := getOrCreateDeviceByNumberForBattery(ctx, claims, req.ItemUUID, deviceConfigID)
	if err != nil {
		return nil, err
	}

	productionDate, err := parseDateYYYYMMDD(ptrToStr(req.ProductionDate))
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "出厂日期格式错误，应为 YYYY-MM-DD"})
	}
	warrantyExpireDate, err := parseDateYYYYMMDD(ptrToStr(req.WarrantyExpireDate))
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "质保到期格式错误，应为 YYYY-MM-DD"})
	}

	// 若未传质保到期，且传了出厂日期 & BMS 板型号带质保月数，则自动推算
	if warrantyExpireDate == nil && productionDate != nil && warrantyMonths != nil && *warrantyMonths > 0 {
		t := productionDate.AddDate(0, int(*warrantyMonths), 0)
		warrantyExpireDate = &t
	}

	var ownerOrgID *string
	if orgID != "" {
		ownerOrgID = &orgID
	}

	bleMac := req.BleMac
	commChipID := req.CommChipID
	batchNumber := req.BatchNumber
	productSpec := req.ProductSpec
	orderNumber := req.OrderNumber
	bmsCommType := req.BmsCommType
	if err := upsertDeviceBattery(
		ctx,
		device.ID,
		req.ItemUUID,
		&batchNumber,
		&productSpec,
		&orderNumber,
		&bmsCommType,
		batteryModelID,
		bleMac,
		commChipID,
		productionDate,
		warrantyExpireDate,
		ownerOrgID,
	); err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if req.Remark != nil {
		if err := global.DB.WithContext(ctx).
			Model(&model.Device{}).
			Where("id = ?", device.ID).
			Update("remark1", *req.Remark).Error; err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
	}

	// 运营日志：CREATE
	desc := "添加单个电池信息"
	if createdDevice {
		desc = "添加单个电池信息（自动创建 devices 记录）"
	}
	_ = CreateBatteryOperationLog(ctx, claims.TenantID, device.ID, req.ItemUUID, BatteryOpTypeCreate, &claims.ID, &desc, map[string]any{
		"battery_model_id": batteryModelID,
		"batch_number":     batchNumber,
		"product_spec":     productSpec,
		"order_number":     orderNumber,
		"bms_comm_type":    bmsCommType,
		"remark":           req.Remark,
		"created_device":   createdDevice,
	})

	if req.PackFactoryName != nil && strings.TrimSpace(*req.PackFactoryName) != "" {
		packFactoryOrg, matched, err := findUniquePackFactoryByName(ctx, claims.TenantID, *req.PackFactoryName)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if matched && packFactoryOrg != nil {
			if err := GroupApp.Battery.FactoryOutBattery(ctx, model.BatteryFactoryOutReq{
				DeviceID: device.ID,
				ToOrgID:  packFactoryOrg.ID,
			}, claims, ""); err != nil {
				return nil, err
			}
		} else {
			desc := fmt.Sprintf("OpenAPI 自动出厂跳过：PACK厂家名称未唯一匹配（%s）", strings.TrimSpace(*req.PackFactoryName))
			_ = CreateBatteryOperationLog(ctx, claims.TenantID, device.ID, req.ItemUUID, BatteryOpTypeCreate, &claims.ID, &desc, map[string]any{
				"pack_factory_name": strings.TrimSpace(*req.PackFactoryName),
				"auto_factory_out":  "skipped",
			})
		}
	}

	// 查询回显（包含型号名称）
	return loadBatteryCreateRespByDeviceID(ctx, device.ID, device.DeviceNumber, batteryModelName), nil
}

// UpdateSingleBattery 编辑单个电池信息（对应“新增 BMS”表单字段）
func (*Battery) UpdateSingleBattery(ctx context.Context, deviceID string, req model.BatteryCreateReq, claims *utils.UserClaims, orgID string) (*model.BatteryCreateResp, error) {
	_, err := query.Device.WithContext(ctx).Where(
		query.Device.ID.Eq(deviceID),
		query.Device.TenantID.Eq(claims.TenantID),
	).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "设备不存在"})
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if err := checkDeviceOrgAccess(ctx, deviceID, claims.TenantID, orgID); err != nil {
		return nil, err
	}

	batteryModelID, batteryModelName, warrantyMonths, deviceConfigID, err := resolveBatteryCreateModelMeta(ctx, claims, req)
	if err != nil {
		return nil, err
	}

	itemUUID := strings.TrimSpace(req.ItemUUID)
	if itemUUID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "item_uuid is required"})
	}
	if err := ensureBatteryDeviceNumberUsable(ctx, claims.TenantID, deviceID, itemUUID); err != nil {
		return nil, err
	}

	productionDate, err := parseDateYYYYMMDD(ptrToStr(req.ProductionDate))
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "出厂日期格式错误，应为 YYYY-MM-DD"})
	}
	warrantyExpireDate, err := parseDateYYYYMMDD(ptrToStr(req.WarrantyExpireDate))
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "质保到期格式错误，应为 YYYY-MM-DD"})
	}
	if warrantyExpireDate == nil && productionDate != nil && warrantyMonths != nil && *warrantyMonths > 0 {
		t := productionDate.AddDate(0, int(*warrantyMonths), 0)
		warrantyExpireDate = &t
	}

	now := time.Now().UTC()
	updates := map[string]any{
		"device_number": itemUUID,
		"update_at":     &now,
		"remark1":       req.Remark,
	}
	if deviceConfigID != nil && *deviceConfigID != "" {
		updates["device_config_id"] = *deviceConfigID
	}
	if err := global.DB.WithContext(ctx).Model(&model.Device{}).Where("id = ?", deviceID).Updates(updates).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	bleMac := req.BleMac
	commChipID := req.CommChipID
	batchNumber := req.BatchNumber
	productSpec := req.ProductSpec
	orderNumber := req.OrderNumber
	bmsCommType := req.BmsCommType
	if err := upsertDeviceBattery(
		ctx,
		deviceID,
		itemUUID,
		&batchNumber,
		&productSpec,
		&orderNumber,
		&bmsCommType,
		batteryModelID,
		bleMac,
		commChipID,
		productionDate,
		warrantyExpireDate,
		nil,
	); err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	desc := "编辑 BMS 信息"
	_ = CreateBatteryOperationLog(ctx, claims.TenantID, deviceID, itemUUID, BatteryOpTypeEditInfo, &claims.ID, &desc, map[string]any{
		"battery_model_id": batteryModelID,
		"batch_number":     batchNumber,
		"product_spec":     productSpec,
		"order_number":     orderNumber,
		"bms_comm_type":    bmsCommType,
		"remark":           req.Remark,
	})

	return loadBatteryCreateRespByDeviceID(ctx, deviceID, itemUUID, batteryModelName), nil
}

// DeleteBattery 删除电池及其关联业务数据（保留运营日志审计）
func (*Battery) DeleteBattery(ctx context.Context, deviceID string, claims *utils.UserClaims, orgID string) error {
	device, err := query.Device.WithContext(ctx).Where(
		query.Device.ID.Eq(deviceID),
		query.Device.TenantID.Eq(claims.TenantID),
	).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "设备不存在"})
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if err := checkDeviceOrgAccess(ctx, deviceID, claims.TenantID, orgID); err != nil {
		return err
	}

	subDevices, err := dal.GetSubDeviceListByParentID(deviceID)
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if len(subDevices) > 0 {
		return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "设备存在子设备，无法删除"})
	}
	conditions, err := dal.GetDeviceTriggerConditionListByDeviceId(deviceID)
	if err != nil {
		return err
	}
	if len(conditions) > 0 {
		return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "设备已关联场景联动，请先解除"})
	}

	desc := "删除电池及关联业务数据"
	err = global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tables := []string{
			"telemetry_current_datas",
			"telemetry_datas",
			"telemetry_set_logs",
			"attribute_datas",
			"attribute_set_logs",
			"event_datas",
			"command_set_logs",
			"expected_datas",
			"device_user_bindings",
			"device_battery_tags",
			"battery_maintenance_records",
			"warranty_applications",
			"offline_command_tasks",
			"ota_upgrade_task_details",
			"device_org_transfers",
			"device_transfers",
			"r_group_device",
			"device_status_history",
			"device_batteries",
		}
		for _, tableName := range tables {
			if err := tx.Table(tableName).Where("device_id = ?", deviceID).Delete(nil).Error; err != nil {
				return err
			}
		}
		if err := tx.Table("devices").Where("id = ? AND tenant_id = ?", deviceID, claims.TenantID).Delete(nil).Error; err != nil {
			return err
		}
		return CreateBatteryOperationLogTx(tx, claims.TenantID, deviceID, device.DeviceNumber, BatteryOpTypeDelete, &claims.ID, &desc, map[string]any{
			"deleted_tables": []string{
				"telemetry_current_datas",
				"telemetry_datas",
				"telemetry_set_logs",
				"attribute_datas",
				"attribute_set_logs",
				"event_datas",
				"command_set_logs",
				"expected_datas",
				"device_user_bindings",
				"device_battery_tags",
				"battery_maintenance_records",
				"warranty_applications",
				"offline_command_tasks",
				"ota_upgrade_task_details",
				"device_org_transfers",
				"device_transfers",
				"r_group_device",
				"device_status_history",
				"device_batteries",
				"devices",
			},
		})
	})
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	initialize.DelDeviceCache(deviceID)
	global.REDIS.Del(context.Background(), device.Voucher)
	if protocolplugin.DisconnectDeviceByDeviceID(deviceID) != nil {
		logrus.Error("DisconnectDeviceByDeviceID failed")
	}
	return nil
}

func loadBatteryCreateRespByDeviceID(ctx context.Context, deviceID, deviceNumber string, fallbackBatteryModelName *string) *model.BatteryCreateResp {
	var row struct {
		BatteryModelID     *string    `gorm:"column:battery_model_id"`
		BatteryModelName   *string    `gorm:"column:battery_model_name"`
		ItemUUID           *string    `gorm:"column:item_uuid"`
		BatchNumber        *string    `gorm:"column:batch_number"`
		ProductSpec        *string    `gorm:"column:product_spec"`
		OrderNumber        *string    `gorm:"column:order_number"`
		BmsCommType        *int       `gorm:"column:bms_comm_type"`
		BleMac             *string    `gorm:"column:ble_mac"`
		CommChipID         *string    `gorm:"column:comm_chip_id"`
		ProductionDate     *time.Time `gorm:"column:production_date"`
		WarrantyExpireDate *time.Time `gorm:"column:warranty_expire_date"`
	}
	_ = global.DB.WithContext(ctx).Table("device_batteries AS dbat").
		Select(`dbat.battery_model_id, COALESCE(bm_pack.name, bm_bms.name) AS battery_model_name, dbat.item_uuid, dbat.batch_number, dbat.product_spec, dbat.order_number, dbat.bms_comm_type, dbat.ble_mac, dbat.comm_chip_id, dbat.production_date, dbat.warranty_expire_date`).
		Joins(`LEFT JOIN battery_models bm_pack ON bm_pack.id = dbat.battery_model_id`).
		Joins(`LEFT JOIN battery_bms_models bm_bms ON bm_bms.id = dbat.battery_model_id`).
		Where("dbat.device_id = ?", deviceID).
		Scan(&row).Error

	var productionDateStr *string
	if row.ProductionDate != nil {
		s := row.ProductionDate.Format("2006-01-02")
		productionDateStr = &s
	}
	var warrantyExpireDateStr *string
	if row.WarrantyExpireDate != nil {
		s := row.WarrantyExpireDate.Format("2006-01-02")
		warrantyExpireDateStr = &s
	}

	// 优先使用查询结果的型号名，否则用解析时获取的
	if row.BatteryModelName != nil {
		fallbackBatteryModelName = row.BatteryModelName
	}

	return &model.BatteryCreateResp{
		DeviceID:           deviceID,
		DeviceNumber:       deviceNumber,
		BatteryModelID:     row.BatteryModelID,
		BatteryModelName:   fallbackBatteryModelName,
		ItemUUID:           row.ItemUUID,
		BatchNumber:        row.BatchNumber,
		ProductSpec:        row.ProductSpec,
		OrderNumber:        row.OrderNumber,
		BmsCommType:        row.BmsCommType,
		BleMac:             row.BleMac,
		CommChipID:         row.CommChipID,
		ProductionDate:     productionDateStr,
		WarrantyExpireDate: warrantyExpireDateStr,
	}
}

func resolveBatteryCreateModelMeta(ctx context.Context, claims *utils.UserClaims, req model.BatteryCreateReq) (*string, *string, *int32, *string, error) {
	var batteryModelID *string
	var batteryModelName *string
	var warrantyMonths *int32
	var deviceConfigID *string

	resolveBmsModelMeta := func(packModelID, packModelName string) error {
		if bmsModel, err := getBmsBatteryModelByID(ctx, claims.TenantID, packModelID); err == nil {
			warrantyMonths = bmsModel.WarrantyMonth
			deviceConfigID = bmsModel.DeviceConfigID
			return nil
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		if bmsModel, err := getBmsBatteryModelByName(ctx, claims.TenantID, packModelName); err == nil {
			warrantyMonths = bmsModel.WarrantyMonth
			deviceConfigID = bmsModel.DeviceConfigID
			return nil
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
		return nil
	}

	if req.BatteryModelID != nil && *req.BatteryModelID != "" {
		if bmsModel, err := getBmsBatteryModelByID(ctx, claims.TenantID, *req.BatteryModelID); err == nil {
			batteryModelID = &bmsModel.ID
			batteryModelName = &bmsModel.Name
			warrantyMonths = bmsModel.WarrantyMonth
			deviceConfigID = bmsModel.DeviceConfigID
		} else if err != gorm.ErrRecordNotFound {
			return nil, nil, nil, nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		} else {
			bm, err := getPackBatteryModelByID(ctx, claims.TenantID, *req.BatteryModelID)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return nil, nil, nil, nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "BMS型号不存在"})
				}
				return nil, nil, nil, nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
			batteryModelID = &bm.ID
			batteryModelName = &bm.Name
			if err := resolveBmsModelMeta(bm.ID, bm.Name); err != nil {
				return nil, nil, nil, nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
		}
	} else if req.BatteryModelName != nil && *req.BatteryModelName != "" {
		if bmsModel, err := getBmsBatteryModelByName(ctx, claims.TenantID, *req.BatteryModelName); err == nil {
			batteryModelID = &bmsModel.ID
			batteryModelName = &bmsModel.Name
			warrantyMonths = bmsModel.WarrantyMonth
			deviceConfigID = bmsModel.DeviceConfigID
		} else if err != gorm.ErrRecordNotFound {
			return nil, nil, nil, nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		} else {
			bm, err := getPackBatteryModelByName(ctx, claims.TenantID, *req.BatteryModelName)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return nil, nil, nil, nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "BMS型号不存在"})
				}
				return nil, nil, nil, nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
			batteryModelID = &bm.ID
			batteryModelName = &bm.Name
			if err := resolveBmsModelMeta(bm.ID, bm.Name); err != nil {
				return nil, nil, nil, nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
		}
	}

	return batteryModelID, batteryModelName, warrantyMonths, deviceConfigID, nil
}

func ensureBatteryDeviceNumberUsable(ctx context.Context, tenantID, currentDeviceID, deviceNumber string) error {
	existing, err := query.Device.WithContext(ctx).Where(query.Device.DeviceNumber.Eq(deviceNumber)).First()
	if err == nil {
		if existing.TenantID != tenantID {
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
				"message": "设备编号已存在（非当前租户），无法保存",
			})
		}
		if existing.ID != currentDeviceID {
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
				"message": "设备编号已存在，无法保存",
			})
		}
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func ptrToStr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func getOrCreateDeviceByNumberForBattery(
	ctx context.Context,
	claims *utils.UserClaims,
	deviceNumber string,
	deviceConfigID *string,
) (*model.Device, bool, error) {
	deviceNumber = strings.TrimSpace(deviceNumber)
	if deviceNumber == "" {
		return nil, false, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "item_uuid is required"})
	}

	// Try find in current tenant first
	device, err := query.Device.WithContext(ctx).Where(
		query.Device.DeviceNumber.Eq(deviceNumber),
		query.Device.TenantID.Eq(claims.TenantID),
	).First()
	if err == nil {
		return device, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	// If device_number exists in other tenant, deny auto create to avoid cross-tenant collision
	other, err := query.Device.WithContext(ctx).Where(query.Device.DeviceNumber.Eq(deviceNumber)).First()
	if err == nil && other != nil && other.ID != "" {
		return nil, false, errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
			"message": "设备编号已存在（非当前租户），无法自动创建 devices 记录",
		})
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, false, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	now := time.Now().UTC()
	name := fmt.Sprintf("Battery_%s", deviceNumber)
	voucher := fmt.Sprintf(`{"username":"%s","password":"%s"}`, uuid.New()[0:22], uuid.New()[0:7])

	dev := &model.Device{
		ID:           uuid.New(),
		Name:         &name,
		Voucher:      voucher,
		TenantID:     claims.TenantID,
		IsEnabled:    "enable",
		ActivateFlag: "active",
		CreatedAt:    &now,
		UpdateAt:     &now,
		DeviceNumber: deviceNumber,
		IsOnline:     0,
		// Keep empty json fields stable
		AdditionalInfo: StringPtr("{}"),
		ProtocolConfig: StringPtr("{}"),
	}
	if deviceConfigID != nil && *deviceConfigID != "" {
		dev.DeviceConfigID = deviceConfigID
	}

	if err := query.Device.WithContext(ctx).Create(dev); err != nil {
		// Handle concurrent create: retry fetch
		device, qerr := query.Device.WithContext(ctx).Where(
			query.Device.DeviceNumber.Eq(deviceNumber),
			query.Device.TenantID.Eq(claims.TenantID),
		).First()
		if qerr == nil {
			return device, false, nil
		}
		return nil, false, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return dev, true, nil
}
