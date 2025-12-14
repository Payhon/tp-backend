package model

// BatteryTagCreateReq 新增标签
type BatteryTagCreateReq struct {
	Name  string  `json:"name" binding:"required"`
	Color *string `json:"color"`
	Scene *string `json:"scene"`
}

// BatteryTagUpdateReq 更新标签
type BatteryTagUpdateReq struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
	Scene *string `json:"scene"`
}

// BatteryTagListReq 标签列表查询
type BatteryTagListReq struct {
	PageReq
	Name  *string `form:"name"`
	Scene *string `form:"scene"`
}

// BatteryTagItemResp 标签行
type BatteryTagItemResp struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Color       *string `json:"color"`
	Scene       *string `json:"scene"`
	DeviceCount int64   `json:"device_count"`
	CreatedAt   string  `json:"created_at"`
}

// BatteryTagListResp 标签列表响应
type BatteryTagListResp struct {
	List     []BatteryTagItemResp `json:"list"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

// BatteryTagAssignReq 给设备设置标签（批量）
type BatteryTagAssignReq struct {
	DeviceIDs []string `json:"device_ids" binding:"required,min=1"`
	TagIDs    []string `json:"tag_ids" binding:"omitempty"`
	Mode      *string  `json:"mode" binding:"omitempty,oneof=REPLACE APPEND"` // REPLACE: 覆盖；APPEND: 追加
}
