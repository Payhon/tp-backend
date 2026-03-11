package model

import "time"

const TableNameBatteryCellBrand = "battery_cell_brands"

// BatteryCellBrand 电芯品牌（租户全局）
type BatteryCellBrand struct {
	ID        string    `gorm:"column:id;primaryKey" json:"id"`
	TenantID  string    `gorm:"column:tenant_id;not null" json:"tenant_id"`
	SeqNo     int16     `gorm:"column:seq_no;not null" json:"seq_no"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (*BatteryCellBrand) TableName() string {
	return TableNameBatteryCellBrand
}
