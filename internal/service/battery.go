package service

import (
	"context"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"
)

// Battery BMS: 电池管理（电池列表/导入导出等）
type Battery struct{}

type batteryListRow struct {
	DeviceID     string  `gorm:"column:device_id"`
	DeviceNumber string  `gorm:"column:device_number"`
	DeviceName   *string `gorm:"column:device_name"`

	BatteryModelID   *string `gorm:"column:battery_model_id"`
	BatteryModelName *string `gorm:"column:battery_model_name"`

	ProductionDate     *time.Time `gorm:"column:production_date"`
	WarrantyExpireDate *time.Time `gorm:"column:warranty_expire_date"`

	DealerID   *string `gorm:"column:dealer_id"`
	DealerName *string `gorm:"column:dealer_name"`

	UserID    *string `gorm:"column:user_id"`
	UserName  *string `gorm:"column:user_name"`
	UserPhone *string `gorm:"column:user_phone"`

	ActivationDate   *time.Time `gorm:"column:activation_date"`
	ActivationStatus *string    `gorm:"column:activation_status"`

	IsOnline       int16    `gorm:"column:is_online"`
	Soc            *float64 `gorm:"column:soc"`
	Soh            *float64 `gorm:"column:soh"`
	CurrentVersion *string  `gorm:"column:current_version"`
	TransferStatus *string  `gorm:"column:transfer_status"`
}

// GetBatteryList 获取电池列表（厂家/经销商视角）
func (*Battery) GetBatteryList(ctx context.Context, req model.BatteryListReq, claims *utils.UserClaims, dealerID string) (*model.BatteryListResp, error) {
	db := global.DB.WithContext(ctx)

	// 以 devices 作为 tenant 过滤主表
	queryBuilder := db.Table("devices AS d").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			d.name AS device_name,
			dbat.battery_model_id AS battery_model_id,
			bm.name AS battery_model_name,
			dbat.production_date AS production_date,
			dbat.warranty_expire_date AS warranty_expire_date,
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
		Joins(`LEFT JOIN dealers AS de ON de.id = dbat.dealer_id`).
		// 仅取主用户（is_owner=true），若无则为空
		Joins(`LEFT JOIN device_user_bindings AS dub ON dub.device_id = d.id AND dub.is_owner = true`).
		Joins(`LEFT JOIN users AS u ON u.id = dub.user_id`).
		Where("d.tenant_id = ?", claims.TenantID)

	// 经销商数据隔离：dealerID 不为空时只看名下设备
	if dealerID != "" {
		queryBuilder = queryBuilder.Where("dbat.dealer_id = ?", dealerID)
	}

	// 条件筛选
	if req.DeviceNumber != nil && *req.DeviceNumber != "" {
		queryBuilder = queryBuilder.Where("d.device_number ILIKE ?", "%"+*req.DeviceNumber+"%")
	}
	if req.BatteryModelID != nil && *req.BatteryModelID != "" {
		queryBuilder = queryBuilder.Where("dbat.battery_model_id = ?", *req.BatteryModelID)
	}
	if req.IsOnline != nil {
		queryBuilder = queryBuilder.Where("d.is_online = ?", *req.IsOnline)
	}
	if req.ActivationStatus != nil && *req.ActivationStatus != "" {
		queryBuilder = queryBuilder.Where("dbat.activation_status = ?", *req.ActivationStatus)
	}
	if req.DealerID != nil && *req.DealerID != "" {
		// 厂家侧可按 dealer_id 过滤；经销商侧该条件与 dealerID 一致/更严
		queryBuilder = queryBuilder.Where("dbat.dealer_id = ?", *req.DealerID)
	}

	// 出厂日期范围（YYYY-MM-DD）
	if req.ProductionDateStart != nil && *req.ProductionDateStart != "" {
		if t, err := time.ParseInLocation("2006-01-02", *req.ProductionDateStart, time.Local); err == nil {
			queryBuilder = queryBuilder.Where("dbat.production_date >= ?", t)
		}
	}
	if req.ProductionDateEnd != nil && *req.ProductionDateEnd != "" {
		if t, err := time.ParseInLocation("2006-01-02", *req.ProductionDateEnd, time.Local); err == nil {
			// end-of-day
			queryBuilder = queryBuilder.Where("dbat.production_date < ?", t.Add(24*time.Hour))
		}
	}

	// 质保状态（IN/OVER）
	if req.WarrantyStatus != nil && *req.WarrantyStatus != "" {
		now := time.Now()
		switch *req.WarrantyStatus {
		case "IN":
			queryBuilder = queryBuilder.Where("dbat.warranty_expire_date IS NOT NULL AND dbat.warranty_expire_date >= ?", now)
		case "OVER":
			queryBuilder = queryBuilder.Where("dbat.warranty_expire_date IS NOT NULL AND dbat.warranty_expire_date < ?", now)
		}
	}

	// 统计总数
	var total int64
	if err := queryBuilder.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 分页
	offset := (req.Page - 1) * req.PageSize
	rows := make([]batteryListRow, 0, req.PageSize)
	if err := queryBuilder.
		Order("d.created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 构建响应
	list := make([]model.BatteryListItemResp, 0, len(rows))
	for _, r := range rows {
		item := model.BatteryListItemResp{
			DeviceID:         r.DeviceID,
			DeviceNumber:     r.DeviceNumber,
			DeviceName:       r.DeviceName,
			BatteryModelID:   r.BatteryModelID,
			BatteryModelName: r.BatteryModelName,
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

		list = append(list, item)
	}

	return &model.BatteryListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
