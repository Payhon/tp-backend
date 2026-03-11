package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

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
		"created_device":   createdDevice,
	})

	// 查询回显（包含型号名称）
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
		Where("dbat.device_id = ?", device.ID).
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
		batteryModelName = row.BatteryModelName
	}

	return &model.BatteryCreateResp{
		DeviceID:           device.ID,
		DeviceNumber:       device.DeviceNumber,
		BatteryModelID:     row.BatteryModelID,
		BatteryModelName:   batteryModelName,
		ItemUUID:           row.ItemUUID,
		BatchNumber:        row.BatchNumber,
		ProductSpec:        row.ProductSpec,
		OrderNumber:        row.OrderNumber,
		BmsCommType:        row.BmsCommType,
		BleMac:             row.BleMac,
		CommChipID:         row.CommChipID,
		ProductionDate:     productionDateStr,
		WarrantyExpireDate: warrantyExpireDateStr,
	}, nil
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
