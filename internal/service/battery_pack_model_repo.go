package service

import (
	"context"
	"strings"

	"project/internal/model"
	global "project/pkg/global"

	"gorm.io/gorm"
)

func getPackBatteryModelByID(ctx context.Context, tenantID, id string) (*model.BatteryPackModel, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var row model.BatteryPackModel
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryPackModel).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func getPackBatteryModelByName(ctx context.Context, tenantID, name string) (*model.BatteryPackModel, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var row model.BatteryPackModel
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryPackModel).
		Where("tenant_id = ? AND name = ?", tenantID, name).
		Order("created_at DESC").
		Limit(1).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func getBmsBatteryModelByID(ctx context.Context, tenantID, id string) (*model.BatteryModel, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var row model.BatteryModel
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryModel).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func getBmsBatteryModelByName(ctx context.Context, tenantID, name string) (*model.BatteryModel, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var row model.BatteryModel
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryModel).
		Where("tenant_id = ? AND name = ?", tenantID, name).
		Order("created_at DESC").
		Limit(1).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
