package model

// BatteryBmsModelCreateReq 创建 BMS 型号请求
type BatteryBmsModelCreateReq struct {
	Name           string   `json:"name" binding:"required,max=100"`
	DeviceConfigID string   `json:"device_config_id" binding:"required,max=36"`
	VoltageRated   *float64 `json:"voltage_rated"`
	CapacityRated  *float64 `json:"capacity_rated"`
	CellCount      *int32   `json:"cell_count"`
	NominalPower   *float64 `json:"nominal_power"`
	WarrantyMonths *int32   `json:"warranty_months"`
	Description    *string  `json:"description"`
}

// BatteryBmsModelUpdateReq 更新 BMS 型号请求
type BatteryBmsModelUpdateReq struct {
	Name           *string  `json:"name" binding:"omitempty,max=100"`
	DeviceConfigID *string  `json:"device_config_id" binding:"omitempty,max=36"`
	VoltageRated   *float64 `json:"voltage_rated"`
	CapacityRated  *float64 `json:"capacity_rated"`
	CellCount      *int32   `json:"cell_count"`
	NominalPower   *float64 `json:"nominal_power"`
	WarrantyMonths *int32   `json:"warranty_months"`
	Description    *string  `json:"description"`
}

// BatteryBmsModelListReq BMS 型号列表查询
type BatteryBmsModelListReq struct {
	Page           int     `form:"page"`
	PageSize       int     `form:"page_size"`
	Name           *string `form:"name"`
	DeviceConfigID *string `form:"device_config_id"`
}

// BatteryBmsModelResp BMS 型号响应
type BatteryBmsModelResp struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	DeviceConfigID      *string  `json:"device_config_id"`
	DeviceConfigName    *string  `json:"device_config_name"`
	VoltageRated        *float64 `json:"voltage_rated"`
	CapacityRated       *float64 `json:"capacity_rated"`
	CellCount           *int32   `json:"cell_count"`
	NominalPower        *float64 `json:"nominal_power"`
	WarrantyMonths      *int32   `json:"warranty_months"`
	Description         *string  `json:"description"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
	WarrantyRecalcJobID *string  `json:"warranty_recalc_job_id,omitempty"`
}

// BatteryBmsModelListResp BMS 型号列表响应
type BatteryBmsModelListResp struct {
	List     []BatteryBmsModelResp `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}
