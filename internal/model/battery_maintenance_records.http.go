package model

import "time"

// BatteryMaintenanceCreateReq 新增电池维保记录（手动）
type BatteryMaintenanceCreateReq struct {
	DeviceNumber   string    `json:"device_number" binding:"required"`
	FaultType      string    `json:"fault_type" binding:"required"`
	MaintainAt     time.Time `json:"maintain_at" binding:"required"`
	Maintainer     string    `json:"maintainer" binding:"required"`
	Solution       *string   `json:"solution"`
	Parts          []string  `json:"parts"` // 配件更换清单
	AffectWarranty bool      `json:"affect_warranty"`
	Remark         *string   `json:"remark"`
}

// BatteryMaintenanceListReq 电池维保记录列表查询
type BatteryMaintenanceListReq struct {
	PageReq
	DeviceNumber *string    `form:"device_number"`
	FaultType    *string    `form:"fault_type"`
	StartTime    *time.Time `form:"start_time"`
	EndTime      *time.Time `form:"end_time"`
}

// BatteryMaintenanceItemResp 电池维保记录行
type BatteryMaintenanceItemResp struct {
	ID             string   `json:"id"`
	DeviceID       string   `json:"device_id"`
	DeviceNumber   string   `json:"device_number"`
	BatteryModel   *string  `json:"battery_model"`
	FaultType      string   `json:"fault_type"`
	MaintainAt     string   `json:"maintain_at"`
	Maintainer     string   `json:"maintainer"`
	Solution       *string  `json:"solution"`
	Parts          []string `json:"parts"`
	AffectWarranty bool     `json:"affect_warranty"`
	Remark         *string  `json:"remark"`
	CreatedAt      string   `json:"created_at"`
}

// BatteryMaintenanceListResp 列表响应
type BatteryMaintenanceListResp struct {
	List     []BatteryMaintenanceItemResp `json:"list"`
	Total    int64                        `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}
