package model

import "time"

const TableNameVersionUpdateRecord = "version_update_records"

const (
	VersionUpdateProjectMobile        = "MOBILE"
	VersionUpdateProjectCloudFrontend = "CLOUD_FRONTEND"
	VersionUpdateProjectCloudBackend  = "CLOUD_BACKEND"
)

const (
	VersionUpdateSourceManual = "manual"
	VersionUpdateSourceAppDoc = "app_update_doc"
	VersionUpdateSourceGitLog = "git_log"
)

// VersionUpdateRecord 平台版本更新记录
type VersionUpdateRecord struct {
	ID            string    `gorm:"column:id;primaryKey" json:"id"`
	Project       string    `gorm:"column:project;not null" json:"project"`
	VersionNo     string    `gorm:"column:version_no;not null" json:"version_no"`
	ReleaseDate   time.Time `gorm:"column:release_date;not null" json:"release_date"`
	UpdateContent string    `gorm:"column:update_content;not null" json:"update_content"`
	Source        string    `gorm:"column:source;not null" json:"source"`
	SourceRef     *string   `gorm:"column:source_ref" json:"source_ref"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (*VersionUpdateRecord) TableName() string {
	return TableNameVersionUpdateRecord
}
