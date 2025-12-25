package model

// EndUserListReq 终端用户列表（BMS穿透）
type EndUserListReq struct {
	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=1,max=100"`

	// 厂家侧可按组织筛选；组织用户侧由中间件自动限定子树
	OwnerOrgID *string `form:"owner_org_id"`

	// 已废弃，保留兼容
	DealerID *string `form:"dealer_id"`

	Phone        *string `form:"phone"`
	DeviceNumber *string `form:"device_number"`
}

// EndUserListItemResp 终端用户列表项
type EndUserListItemResp struct {
	UserID    string  `json:"user_id"`
	UserName  *string `json:"user_name"`
	UserPhone string  `json:"user_phone"`

	DeviceCount int64   `json:"device_count"`
	LastBindAt  *string `json:"last_bind_at"`

	// 组织相关
	OwnerOrgID   *string `json:"owner_org_id"`
	OwnerOrgName *string `json:"owner_org_name"`

	// 已废弃，保留兼容
	DealerID   *string `json:"dealer_id"`
	DealerName *string `json:"dealer_name"`
}

// EndUserListResp 终端用户列表响应
type EndUserListResp struct {
	List     []EndUserListItemResp `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

// EndUserDeviceListReq 查询用户绑定设备列表（用于穿透查看）
type EndUserDeviceListReq struct {
	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=1,max=100"`

	UserID string `form:"user_id" binding:"required"`
}

// EndUserDeviceItemResp 用户绑定设备项
type EndUserDeviceItemResp struct {
	BindingID    string  `json:"binding_id"`
	DeviceID     string  `json:"device_id"`
	DeviceNumber string  `json:"device_number"`
	DeviceName   *string `json:"device_name"`
	IsOwner      bool    `json:"is_owner"`
	BindingTime  string  `json:"binding_time"`
}

// EndUserDeviceListResp 用户绑定设备列表响应
type EndUserDeviceListResp struct {
	List     []EndUserDeviceItemResp `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

// EndUserForceUnbindReq 强制解绑请求
type EndUserForceUnbindReq struct {
	BindingID string `json:"binding_id" binding:"required"`
}
