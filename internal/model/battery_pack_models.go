package model

import "time"

const TableNameBatteryPackModel = "battery_models"

// BatteryPackModel 电池型号（PACK 厂家机构维度）
type BatteryPackModel struct {
	ID        string     `gorm:"column:id;primaryKey" json:"id"`
	SeqNo     int16      `gorm:"column:seq_no;not null" json:"seq_no"`
	Name      string     `gorm:"column:name;not null" json:"name"`
	OrgID     string     `gorm:"column:org_id;not null" json:"org_id"`
	TenantID  string     `gorm:"column:tenant_id;not null" json:"tenant_id"`
	CreatedAt *time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (*BatteryPackModel) TableName() string {
	return TableNameBatteryPackModel
}
