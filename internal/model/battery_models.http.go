package model

// BatteryModelCreateReq 创建电池型号请求
type BatteryModelCreateReq struct {
	SeqNo int16   `json:"seq_no" binding:"required,gte=1,lte=255"`
	Name  string  `json:"name" binding:"required,max=64"`
	OrgID *string `json:"org_id" binding:"omitempty,max=36"`
}

// BatteryModelUpdateReq 更新电池型号请求
type BatteryModelUpdateReq struct {
	SeqNo *int16  `json:"seq_no" binding:"omitempty,gte=1,lte=255"`
	Name  *string `json:"name" binding:"omitempty,max=64"`
	OrgID *string `json:"org_id" binding:"omitempty,max=36"`
}

// BatteryModelListReq 电池型号列表查询请求
// page/page_size 仅用于兼容历史调用；缺省时返回全量。
type BatteryModelListReq struct {
	Page     int     `form:"page"`
	PageSize int     `form:"page_size"`
	Name     *string `form:"name"`
	OrgID    *string `form:"org_id"`
}

// BatteryModelResp 电池型号响应
type BatteryModelResp struct {
	ID          string  `json:"id"`
	SeqNo       *int16  `json:"seq_no"`
	Name        string  `json:"name"`
	OrgID       *string `json:"org_id"`
	OrgName     *string `json:"org_name"`
	DeviceCount int64   `json:"device_count"`
	CreatedAt   string  `json:"created_at"`
}

// BatteryModelListResp 电池型号列表响应
type BatteryModelListResp struct {
	List     []BatteryModelResp `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}
