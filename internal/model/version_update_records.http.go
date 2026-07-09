package model

// VersionUpdateCreateReq 新增版本更新记录
type VersionUpdateCreateReq struct {
	Project       string `json:"project" binding:"required,oneof=MOBILE CLOUD_FRONTEND CLOUD_BACKEND"`
	VersionNo     string `json:"version_no" binding:"required,max=64"`
	ReleaseDate   string `json:"release_date" binding:"required"`
	UpdateContent string `json:"update_content" binding:"required,max=10000"`
}

// VersionUpdateUpdateReq 更新版本更新记录
type VersionUpdateUpdateReq struct {
	Project       *string `json:"project" binding:"omitempty,oneof=MOBILE CLOUD_FRONTEND CLOUD_BACKEND"`
	VersionNo     *string `json:"version_no" binding:"omitempty,max=64"`
	ReleaseDate   *string `json:"release_date" binding:"omitempty"`
	UpdateContent *string `json:"update_content" binding:"omitempty,max=10000"`
}

// VersionUpdateListReq 版本更新记录列表
type VersionUpdateListReq struct {
	Page      int     `form:"page" binding:"required,min=1"`
	PageSize  int     `form:"page_size" binding:"required,min=1,max=100"`
	Project   *string `form:"project" binding:"omitempty,oneof=MOBILE CLOUD_FRONTEND CLOUD_BACKEND"`
	VersionNo *string `form:"version_no"`
	Keyword   *string `form:"keyword"`
	StartDate *string `form:"start_date"`
	EndDate   *string `form:"end_date"`
}

// VersionUpdateResp 版本更新记录响应
type VersionUpdateResp struct {
	ID            string  `json:"id"`
	Project       string  `json:"project"`
	VersionNo     string  `json:"version_no"`
	ReleaseDate   string  `json:"release_date"`
	UpdateContent string  `json:"update_content"`
	Source        string  `json:"source"`
	SourceRef     *string `json:"source_ref"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// VersionUpdateListResp 版本更新记录列表响应
type VersionUpdateListResp struct {
	List     []VersionUpdateResp `json:"list"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}
