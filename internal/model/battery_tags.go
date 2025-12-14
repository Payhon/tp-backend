package model

import "time"

const TableNameBatteryTag = "battery_tags"

// BatteryTag 电池标签
type BatteryTag struct {
	ID        string    `gorm:"column:id;primaryKey" json:"id"`
	TenantID  string    `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	Color     *string   `gorm:"column:color" json:"color"`
	Scene     *string   `gorm:"column:scene" json:"scene"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (*BatteryTag) TableName() string {
	return TableNameBatteryTag
}

