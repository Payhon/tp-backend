package model

import (
	"time"

	"gorm.io/datatypes"
)

const TableNameFileStorageConfig = "file_storage_config"

// FileStorageConfig mapped from table <file_storage_config>
type FileStorageConfig struct {
	ID          string         `gorm:"column:id;primaryKey" json:"id"`
	StorageType string         `gorm:"column:storage_type;not null" json:"storage_type"` // local/cloud
	Provider    *string        `gorm:"column:provider" json:"provider,omitempty"`        // aliyun/qiniu
	Config      datatypes.JSON `gorm:"column:config;not null" json:"config"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null" json:"updated_at"`
	Remark      *string        `gorm:"column:remark" json:"remark,omitempty"`
}

func (*FileStorageConfig) TableName() string {
	return TableNameFileStorageConfig
}

