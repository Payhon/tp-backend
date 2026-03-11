package model

// BatteryCellBrandCreateReq 新增电芯品牌
type BatteryCellBrandCreateReq struct {
	SeqNo int16  `json:"seq_no" binding:"required,gte=1,lte=255"`
	Name  string `json:"name" binding:"required,max=16"`
}

// BatteryCellBrandUpdateReq 更新电芯品牌
type BatteryCellBrandUpdateReq struct {
	SeqNo *int16  `json:"seq_no" binding:"omitempty,gte=1,lte=255"`
	Name  *string `json:"name" binding:"omitempty,max=16"`
}

// BatteryCellBrandListReq 电芯品牌列表
type BatteryCellBrandListReq struct {
	Name *string `form:"name"`
}

// BatteryCellBrandItemResp 电芯品牌项
type BatteryCellBrandItemResp struct {
	ID        string `json:"id"`
	SeqNo     int16  `json:"seq_no"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// BatteryCellBrandListResp 电芯品牌列表响应
type BatteryCellBrandListResp struct {
	List  []BatteryCellBrandItemResp `json:"list"`
	Total int64                      `json:"total"`
}
