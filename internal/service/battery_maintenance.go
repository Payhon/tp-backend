package service

import (
	"context"
	"encoding/json"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

// BatteryMaintenance 电池维保记录（手动录入）
type BatteryMaintenance struct{}

func (*BatteryMaintenance) Create(ctx context.Context, req model.BatteryMaintenanceCreateReq, claims *utils.UserClaims, dealerScopeID string) error {
	// 查 device_id
	var device struct {
		ID           string
		DeviceNumber string
		TenantID     string
	}
	if err := global.DB.WithContext(ctx).
		Table("devices").
		Select("id, device_number, tenant_id").
		Where("tenant_id = ? AND device_number = ?", claims.TenantID, req.DeviceNumber).
		Scan(&device).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if device.ID == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "device not found"})
	}

	// 经销商数据范围校验（只允许写名下设备）
	if dealerScopeID != "" {
		var cnt int64
		if err := global.DB.WithContext(ctx).
			Table("device_batteries").
			Where("device_id = ? AND dealer_id = ?", device.ID, dealerScopeID).
			Count(&cnt).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if cnt == 0 {
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "no permission"})
		}
	}

	var partsJSON *string
	if len(req.Parts) > 0 {
		if b, err := json.Marshal(req.Parts); err == nil {
			s := string(b)
			partsJSON = &s
		}
	}

	now := time.Now().UTC()
	rec := &model.BatteryMaintenanceRecord{
		ID:             uuid.New(),
		TenantID:       claims.TenantID,
		DeviceID:       device.ID,
		FaultType:      req.FaultType,
		MaintainAt:     req.MaintainAt.UTC(),
		Maintainer:     req.Maintainer,
		Solution:       req.Solution,
		Parts:          partsJSON,
		AffectWarranty: req.AffectWarranty,
		Remark:         req.Remark,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	if err := global.DB.WithContext(ctx).Create(rec).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func (*BatteryMaintenance) List(ctx context.Context, req model.BatteryMaintenanceListReq, claims *utils.UserClaims, dealerScopeID string) (*model.BatteryMaintenanceListResp, error) {
	type row struct {
		ID             string     `gorm:"column:id"`
		DeviceID       string     `gorm:"column:device_id"`
		DeviceNumber   string     `gorm:"column:device_number"`
		BatteryModel   *string    `gorm:"column:battery_model"`
		FaultType      string     `gorm:"column:fault_type"`
		MaintainAt     time.Time  `gorm:"column:maintain_at"`
		Maintainer     string     `gorm:"column:maintainer"`
		Solution       *string    `gorm:"column:solution"`
		Parts          *string    `gorm:"column:parts"`
		AffectWarranty bool       `gorm:"column:affect_warranty"`
		Remark         *string    `gorm:"column:remark"`
		CreatedAt      time.Time  `gorm:"column:created_at"`
	}

	db := global.DB.WithContext(ctx).Table("battery_maintenance_records AS bmr").
		Joins("LEFT JOIN devices d ON d.id = bmr.device_id").
		Joins("LEFT JOIN device_batteries dbat ON dbat.device_id = d.id").
		Joins("LEFT JOIN battery_models bm ON bm.id = dbat.battery_model_id").
		Where("bmr.tenant_id = ?", claims.TenantID)

	if dealerScopeID != "" {
		db = db.Where("dbat.dealer_id = ?", dealerScopeID)
	}
	if req.DeviceNumber != nil && *req.DeviceNumber != "" {
		db = db.Where("d.device_number LIKE ?", "%"+*req.DeviceNumber+"%")
	}
	if req.FaultType != nil && *req.FaultType != "" {
		db = db.Where("bmr.fault_type LIKE ?", "%"+*req.FaultType+"%")
	}
	if req.StartTime != nil && req.EndTime != nil {
		db = db.Where("bmr.maintain_at BETWEEN ? AND ?", req.StartTime.UTC(), req.EndTime.UTC())
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (req.Page - 1) * req.PageSize
	var rows []row
	if err := db.Select(`
  bmr.id,
  bmr.device_id,
  d.device_number AS device_number,
  bm.name AS battery_model,
  bmr.fault_type,
  bmr.maintain_at,
  bmr.maintainer,
  bmr.solution,
  bmr.parts,
  bmr.affect_warranty,
  bmr.remark,
  bmr.created_at
`).
		Order("bmr.maintain_at DESC").
		Limit(req.PageSize).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	out := make([]model.BatteryMaintenanceItemResp, 0, len(rows))
	for _, r := range rows {
		var parts []string
		if r.Parts != nil && *r.Parts != "" {
			_ = json.Unmarshal([]byte(*r.Parts), &parts)
		}
		out = append(out, model.BatteryMaintenanceItemResp{
			ID:             r.ID,
			DeviceID:       r.DeviceID,
			DeviceNumber:   r.DeviceNumber,
			BatteryModel:   r.BatteryModel,
			FaultType:      r.FaultType,
			MaintainAt:     r.MaintainAt.In(time.Local).Format("2006-01-02 15:04:05"),
			Maintainer:     r.Maintainer,
			Solution:       r.Solution,
			Parts:          parts,
			AffectWarranty: r.AffectWarranty,
			Remark:         r.Remark,
			CreatedAt:      r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
		})
	}

	return &model.BatteryMaintenanceListResp{
		List:     out,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (*BatteryMaintenance) Detail(ctx context.Context, id string, claims *utils.UserClaims, dealerScopeID string) (*model.BatteryMaintenanceItemResp, error) {
	// 直接复用 List 的 join 逻辑
	req := model.BatteryMaintenanceListReq{
		PageReq: model.PageReq{Page: 1, PageSize: 1},
	}
	_ = req

	type row struct {
		ID             string     `gorm:"column:id"`
		DeviceID       string     `gorm:"column:device_id"`
		DeviceNumber   string     `gorm:"column:device_number"`
		BatteryModel   *string    `gorm:"column:battery_model"`
		FaultType      string     `gorm:"column:fault_type"`
		MaintainAt     time.Time  `gorm:"column:maintain_at"`
		Maintainer     string     `gorm:"column:maintainer"`
		Solution       *string    `gorm:"column:solution"`
		Parts          *string    `gorm:"column:parts"`
		AffectWarranty bool       `gorm:"column:affect_warranty"`
		Remark         *string    `gorm:"column:remark"`
		CreatedAt      time.Time  `gorm:"column:created_at"`
	}

	db := global.DB.WithContext(ctx).Table("battery_maintenance_records AS bmr").
		Joins("LEFT JOIN devices d ON d.id = bmr.device_id").
		Joins("LEFT JOIN device_batteries dbat ON dbat.device_id = d.id").
		Joins("LEFT JOIN battery_models bm ON bm.id = dbat.battery_model_id").
		Where("bmr.tenant_id = ? AND bmr.id = ?", claims.TenantID, id)

	if dealerScopeID != "" {
		db = db.Where("dbat.dealer_id = ?", dealerScopeID)
	}

	var r row
	if err := db.Select(`
  bmr.id,
  bmr.device_id,
  d.device_number AS device_number,
  bm.name AS battery_model,
  bmr.fault_type,
  bmr.maintain_at,
  bmr.maintainer,
  bmr.solution,
  bmr.parts,
  bmr.affect_warranty,
  bmr.remark,
  bmr.created_at
`).Scan(&r).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if r.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var parts []string
	if r.Parts != nil && *r.Parts != "" {
		_ = json.Unmarshal([]byte(*r.Parts), &parts)
	}
	return &model.BatteryMaintenanceItemResp{
		ID:             r.ID,
		DeviceID:       r.DeviceID,
		DeviceNumber:   r.DeviceNumber,
		BatteryModel:   r.BatteryModel,
		FaultType:      r.FaultType,
		MaintainAt:     r.MaintainAt.In(time.Local).Format("2006-01-02 15:04:05"),
		Maintainer:     r.Maintainer,
		Solution:       r.Solution,
		Parts:          parts,
		AffectWarranty: r.AffectWarranty,
		Remark:         r.Remark,
		CreatedAt:      r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
	}, nil
}

