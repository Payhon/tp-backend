package model

// OfflineCommandCreateReq 创建离线指令
type OfflineCommandCreateReq struct {
	DeviceID    string  `json:"device_id" binding:"required,max=36"`
	CommandType string  `json:"command_type" binding:"required,max=64"` // 展示用名称
	Identify    string  `json:"identify" binding:"required,max=255"`
	Value       *string `json:"value" binding:"omitempty,max=9999"` // JSON 字符串
}

// OfflineCommandListReq 离线指令列表查询
type OfflineCommandListReq struct {
	PageReq
	DeviceNumber *string `form:"device_number"`
	CommandType  *string `form:"command_type"`
	Status       *string `form:"status" binding:"omitempty,oneof=PENDING SENT SUCCESS FAILED CANCELLED"`
}

type OfflineCommandListItemResp struct {
	ID           string  `json:"id"`
	DeviceID     string  `json:"device_id"`
	DeviceNumber string  `json:"device_number"`
	CommandType  string  `json:"command_type"`
	Identify     string  `json:"identify"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
	OperatorName *string `json:"operator_name"`
	DispatchedAt *string `json:"dispatched_at"`
	ExecutedAt   *string `json:"executed_at"`
	ErrorMessage *string `json:"error_message"`
}

type OfflineCommandListResp struct {
	List     []OfflineCommandListItemResp `json:"list"`
	Total    int64                        `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}

type OfflineCommandDetailResp struct {
	ID           string  `json:"id"`
	DeviceID     string  `json:"device_id"`
	DeviceNumber string  `json:"device_number"`
	CommandType  string  `json:"command_type"`
	Identify     string  `json:"identify"`
	Payload      *string `json:"payload"`
	Status       string  `json:"status"`
	MessageID    *string `json:"message_id"`
	CreatedAt    string  `json:"created_at"`
	OperatorName *string `json:"operator_name"`
	DispatchedAt *string `json:"dispatched_at"`
	ExecutedAt   *string `json:"executed_at"`
	ErrorMessage *string `json:"error_message"`
	// 来自 command_set_logs
	CommandLogStatus   *string `json:"command_log_status"`
	CommandLogRspData  *string `json:"command_log_rsp_data"`
	CommandLogErrorMsg *string `json:"command_log_error_message"`
}

