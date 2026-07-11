package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

// BatteryBmsModel BMS 型号管理（租户维度）
type BatteryBmsModel struct{}

func normalizeBmsModelName(name string) (string, error) {
	v := strings.TrimSpace(name)
	if v == "" {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "型号名称不能为空"})
	}
	if utf8.RuneCountInString(v) > 100 {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "型号名称长度不能超过100个字符"})
	}
	return v, nil
}

func validateDeviceConfig(ctx context.Context, tenantID string, deviceConfigID *string) error {
	if deviceConfigID == nil {
		return nil
	}
	v := strings.TrimSpace(*deviceConfigID)
	if v == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "关联设备模板不能为空"})
	}

	var cnt int64
	if err := global.DB.WithContext(ctx).Table("device_configs").
		Where("id = ? AND tenant_id = ?", v, tenantID).
		Count(&cnt).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if cnt == 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "关联设备模板不存在"})
	}
	return nil
}

func getBmsModelRespByID(ctx context.Context, tenantID, id string) (*model.BatteryBmsModelResp, error) {
	type row struct {
		ID               string     `gorm:"column:id"`
		Name             string     `gorm:"column:name"`
		DeviceConfigID   *string    `gorm:"column:device_config_id"`
		DeviceConfigName *string    `gorm:"column:device_config_name"`
		VoltageRated     *float64   `gorm:"column:voltage_rated"`
		CapacityRated    *float64   `gorm:"column:capacity_rated"`
		CellCount        *int32     `gorm:"column:cell_count"`
		NominalPower     *float64   `gorm:"column:nominal_power"`
		WarrantyMonths   *int32     `gorm:"column:warranty_months"`
		Description      *string    `gorm:"column:description"`
		CreatedAt        *time.Time `gorm:"column:created_at"`
		UpdatedAt        *time.Time `gorm:"column:updated_at"`
	}

	var r row
	err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryModel+" AS bm").
		Select(`
			bm.id,
			bm.name,
			bm.device_config_id,
			dc.name AS device_config_name,
			bm.voltage_rated,
			bm.capacity_rated,
			bm.cell_count,
			bm.nominal_power,
			bm.warranty_months,
			bm.description,
			bm.created_at,
			bm.updated_at
		`).
		Joins("LEFT JOIN device_configs dc ON dc.id = bm.device_config_id").
		Where("bm.id = ? AND bm.tenant_id = ?", id, tenantID).
		Limit(1).
		Scan(&r).Error
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if r.ID == "" {
		return nil, errcode.New(404)
	}

	resp := &model.BatteryBmsModelResp{
		ID:               r.ID,
		Name:             r.Name,
		DeviceConfigID:   r.DeviceConfigID,
		DeviceConfigName: r.DeviceConfigName,
		VoltageRated:     r.VoltageRated,
		CapacityRated:    r.CapacityRated,
		CellCount:        r.CellCount,
		NominalPower:     r.NominalPower,
		WarrantyMonths:   r.WarrantyMonths,
		Description:      r.Description,
	}
	if r.CreatedAt != nil {
		resp.CreatedAt = r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05")
	}
	if r.UpdatedAt != nil {
		resp.UpdatedAt = r.UpdatedAt.In(time.Local).Format("2006-01-02 15:04:05")
	}
	return resp, nil
}

// CreateBatteryBmsModel 创建 BMS 型号
func (*BatteryBmsModel) CreateBatteryBmsModel(req model.BatteryBmsModelCreateReq, claims *utils.UserClaims) (*model.BatteryBmsModelResp, error) {
	ctx := context.Background()
	name, err := normalizeBmsModelName(req.Name)
	if err != nil {
		return nil, err
	}

	deviceConfigID := strings.TrimSpace(req.DeviceConfigID)
	if err := validateDeviceConfig(ctx, claims.TenantID, &deviceConfigID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	row := &model.BatteryModel{
		ID:             uuid.New(),
		Name:           name,
		DeviceConfigID: &deviceConfigID,
		VoltageRated:   req.VoltageRated,
		CapacityRated:  req.CapacityRated,
		CellCount:      req.CellCount,
		NominalPower:   req.NominalPower,
		WarrantyMonth:  req.WarrantyMonths,
		Description:    req.Description,
		TenantID:       claims.TenantID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	if err := global.DB.WithContext(ctx).Table(model.TableNameBatteryModel).Create(row).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return getBmsModelRespByID(ctx, claims.TenantID, row.ID)
}

// UpdateBatteryBmsModel 更新 BMS 型号
func (*BatteryBmsModel) UpdateBatteryBmsModel(id string, req model.BatteryBmsModelUpdateReq, claims *utils.UserClaims) (*model.BatteryBmsModelResp, error) {
	ctx := context.Background()

	var exists model.BatteryModel
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryModel).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		First(&exists).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(404)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		name, err := normalizeBmsModelName(*req.Name)
		if err != nil {
			return nil, err
		}
		updates["name"] = name
	}
	if req.DeviceConfigID != nil {
		v := strings.TrimSpace(*req.DeviceConfigID)
		if err := validateDeviceConfig(ctx, claims.TenantID, &v); err != nil {
			return nil, err
		}
		updates["device_config_id"] = v
	}
	if req.VoltageRated != nil {
		updates["voltage_rated"] = *req.VoltageRated
	}
	if req.CapacityRated != nil {
		updates["capacity_rated"] = *req.CapacityRated
	}
	if req.CellCount != nil {
		updates["cell_count"] = *req.CellCount
	}
	if req.NominalPower != nil {
		updates["nominal_power"] = *req.NominalPower
	}
	if req.WarrantyMonths != nil {
		updates["warranty_months"] = *req.WarrantyMonths
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if len(updates) == 0 {
		return getBmsModelRespByID(ctx, claims.TenantID, id)
	}

	updates["updated_at"] = time.Now().UTC()
	shouldRecalcWarranty := req.WarrantyMonths != nil &&
		*req.WarrantyMonths > 0 &&
		(exists.WarrantyMonth == nil || *exists.WarrantyMonth != *req.WarrantyMonths)
	var warrantyRecalcJobID string
	if err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(model.TableNameBatteryModel).
			Where("id = ? AND tenant_id = ?", id, claims.TenantID).
			Updates(updates).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if shouldRecalcWarranty {
			jobID, err := createBatteryWarrantyRecalcJobTx(
				ctx,
				tx,
				claims.TenantID,
				claims.ID,
				batteryWarrantyRecalcSourceModelChange,
				&id,
			)
			if err != nil {
				return err
			}
			warrantyRecalcJobID = jobID
		}
		return nil
	}); err != nil {
		return nil, err
	}

	resp, err := getBmsModelRespByID(ctx, claims.TenantID, id)
	if err != nil {
		return nil, err
	}
	if warrantyRecalcJobID != "" {
		resp.WarrantyRecalcJobID = &warrantyRecalcJobID
		startBatteryWarrantyRecalcJob(warrantyRecalcJobID)
	}
	return resp, nil
}

// DeleteBatteryBmsModel 删除 BMS 型号
func (*BatteryBmsModel) DeleteBatteryBmsModel(id string, claims *utils.UserClaims) error {
	ctx := context.Background()
	var exists model.BatteryModel
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryModel).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		First(&exists).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.New(404)
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryModel).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		Delete(&model.BatteryModel{}).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

// GetBatteryBmsModelByID 查询 BMS 型号详情
func (*BatteryBmsModel) GetBatteryBmsModelByID(id string, claims *utils.UserClaims) (*model.BatteryBmsModelResp, error) {
	return getBmsModelRespByID(context.Background(), claims.TenantID, id)
}

// GetBatteryBmsModelList 查询 BMS 型号列表
func (*BatteryBmsModel) GetBatteryBmsModelList(req model.BatteryBmsModelListReq, claims *utils.UserClaims) (*model.BatteryBmsModelListResp, error) {
	type row struct {
		ID               string     `gorm:"column:id"`
		Name             string     `gorm:"column:name"`
		DeviceConfigID   *string    `gorm:"column:device_config_id"`
		DeviceConfigName *string    `gorm:"column:device_config_name"`
		VoltageRated     *float64   `gorm:"column:voltage_rated"`
		CapacityRated    *float64   `gorm:"column:capacity_rated"`
		CellCount        *int32     `gorm:"column:cell_count"`
		NominalPower     *float64   `gorm:"column:nominal_power"`
		WarrantyMonths   *int32     `gorm:"column:warranty_months"`
		Description      *string    `gorm:"column:description"`
		CreatedAt        *time.Time `gorm:"column:created_at"`
		UpdatedAt        *time.Time `gorm:"column:updated_at"`
	}

	ctx := context.Background()
	q := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryModel+" AS bm").
		Select(`
			bm.id,
			bm.name,
			bm.device_config_id,
			dc.name AS device_config_name,
			bm.voltage_rated,
			bm.capacity_rated,
			bm.cell_count,
			bm.nominal_power,
			bm.warranty_months,
			bm.description,
			bm.created_at,
			bm.updated_at
		`).
		Joins("LEFT JOIN device_configs dc ON dc.id = bm.device_config_id").
		Where("bm.tenant_id = ?", claims.TenantID)

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		q = q.Where("bm.name ILIKE ?", "%"+strings.TrimSpace(*req.Name)+"%")
	}
	if req.DeviceConfigID != nil && strings.TrimSpace(*req.DeviceConfigID) != "" {
		q = q.Where("bm.device_config_id = ?", strings.TrimSpace(*req.DeviceConfigID))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	var rows []row
	if err := q.Order("bm.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.BatteryBmsModelResp, 0, len(rows))
	for _, r := range rows {
		item := model.BatteryBmsModelResp{
			ID:               r.ID,
			Name:             r.Name,
			DeviceConfigID:   r.DeviceConfigID,
			DeviceConfigName: r.DeviceConfigName,
			VoltageRated:     r.VoltageRated,
			CapacityRated:    r.CapacityRated,
			CellCount:        r.CellCount,
			NominalPower:     r.NominalPower,
			WarrantyMonths:   r.WarrantyMonths,
			Description:      r.Description,
		}
		if r.CreatedAt != nil {
			item.CreatedAt = r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05")
		}
		if r.UpdatedAt != nil {
			item.UpdatedAt = r.UpdatedAt.In(time.Local).Format("2006-01-02 15:04:05")
		}
		list = append(list, item)
	}

	return &model.BatteryBmsModelListResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
