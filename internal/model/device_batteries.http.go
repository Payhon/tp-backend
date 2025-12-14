package model

// BatteryListReq 电池列表查询请求（BMS：电池管理-电池列表）
type BatteryListReq struct {
	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=1,max=100"`

	// 设备编号（序列号）
	DeviceNumber *string `form:"device_number"`

	// 电池型号
	BatteryModelID *string `form:"battery_model_id"`

	// 在线状态：1-在线 0-离线
	IsOnline *int16 `form:"is_online" binding:"omitempty,oneof=0 1"`

	// 激活状态：ACTIVE/INACTIVE
	ActivationStatus *string `form:"activation_status" binding:"omitempty,oneof=ACTIVE INACTIVE"`

	// 经销商
	DealerID *string `form:"dealer_id"`

	// 出厂日期范围（YYYY-MM-DD）
	ProductionDateStart *string `form:"production_date_start"`
	ProductionDateEnd   *string `form:"production_date_end"`

	// 质保状态：IN-在保 OVER-过保
	WarrantyStatus *string `form:"warranty_status" binding:"omitempty,oneof=IN OVER"`
}

// BatteryListItemResp 电池列表项
type BatteryListItemResp struct {
	DeviceID     string  `json:"device_id"`
	DeviceNumber string  `json:"device_number"`
	DeviceName   *string `json:"device_name"`

	BatteryModelID   *string `json:"battery_model_id"`
	BatteryModelName *string `json:"battery_model_name"`

	ProductionDate     *string `json:"production_date"`
	WarrantyExpireDate *string `json:"warranty_expire_date"`

	DealerID   *string `json:"dealer_id"`
	DealerName *string `json:"dealer_name"`

	UserID    *string `json:"user_id"`
	UserName  *string `json:"user_name"`
	UserPhone *string `json:"user_phone"`

	ActivationDate   *string `json:"activation_date"`
	ActivationStatus *string `json:"activation_status"`

	IsOnline       int16    `json:"is_online"`
	Soc            *float64 `json:"soc"`
	Soh            *float64 `json:"soh"`
	CurrentVersion *string  `json:"current_version"`
	TransferStatus *string  `json:"transfer_status"`
}

// BatteryListResp 电池列表响应
type BatteryListResp struct {
	List     []BatteryListItemResp `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

// BatteryTemplateResp 电池导入模板（CSV 内容）
type BatteryTemplateResp struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// BatteryExportResp 电池导出（CSV 内容）
type BatteryExportResp struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// BatteryImportFailure 导入失败明细
type BatteryImportFailure struct {
	Row          int     `json:"row"`
	DeviceNumber *string `json:"device_number"`
	Message      string  `json:"message"`
}

// BatteryImportResp 导入结果
type BatteryImportResp struct {
	Total    int                    `json:"total"`
	Success  int                    `json:"success"`
	Failed   int                    `json:"failed"`
	Failures []BatteryImportFailure `json:"failures"`
}

// BatteryExportReq 电池导出请求（复用列表查询条件，但不分页）
type BatteryExportReq struct {
	// 设备编号（序列号）
	DeviceNumber *string `form:"device_number"`

	// 电池型号
	BatteryModelID *string `form:"battery_model_id"`

	// 在线状态：1-在线 0-离线
	IsOnline *int16 `form:"is_online" binding:"omitempty,oneof=0 1"`

	// 激活状态：ACTIVE/INACTIVE
	ActivationStatus *string `form:"activation_status" binding:"omitempty,oneof=ACTIVE INACTIVE"`

	// 经销商
	DealerID *string `form:"dealer_id"`

	// 出厂日期范围（YYYY-MM-DD）
	ProductionDateStart *string `form:"production_date_start"`
	ProductionDateEnd   *string `form:"production_date_end"`

	// 质保状态：IN-在保 OVER-过保
	WarrantyStatus *string `form:"warranty_status" binding:"omitempty,oneof=IN OVER"`
}

// BatteryBatchAssignDealerReq 批量分配经销商请求
type BatteryBatchAssignDealerReq struct {
	DeviceIDs []string `json:"device_ids" binding:"required,min=1"`
	DealerID  string   `json:"dealer_id" binding:"required"`
}

// BatteryBatchCommandReq 批量下发指令请求
type BatteryBatchCommandReq struct {
	DeviceIDs   []string `json:"device_ids" binding:"required,min=1"`
	CommandType string   `json:"command_type" binding:"required,max=64"` // 展示用
	Identify    string   `json:"identify" binding:"required,max=255"`
	Value       *string  `json:"value" binding:"omitempty,max=9999"` // JSON 字符串
}

type BatteryBatchCommandFailure struct {
	DeviceID     string `json:"device_id"`
	DeviceNumber string `json:"device_number"`
	Message      string `json:"message"`
}

type BatteryBatchCommandResp struct {
	Total    int                          `json:"total"`
	Success  int                          `json:"success"`
	Failed   int                          `json:"failed"`
	Failures []BatteryBatchCommandFailure `json:"failures"`
}

// BatteryBatchOtaPushReq 批量 OTA 推送
type BatteryBatchOtaPushReq struct {
	DeviceIDs          []string `json:"device_ids" binding:"required,min=1"`
	OTAUpgradePackageID string   `json:"ota_upgrade_package_id" binding:"required,max=36"`
	Name               *string  `json:"name" binding:"omitempty,max=200"` // 可选：任务名称
	Description        *string  `json:"description" binding:"omitempty,max=500"`
	Remark             *string  `json:"remark" binding:"omitempty,max=255"`
}

type BatteryBatchOtaPushFailure struct {
	DeviceID     string `json:"device_id"`
	DeviceNumber string `json:"device_number"`
	Message      string `json:"message"`
}

type BatteryBatchOtaPushResp struct {
	TaskID   string                   `json:"task_id"`
	Total    int                      `json:"total"`
	Accepted int                      `json:"accepted"`
	Rejected int                      `json:"rejected"`
	Failures []BatteryBatchOtaPushFailure `json:"failures"`
}
