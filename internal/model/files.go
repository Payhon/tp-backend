package model

import (
	"time"

	"gorm.io/datatypes"
)

const TableNameFile = "files"

// File mapped from table <files>
type File struct {
	ID              string         `gorm:"column:id;primaryKey" json:"id"`
	TenantID        string         `gorm:"column:tenant_id;not null" json:"tenant_id"`
	FileName        string         `gorm:"column:file_name;not null" json:"file_name"`
	OriginalFileName *string       `gorm:"column:original_file_name" json:"original_file_name,omitempty"`
	FileSize        int64          `gorm:"column:file_size;not null" json:"file_size"`
	StorageLocation string         `gorm:"column:storage_location;not null" json:"storage_location"` // local/aliyun/qiniu
	BizType         string         `gorm:"column:biz_type;not null" json:"biz_type"`
	MimeType        *string        `gorm:"column:mime_type" json:"mime_type,omitempty"`
	FileExt         *string        `gorm:"column:file_ext" json:"file_ext,omitempty"`
	MD5             *string        `gorm:"column:md5" json:"md5,omitempty"`
	SHA256          *string        `gorm:"column:sha256" json:"sha256,omitempty"`
	FilePath        string         `gorm:"column:file_path;not null" json:"file_path"`
	FullURL         string         `gorm:"column:full_url;not null" json:"full_url"`
	UploadedAt      time.Time      `gorm:"column:uploaded_at;not null" json:"uploaded_at"`
	UploadedBy      *string        `gorm:"column:uploaded_by" json:"uploaded_by,omitempty"`
	Meta            datatypes.JSON `gorm:"column:meta;not null" json:"meta"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null" json:"updated_at"`
	Remark          *string        `gorm:"column:remark" json:"remark,omitempty"`
}

func (*File) TableName() string {
	return TableNameFile
}

