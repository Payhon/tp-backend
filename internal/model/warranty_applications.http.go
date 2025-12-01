package model

// WarrantyApplicationCreateReq 创建维保申请请求
type WarrantyApplicationCreateReq struct {
	DeviceID    string   `json:"device_id" binding:"required"`
	Type        string   `json:"type" binding:"required,oneof=REPAIR RETURN EXCHANGE"`
	Description *string  `json:"description"`
	Images      []string `json:"images"` // 图片URL数组
}

// WarrantyApplicationUpdateReq 更新维保申请请求
type WarrantyApplicationUpdateReq struct {
	Status     *string                `json:"status" binding:"omitempty,oneof=APPROVED REJECTED PROCESSING COMPLETED"`
	ResultInfo map[string]interface{} `json:"result_info"` // 处理结果信息
	HandlerID  *string                `json:"handler_id"`
}

// WarrantyApplicationListReq 维保申请列表查询请求
type WarrantyApplicationListReq struct {
	Page         int     `form:"page" binding:"required,min=1"`
	PageSize     int     `form:"page_size" binding:"required,min=1,max=100"`
	DeviceNumber *string `form:"device_number"`
	UserID       *string `form:"user_id"`
	Type         *string `form:"type"`
	Status       *string `form:"status"`
	StartTime    *string `form:"start_time"`
	EndTime      *string `form:"end_time"`
}

// WarrantyApplicationResp 维保申请响应
type WarrantyApplicationResp struct {
	ID             string                 `json:"id"`
	DeviceID       string                 `json:"device_id"`
	DeviceNumber   string                 `json:"device_number"`
	DeviceName     string                 `json:"device_name"`
	UserID         string                 `json:"user_id"`
	UserName       *string                `json:"user_name"`
	UserPhone      string                 `json:"user_phone"`
	Type           string                 `json:"type"`
	Description    *string                `json:"description"`
	Images         []string               `json:"images"`
	Status         string                 `json:"status"`
	ResultInfo     map[string]interface{} `json:"result_info"`
	HandlerID      *string                `json:"handler_id"`
	HandlerName    *string                `json:"handler_name"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
}

// WarrantyApplicationListResp 维保申请列表响应
type WarrantyApplicationListResp struct {
	List     []WarrantyApplicationResp `json:"list"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}
