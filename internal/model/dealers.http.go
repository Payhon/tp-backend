package model

// DealerCreateReq 创建经销商请求
type DealerCreateReq struct {
	Name          string  `json:"name" binding:"required"`
	ContactPerson *string `json:"contact_person"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Province      *string `json:"province"`
	City          *string `json:"city"`
	District      *string `json:"district"`
	Address       *string `json:"address"`
	ParentID      *string `json:"parent_id"`
	Remark        *string `json:"remark"`
}

// DealerUpdateReq 更新经销商请求
type DealerUpdateReq struct {
	Name          *string `json:"name"`
	ContactPerson *string `json:"contact_person"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Province      *string `json:"province"`
	City          *string `json:"city"`
	District      *string `json:"district"`
	Address       *string `json:"address"`
	ParentID      *string `json:"parent_id"`
	Remark        *string `json:"remark"`
}

// DealerListReq 经销商列表查询请求
type DealerListReq struct {
	Page     int     `form:"page" binding:"required,min=1"`
	PageSize int     `form:"page_size" binding:"required,min=1,max=1000"`
	Name     *string `form:"name"`
	Phone    *string `form:"phone"`
	Province *string `form:"province"`
	City     *string `form:"city"`
}

// DealerResp 经销商响应
type DealerResp struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ContactPerson *string `json:"contact_person"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Province      *string `json:"province"`
	City          *string `json:"city"`
	District      *string `json:"district"`
	Address       *string `json:"address"`
	ParentID      *string `json:"parent_id"`
	DeviceCount   int64   `json:"device_count"` // 设备总数
	ActiveCount   int64   `json:"active_count"` // 激活设备数
	CreatedAt     string  `json:"created_at"`
	Remark        *string `json:"remark"`
}

// DealerListResp 经销商列表响应
type DealerListResp struct {
	List     []DealerResp `json:"list"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}
