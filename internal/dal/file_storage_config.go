package dal

import (
	"context"

	model "project/internal/model"
	global "project/pkg/global"

	"gorm.io/gorm"
)

func GetFileStorageConfig(ctx context.Context, id string) (*model.FileStorageConfig, error) {
	var cfg model.FileStorageConfig
	err := global.DB.WithContext(ctx).First(&cfg, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func UpsertFileStorageConfig(ctx context.Context, cfg *model.FileStorageConfig) error {
	return global.DB.WithContext(ctx).Save(cfg).Error
}

