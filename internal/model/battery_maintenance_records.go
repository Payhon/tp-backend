package model

import "time"

const TableNameBatteryMaintenanceRecord = "battery_maintenance_records"

// BatteryMaintenanceRecord 电池维保记录（手工录入）
type BatteryMaintenanceRecord struct {
	ID             string     `gorm:"column:id;primaryKey" json:"id"`
	TenantID       string     `gorm:"column:tenant_id;not null" json:"tenant_id"`
	DeviceID       string     `gorm:"column:device_id;not null" json:"device_id"`
	FaultType      string     `gorm:"column:fault_type;not null" json:"fault_type"`
	MaintainAt     time.Time  `gorm:"column:maintain_at;not null" json:"maintain_at"`
	Maintainer     string     `gorm:"column:maintainer;not null" json:"maintainer"`
	Solution       *string    `gorm:"column:solution" json:"solution"`
	Parts          *string    `gorm:"column:parts" json:"parts"` // jsonb string
	AffectWarranty bool       `gorm:"column:affect_warranty;not null" json:"affect_warranty"`
	Remark         *string    `gorm:"column:remark" json:"remark"`
	CreatedAt      *time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      *time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (*BatteryMaintenanceRecord) TableName() string {
	return TableNameBatteryMaintenanceRecord
}
