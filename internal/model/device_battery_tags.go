package model

import "time"

const TableNameDeviceBatteryTag = "device_battery_tags"

// DeviceBatteryTag 设备-电池标签关联
type DeviceBatteryTag struct {
	ID        string    `gorm:"column:id;primaryKey" json:"id"`
	TenantID  string    `gorm:"column:tenant_id;not null" json:"tenant_id"`
	DeviceID  string    `gorm:"column:device_id;not null" json:"device_id"`
	TagID     string    `gorm:"column:tag_id;not null" json:"tag_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (*DeviceBatteryTag) TableName() string {
	return TableNameDeviceBatteryTag
}

