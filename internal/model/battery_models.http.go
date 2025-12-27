package model

// BatteryModelCreateReq 创建电池型号请求
type BatteryModelCreateReq struct {
	Name           string   `json:"name" binding:"required"`
	VoltageRated   *float64 `json:"voltage_rated"`
	CapacityRated  *float64 `json:"capacity_rated"`
	CellCount      *int32   `json:"cell_count"`
	NominalPower   *float64 `json:"nominal_power"`
	WarrantyMonths *int32   `json:"warranty_months"`
	Description    *string  `json:"description"`
}

// BatteryModelUpdateReq 更新电池型号请求
type BatteryModelUpdateReq struct {
	Name           *string  `json:"name"`
	VoltageRated   *float64 `json:"voltage_rated"`
	CapacityRated  *float64 `json:"capacity_rated"`
	CellCount      *int32   `json:"cell_count"`
	NominalPower   *float64 `json:"nominal_power"`
	WarrantyMonths *int32   `json:"warranty_months"`
	Description    *string  `json:"description"`
}

// BatteryModelListReq 电池型号列表查询请求
type BatteryModelListReq struct {
	Page     int     `form:"page" binding:"required,min=1"`
	PageSize int     `form:"page_size" binding:"required,min=1,max=1000"`
	Name     *string `form:"name"`
}

// BatteryModelResp 电池型号响应
type BatteryModelResp struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	VoltageRated   *float64 `json:"voltage_rated"`
	CapacityRated  *float64 `json:"capacity_rated"`
	CellCount      *int32   `json:"cell_count"`
	NominalPower   *float64 `json:"nominal_power"`
	WarrantyMonths *int32   `json:"warranty_months"`
	Description    *string  `json:"description"`
	DeviceCount    int64    `json:"device_count"` // 关联设备数
	CreatedAt      string   `json:"created_at"`
}

// BatteryModelListResp 电池型号列表响应
type BatteryModelListResp struct {
	List     []BatteryModelResp `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}
