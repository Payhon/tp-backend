package model

// DeviceBindReq APP端设备绑定请求
type DeviceBindReq struct {
	DeviceNumber string  `json:"device_number" binding:"required"`
	DeviceSecret *string `json:"device_secret"` // 设备密钥（可选，用于验证）
}

// DeviceUnbindReq APP端设备解绑请求
type DeviceUnbindReq struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// DeviceUserBindingListReq 设备绑定记录查询请求
type DeviceUserBindingListReq struct {
	Page         int     `form:"page" binding:"required,min=1"`
	PageSize     int     `form:"page_size" binding:"required,min=1,max=100"`
	UserID       *string `form:"user_id"`
	DeviceNumber *string `form:"device_number"`
}

// DeviceUserBindingResp 设备绑定记录响应
type DeviceUserBindingResp struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	UserName     *string `json:"user_name"`
	UserPhone    string  `json:"user_phone"`
	DeviceID     string  `json:"device_id"`
	DeviceNumber string  `json:"device_number"`
	DeviceName   string  `json:"device_name"`
	IsOwner      bool    `json:"is_owner"`
	BindingTime  string  `json:"binding_time"`
}

// DeviceUserBindingListResp 设备绑定记录列表响应
type DeviceUserBindingListResp struct {
	List     []DeviceUserBindingResp `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}
