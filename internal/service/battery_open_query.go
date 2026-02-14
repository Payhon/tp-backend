package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"project/internal/model"
	"project/pkg/errcode"
	"project/pkg/global"
	"project/pkg/utils"
)

func buildBatteryListItemRespFromRow(r batteryListRow) *model.BatteryListItemResp {
	item := &model.BatteryListItemResp{
		DeviceID:         r.DeviceID,
		DeviceNumber:     r.DeviceNumber,
		DeviceName:       r.DeviceName,
		BatteryModelID:   r.BatteryModelID,
		BatteryModelName: r.BatteryModelName,
		ItemUUID:         r.ItemUUID,
		BatchNumber:      r.BatchNumber,
		BleMac:           r.BleMac,
		CommChipID:       r.CommChipID,
		OwnerOrgID:       r.OwnerOrgID,
		OwnerOrgName:     r.OwnerOrgName,
		OwnerOrgType:     r.OwnerOrgType,
		DealerID:         r.DealerID,
		DealerName:       r.DealerName,
		UserID:           r.UserID,
		UserName:         r.UserName,
		UserPhone:        r.UserPhone,
		ActivationStatus: r.ActivationStatus,
		IsOnline:         r.IsOnline,
		Soc:              r.Soc,
		Soh:              r.Soh,
		CurrentVersion:   r.CurrentVersion,
		TransferStatus:   r.TransferStatus,
	}

	if r.ProductionDate != nil {
		s := r.ProductionDate.Format("2006-01-02")
		item.ProductionDate = &s
	}
	if r.WarrantyExpireDate != nil {
		s := r.WarrantyExpireDate.Format("2006-01-02")
		item.WarrantyExpireDate = &s
	}
	if r.ActivationDate != nil {
		s := r.ActivationDate.Format("2006-01-02 15:04:05")
		item.ActivationDate = &s
	}
	return item
}

// GetBatteryByDeviceNumber 根据序列号查询单个电池详情（第三方 OpenAPI）
func (*Battery) GetBatteryByDeviceNumber(ctx context.Context, deviceNumber string, claims *utils.UserClaims) (*model.BatteryListItemResp, error) {
	deviceNumber = strings.TrimSpace(deviceNumber)
	if deviceNumber == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "device_number is required")
	}

	row := batteryListRow{}
	err := global.DB.WithContext(ctx).Table("devices AS d").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			d.name AS device_name,
			dbat.battery_model_id AS battery_model_id,
			bm.name AS battery_model_name,
			dbat.item_uuid AS item_uuid,
			dbat.batch_number AS batch_number,
			dbat.ble_mac AS ble_mac,
			dbat.comm_chip_id AS comm_chip_id,
			dbat.production_date AS production_date,
			dbat.warranty_expire_date AS warranty_expire_date,
			dbat.owner_org_id AS owner_org_id,
			org.name AS owner_org_name,
			org.org_type AS owner_org_type,
			dbat.dealer_id AS dealer_id,
			de.name AS dealer_name,
			u.id AS user_id,
			u.name AS user_name,
			u.phone_number AS user_phone,
			dbat.activation_date AS activation_date,
			dbat.activation_status AS activation_status,
			d.is_online AS is_online,
			dbat.soc AS soc,
			dbat.soh AS soh,
			d.current_version AS current_version,
			dbat.transfer_status AS transfer_status
		`).
		Joins(`LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id`).
		Joins(`LEFT JOIN battery_models AS bm ON bm.id = dbat.battery_model_id`).
		Joins(`LEFT JOIN orgs AS org ON org.id = dbat.owner_org_id`).
		Joins(`LEFT JOIN dealers AS de ON de.id = dbat.dealer_id`).
		Joins(`LEFT JOIN device_user_bindings AS dub ON dub.device_id = d.id AND dub.is_owner = true`).
		Joins(`LEFT JOIN users AS u ON u.id = dub.user_id`).
		Where("d.tenant_id = ? AND d.device_number = ?", claims.TenantID, deviceNumber).
		Take(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.WithData(errcode.CodeNotFound, map[string]interface{}{
				"message":       "电池不存在",
				"device_number": deviceNumber,
				"ts":            time.Now().UTC().Format(time.RFC3339),
			})
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return buildBatteryListItemRespFromRow(row), nil
}
