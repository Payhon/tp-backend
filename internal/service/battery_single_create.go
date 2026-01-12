package service

import (
	"context"
	"time"

	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"gorm.io/gorm"
)

// CreateSingleBattery BMS：添加/更新单个电池信息（device_batteries）
func (*Battery) CreateSingleBattery(ctx context.Context, req model.BatteryCreateReq, claims *utils.UserClaims, orgID string) (*model.BatteryCreateResp, error) {
	// item_uuid -> devices.device_number
	device, err := query.Device.WithContext(ctx).Where(
		query.Device.DeviceNumber.Eq(req.ItemUUID),
		query.Device.TenantID.Eq(claims.TenantID),
	).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "设备不存在（devices.device_number 未找到）"})
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	// 电池型号：支持 id 或 name
	var batteryModelID *string
	var batteryModelName *string
	var warrantyMonths *int32
	if req.BatteryModelID != nil && *req.BatteryModelID != "" {
		bm, err := query.BatteryModel.WithContext(ctx).Where(
			query.BatteryModel.ID.Eq(*req.BatteryModelID),
			query.BatteryModel.TenantID.Eq(claims.TenantID),
		).First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "电池型号不存在"})
			}
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		batteryModelID = &bm.ID
		batteryModelName = &bm.Name
		warrantyMonths = bm.WarrantyMonth
	} else if req.BatteryModelName != nil && *req.BatteryModelName != "" {
		bm, err := query.BatteryModel.WithContext(ctx).Where(
			query.BatteryModel.Name.Eq(*req.BatteryModelName),
			query.BatteryModel.TenantID.Eq(claims.TenantID),
		).First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "电池型号不存在"})
			}
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		batteryModelID = &bm.ID
		batteryModelName = &bm.Name
		warrantyMonths = bm.WarrantyMonth
	}

	productionDate, err := parseDateYYYYMMDD(ptrToStr(req.ProductionDate))
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "出厂日期格式错误，应为 YYYY-MM-DD"})
	}
	warrantyExpireDate, err := parseDateYYYYMMDD(ptrToStr(req.WarrantyExpireDate))
	if err != nil {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "质保到期格式错误，应为 YYYY-MM-DD"})
	}

	// 若未传质保到期，且传了出厂日期 & 型号带质保月数，则自动推算
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
	if err := upsertDeviceBattery(
		ctx,
		device.ID,
		req.ItemUUID,
		&batchNumber,
		&productSpec,
		&orderNumber,
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
	_ = CreateBatteryOperationLog(ctx, claims.TenantID, device.ID, req.ItemUUID, BatteryOpTypeCreate, &claims.ID, &desc, map[string]any{
		"battery_model_id": batteryModelID,
		"batch_number":     batchNumber,
		"product_spec":     productSpec,
		"order_number":     orderNumber,
	})

	// 查询回显（包含型号名称）
	var row struct {
		BatteryModelID     *string    `gorm:"column:battery_model_id"`
		BatteryModelName   *string    `gorm:"column:battery_model_name"`
		ItemUUID           *string    `gorm:"column:item_uuid"`
		BatchNumber        *string    `gorm:"column:batch_number"`
		ProductSpec        *string    `gorm:"column:product_spec"`
		OrderNumber        *string    `gorm:"column:order_number"`
		BleMac             *string    `gorm:"column:ble_mac"`
		CommChipID         *string    `gorm:"column:comm_chip_id"`
		ProductionDate     *time.Time `gorm:"column:production_date"`
		WarrantyExpireDate *time.Time `gorm:"column:warranty_expire_date"`
	}
	_ = global.DB.WithContext(ctx).Table("device_batteries AS dbat").
		Select(`dbat.battery_model_id, bm.name AS battery_model_name, dbat.item_uuid, dbat.batch_number, dbat.product_spec, dbat.order_number, dbat.ble_mac, dbat.comm_chip_id, dbat.production_date, dbat.warranty_expire_date`).
		Joins(`LEFT JOIN battery_models bm ON bm.id = dbat.battery_model_id`).
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
