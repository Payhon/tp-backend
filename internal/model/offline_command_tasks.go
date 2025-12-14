package model

import "time"

const TableNameOfflineCommandTask = "offline_command_tasks"

// OfflineCommandTask 离线指令任务：设备离线时存储，设备上线后自动下发
type OfflineCommandTask struct {
	ID           string     `gorm:"column:id;primaryKey" json:"id"`
	TenantID     string     `gorm:"column:tenant_id;not null" json:"tenant_id"`
	DeviceID     string     `gorm:"column:device_id;not null" json:"device_id"`
	DeviceNumber string     `gorm:"column:device_number;not null" json:"device_number"`
	CommandType  string     `gorm:"column:command_type;not null" json:"command_type"`
	Identify     string     `gorm:"column:identify;not null" json:"identify"`
	Payload      *string    `gorm:"column:payload" json:"payload"`
	CreatedBy    *string    `gorm:"column:created_by" json:"created_by"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
	Status       string     `gorm:"column:status;not null" json:"status"`
	DispatchedAt *time.Time `gorm:"column:dispatched_at" json:"dispatched_at"`
	ExecutedAt   *time.Time `gorm:"column:executed_at" json:"executed_at"`
	MessageID    *string    `gorm:"column:message_id" json:"message_id"`
	ErrorMessage *string    `gorm:"column:error_message" json:"error_message"`
}

func (*OfflineCommandTask) TableName() string {
	return TableNameOfflineCommandTask
}

