package service

import (
	"context"
	"time"

	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

type BatteryModel struct{}

// CreateBatteryModel 创建电池型号
func (*BatteryModel) CreateBatteryModel(req model.BatteryModelCreateReq, claims *utils.UserClaims) (*model.BatteryModel, error) {
	t := time.Now().UTC()

	// 校验关联设备模板
	if _, err := query.DeviceConfig.WithContext(context.Background()).Where(
		query.DeviceConfig.ID.Eq(req.DeviceConfigID),
		query.DeviceConfig.TenantID.Eq(claims.TenantID),
	).First(); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "关联设备模板不存在"})
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	batteryModel := &model.BatteryModel{
		ID:             uuid.New(),
		Name:           req.Name,
		DeviceConfigID: StringPtr(req.DeviceConfigID),
		VoltageRated:   req.VoltageRated,
		CapacityRated:  req.CapacityRated,
		CellCount:      req.CellCount,
		NominalPower:   req.NominalPower,
		WarrantyMonth:  req.WarrantyMonths,
		Description:    req.Description,
		TenantID:       claims.TenantID,
		CreatedAt:      &t,
		UpdatedAt:      &t,
	}

	// 创建电池型号
	if err := query.BatteryModel.Create(batteryModel); err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return batteryModel, nil
}

// UpdateBatteryModel 更新电池型号
func (*BatteryModel) UpdateBatteryModel(id string, req model.BatteryModelUpdateReq, claims *utils.UserClaims) (*model.BatteryModel, error) {
	t := time.Now().UTC()

	// 查询电池型号是否存在
	batteryModel, err := query.BatteryModel.Where(
		query.BatteryModel.ID.Eq(id),
		query.BatteryModel.TenantID.Eq(claims.TenantID),
	).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(404)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.DeviceConfigID != nil {
		// 校验关联设备模板
		if _, err := query.DeviceConfig.WithContext(context.Background()).Where(
			query.DeviceConfig.ID.Eq(*req.DeviceConfigID),
			query.DeviceConfig.TenantID.Eq(claims.TenantID),
		).First(); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "关联设备模板不存在"})
			}
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		updates["device_config_id"] = *req.DeviceConfigID
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
	updates["updated_at"] = t

	// 执行更新
	if _, err := query.BatteryModel.Where(query.BatteryModel.ID.Eq(id)).Updates(updates); err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 重新查询返回
	batteryModel, err = query.BatteryModel.Where(query.BatteryModel.ID.Eq(id)).First()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return batteryModel, nil
}

// DeleteBatteryModel 删除电池型号
func (*BatteryModel) DeleteBatteryModel(id string, claims *utils.UserClaims) error {
	// 检查电池型号是否存在
	_, err := query.BatteryModel.Where(
		query.BatteryModel.ID.Eq(id),
		query.BatteryModel.TenantID.Eq(claims.TenantID),
	).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.New(404)
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 检查是否有关联设备
	deviceCount, err := query.DeviceBattery.Where(query.DeviceBattery.BatteryModelID.Eq(id)).Count()
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	if deviceCount > 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "battery model has devices, cannot delete",
		})
	}

	// 删除电池型号
	if _, err := query.BatteryModel.Where(query.BatteryModel.ID.Eq(id)).Delete(); err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return nil
}

// GetBatteryModelByID 获取电池型号详情
func (*BatteryModel) GetBatteryModelByID(id string, claims *utils.UserClaims) (*model.BatteryModelResp, error) {
	batteryModel, err := query.BatteryModel.Where(
		query.BatteryModel.ID.Eq(id),
		query.BatteryModel.TenantID.Eq(claims.TenantID),
	).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(404)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 统计设备数量
	deviceCount, _ := query.DeviceBattery.Where(query.DeviceBattery.BatteryModelID.Eq(id)).Count()

	// 关联设备模板名称
	var deviceConfigName *string
	if batteryModel.DeviceConfigID != nil && *batteryModel.DeviceConfigID != "" {
		var name string
		if err := global.DB.Table("device_configs").
			Select("name").
			Where("id = ? AND tenant_id = ?", *batteryModel.DeviceConfigID, claims.TenantID).
			Scan(&name).Error; err == nil && name != "" {
			deviceConfigName = &name
		}
	}

	createdAt := ""
	if batteryModel.CreatedAt != nil {
		createdAt = batteryModel.CreatedAt.Format("2006-01-02 15:04:05")
	}

	resp := &model.BatteryModelResp{
		ID:               batteryModel.ID,
		Name:             batteryModel.Name,
		DeviceConfigID:   batteryModel.DeviceConfigID,
		DeviceConfigName: deviceConfigName,
		VoltageRated:     batteryModel.VoltageRated,
		CapacityRated:    batteryModel.CapacityRated,
		CellCount:        batteryModel.CellCount,
		NominalPower:     batteryModel.NominalPower,
		WarrantyMonths:   batteryModel.WarrantyMonth,
		Description:      batteryModel.Description,
		DeviceCount:      deviceCount,
		CreatedAt:        createdAt,
	}

	return resp, nil
}

// GetBatteryModelList 获取电池型号列表
func (*BatteryModel) GetBatteryModelList(req model.BatteryModelListReq, claims *utils.UserClaims) (*model.BatteryModelListResp, error) {
	q := query.BatteryModel
	queryBuilder := q.Where(q.TenantID.Eq(claims.TenantID))

	// 条件筛选
	if req.Name != nil && *req.Name != "" {
		queryBuilder = queryBuilder.Where(q.Name.Like("%" + *req.Name + "%"))
	}

	// 统计总数
	total, err := queryBuilder.Count()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	batteryModels, err := queryBuilder.Offset(offset).Limit(req.PageSize).Order(q.CreatedAt.Desc()).Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 构建响应
	// 批量加载 device_config 名称
	deviceConfigIDSet := make(map[string]struct{})
	for _, bm := range batteryModels {
		if bm.DeviceConfigID != nil && *bm.DeviceConfigID != "" {
			deviceConfigIDSet[*bm.DeviceConfigID] = struct{}{}
		}
	}
	deviceConfigNameMap := make(map[string]string)
	if len(deviceConfigIDSet) > 0 {
		ids := make([]string, 0, len(deviceConfigIDSet))
		for id := range deviceConfigIDSet {
			ids = append(ids, id)
		}
		var rows []struct {
			ID   string `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		_ = global.DB.Table("device_configs").
			Select("id, name").
			Where("tenant_id = ? AND id IN ?", claims.TenantID, ids).
			Scan(&rows).Error
		for _, r := range rows {
			deviceConfigNameMap[r.ID] = r.Name
		}
	}

	list := make([]model.BatteryModelResp, 0, len(batteryModels))
	for _, bm := range batteryModels {
		// 统计设备数量
		deviceCount, _ := query.DeviceBattery.Where(query.DeviceBattery.BatteryModelID.Eq(bm.ID)).Count()

		var deviceConfigName *string
		if bm.DeviceConfigID != nil && *bm.DeviceConfigID != "" {
			if name, ok := deviceConfigNameMap[*bm.DeviceConfigID]; ok && name != "" {
				deviceConfigName = &name
			}
		}

		createdAt := ""
		if bm.CreatedAt != nil {
			createdAt = bm.CreatedAt.Format("2006-01-02 15:04:05")
		}

		list = append(list, model.BatteryModelResp{
			ID:               bm.ID,
			Name:             bm.Name,
			DeviceConfigID:   bm.DeviceConfigID,
			DeviceConfigName: deviceConfigName,
			VoltageRated:     bm.VoltageRated,
			CapacityRated:    bm.CapacityRated,
			CellCount:        bm.CellCount,
			NominalPower:     bm.NominalPower,
			WarrantyMonths:   bm.WarrantyMonth,
			Description:      bm.Description,
			DeviceCount:      deviceCount,
			CreatedAt:        createdAt,
		})
	}

	return &model.BatteryModelListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
