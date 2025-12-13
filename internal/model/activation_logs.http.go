package model

import "time"

// ActivationLogListReq 激活日志查询
type ActivationLogListReq struct {
	PageReq
	DeviceNumber *string    `form:"device_number"`
	UserPhone    *string    `form:"user_phone"`
	StartTime    *time.Time `form:"start_time"`
	EndTime      *time.Time `form:"end_time"`
	Method       *string    `form:"method" binding:"omitempty,oneof=APP WEB"` // APP扫码/WEB手动
}

// ActivationLogResp 激活日志行
type ActivationLogResp struct {
	DeviceNumber    string  `json:"device_number"`
	BatteryModel    *string `json:"battery_model"`
	UserPhone       string  `json:"user_phone"`
	ActivationTime  string  `json:"activation_time"`
	ActivationWay   string  `json:"activation_way"`   // APP扫码/WEB手动
	BindingTerminal string  `json:"binding_terminal"` // APP/小程序/WEB
	IP              string  `json:"ip"`
}

// ActivationLogListResp 激活日志列表
type ActivationLogListResp struct {
	List     []ActivationLogResp `json:"list"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

