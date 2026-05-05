package model

// BatteryCreateReq BMS：添加单个电池（设备电池扩展信息）
// item_uuid 需要对应 devices.device_number
type BatteryCreateReq struct {
	ItemUUID           string  `json:"item_uuid" binding:"required,max=64"`
	BatchNumber        string  `json:"batch_number" binding:"required,max=100"`
	ProductSpec        string  `json:"product_spec" binding:"required,max=32"`
	OrderNumber        string  `json:"order_number" binding:"required,max=32"`
	BmsCommType        int     `json:"bms_comm_type" binding:"required,oneof=1 2 3"`
	BleMac             *string `json:"ble_mac" binding:"omitempty,max=32"`
	CommChipID         *string `json:"comm_chip_id" binding:"omitempty,max=64"`
	BatteryModelID     *string `json:"battery_model_id" binding:"omitempty,max=36"`
	BatteryModelName   *string `json:"battery_model_name" binding:"omitempty,max=100"`
	PackFactoryName    *string `json:"pack_factory_name" binding:"omitempty,max=100"`
	ProductionDate     *string `json:"production_date" binding:"omitempty"`      // YYYY-MM-DD
	WarrantyExpireDate *string `json:"warranty_expire_date" binding:"omitempty"` // YYYY-MM-DD
	Remark             *string `json:"remark" binding:"omitempty,max=255"`
}

type BatteryCreateResp struct {
	DeviceID           string  `json:"device_id"`
	DeviceNumber       string  `json:"device_number"`
	BatteryModelID     *string `json:"battery_model_id"`
	BatteryModelName   *string `json:"battery_model_name"`
	ItemUUID           *string `json:"item_uuid"`
	BatchNumber        *string `json:"batch_number"`
	ProductSpec        *string `json:"product_spec"`
	OrderNumber        *string `json:"order_number"`
	BmsCommType        *int    `json:"bms_comm_type"`
	BleMac             *string `json:"ble_mac"`
	CommChipID         *string `json:"comm_chip_id"`
	ProductionDate     *string `json:"production_date"`
	WarrantyExpireDate *string `json:"warranty_expire_date"`
}

// BatteryImportJobCreateResp 创建导入任务响应
type BatteryImportJobCreateResp struct {
	JobID string `json:"job_id"`
}

// BatteryImportJobStatusResp 导入任务状态
type BatteryImportJobStatusResp struct {
	JobID         string  `json:"job_id"`
	Status        string  `json:"status"`
	TotalRows     int     `json:"total_rows"`
	ProcessedRows int     `json:"processed_rows"`
	SuccessRows   int     `json:"success_rows"`
	FailedRows    int     `json:"failed_rows"`
	ErrorMessage  *string `json:"error_message"`
	StartedAt     *string `json:"started_at"`
	FinishedAt    *string `json:"finished_at"`
	CreatedAt     string  `json:"created_at"`
}

type BatteryImportJobLogItem struct {
	ID           int64   `json:"id"`
	RowNumber    *int    `json:"row_number"`
	Level        string  `json:"level"`
	DeviceNumber *string `json:"device_number"`
	Message      string  `json:"message"`
	CreatedAt    string  `json:"created_at"`
}

type BatteryImportJobLogListResp struct {
	List        []BatteryImportJobLogItem `json:"list"`
	NextAfterID int64                     `json:"next_after_id"`
}

// BatteryOperationLogListReq 运营日志查询
type BatteryOperationLogListReq struct {
	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=1,max=100"`

	// 设备ID（电池详情按设备精确查询）
	DeviceID *string `form:"device_id"`

	// 电池编号（支持模糊查询）
	DeviceNumber *string `form:"device_number"`

	// 操作类型
	OperationType *string `form:"operation_type"`

	StartTime *string `form:"start_time"` // RFC3339 or YYYY-MM-DDTHH:mm:ssZ
	EndTime   *string `form:"end_time"`
}

type BatteryOperationLogItemResp struct {
	ID            int64   `json:"id"`
	OccurredAt    string  `json:"occurred_at"`
	DeviceID      string  `json:"device_id"`
	DeviceNumber  string  `json:"device_number"`
	OperationType string  `json:"operation_type"`
	OperatorID    *string `json:"operator_id"`
	OperatorName  *string `json:"operator_name"`
	Description   *string `json:"description"`
}

type BatteryOperationLogListResp struct {
	List     []BatteryOperationLogItemResp `json:"list"`
	Total    int64                         `json:"total"`
	Page     int                           `json:"page"`
	PageSize int                           `json:"page_size"`
}

// BatteryFactoryOutReq 电池出厂（厂家 -> PACK/经销商）
type BatteryFactoryOutReq struct {
	DeviceID string  `json:"device_id" binding:"required"`
	ToOrgID  string  `json:"to_org_id" binding:"required"`
	Remark   *string `json:"remark"`
}

type BatteryBatchFactoryOutReq struct {
	DeviceIDs []string `json:"device_ids" binding:"required,min=1"`
	ToOrgID   string   `json:"to_org_id" binding:"required"`
	Remark    *string  `json:"remark"`
}

type BatteryBatchFactoryOutFailure struct {
	DeviceID     string `json:"device_id"`
	DeviceNumber string `json:"device_number"`
	Message      string `json:"message"`
}

type BatteryBatchFactoryOutResp struct {
	Total    int                             `json:"total"`
	Success  int                             `json:"success"`
	Failed   int                             `json:"failed"`
	Failures []BatteryBatchFactoryOutFailure `json:"failures"`
}

// BatteryTransferReq 电池调拨（组织转移）
type BatteryTransferReq struct {
	DeviceID string  `json:"device_id" binding:"required"`
	ToOrgID  string  `json:"to_org_id" binding:"required"`
	Remark   *string `json:"remark"`
}

// BatteryFactoryRestoreReq 电池恢复出厂（退回厂家库存）
type BatteryFactoryRestoreReq struct {
	DeviceID string  `json:"device_id" binding:"required"`
	Remark   *string `json:"remark"`
}

type BatteryRollbackPreviewReq struct {
	DeviceID string `form:"device_id" binding:"required"`
}

type BatteryRollbackPreviewResp struct {
	DeviceID          string  `json:"device_id"`
	DeviceNumber      string  `json:"device_number"`
	CurrentOrgID      *string `json:"current_org_id,omitempty"`
	CurrentOrgName    *string `json:"current_org_name,omitempty"`
	RollbackToOrgID   *string `json:"rollback_to_org_id,omitempty"`
	RollbackToOrgName *string `json:"rollback_to_org_name,omitempty"`
	CanRollback       bool    `json:"can_rollback"`
	Reason            *string `json:"reason,omitempty"`
}

type BatteryRollbackReq struct {
	DeviceID string  `json:"device_id" binding:"required"`
	Remark   *string `json:"remark"`
}

// BatteryActivateReq 电池激活（绑定到 APP 用户）
type BatteryActivateReq struct {
	DeviceID string  `json:"device_id" binding:"required"`
	UserID   string  `json:"user_id" binding:"required"`
	Remark   *string `json:"remark"`
}
