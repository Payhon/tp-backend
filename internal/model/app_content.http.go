package model

// ---------------------------------------------------------------------------
// APP端（无需登录）：单页内容/FAQ
// ---------------------------------------------------------------------------

type AppContentPageGetReq struct {
	AppID string  `form:"appid" json:"appid" validate:"required,max=100"` // 应用AppID（apps.appid）
	Lang  *string `form:"lang" json:"lang" validate:"omitempty,max=10"`   // zh-CN/en-US（其他语言默认en-US）
}

type AppContentPageResp struct {
	ContentKey      string `json:"content_key"`
	Lang            string `json:"lang"`
	Title           string `json:"title"`
	ContentMarkdown string `json:"content_markdown"`
	ContentHTML     string `json:"content_html"`
	UpdatedAt       string `json:"updated_at"`
}

type AppFaqListReq struct {
	PageReq
	AppID string  `form:"appid" json:"appid" validate:"required,max=100"` // 应用AppID（apps.appid）
	Lang  *string `form:"lang" json:"lang" validate:"omitempty,max=10"`   // zh-CN/en-US（其他语言默认en-US）
}

type AppFaqItemResp struct {
	ID              string `json:"id"`
	Question        string `json:"question"`
	AnswerMarkdown  string `json:"answer_markdown"`
	AnswerHTML      string `json:"answer_html"`
	IsPinned        bool   `json:"is_pinned"`
	Sort            int    `json:"sort"`
	UpdatedAt       string `json:"updated_at"`
}

type AppFaqListResp struct {
	List     []AppFaqItemResp `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// ---------------------------------------------------------------------------
// APP端（需登录）：用户反馈
// ---------------------------------------------------------------------------

type AppFeedbackCreateReq struct {
	AppID        string   `json:"appid" validate:"required,max=100"`            // 应用AppID（apps.appid）
	Content      string   `json:"content" validate:"required,min=1,max=10000"`  // 反馈内容
	Images       []string `json:"images" validate:"omitempty,max=20"`           // 上传后的图片 path 列表（/files/...）
	Platform     *string  `json:"platform" validate:"omitempty,max=30"`         // ios/android/harmony/web...
	AppVersion   *string  `json:"app_version" validate:"omitempty,max=50"`      // 客户端版本
	DeviceModel  *string  `json:"device_model" validate:"omitempty,max=100"`    // 设备型号
	OSVersion    *string  `json:"os_version" validate:"omitempty,max=50"`       // 系统版本
}

type AppFeedbackListReq struct {
	PageReq
	AppID *string `form:"appid" json:"appid" validate:"omitempty,max=100"`
}

type AppFeedbackItemResp struct {
	ID        string  `json:"id"`
	AppID     string  `json:"appid"`
	Content   string  `json:"content"`
	Images    []string `json:"images"`
	Status    string  `json:"status"`
	Reply     *string `json:"reply"`
	RepliedAt *string `json:"replied_at"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type AppFeedbackListResp struct {
	List     []AppFeedbackItemResp `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

// ---------------------------------------------------------------------------
// 管理端（WEB）：内容管理
// ---------------------------------------------------------------------------

type AdminContentPageGetReq struct {
	AppID string  `form:"app_id" json:"app_id" validate:"required,uuid"` // apps.id
	Lang  *string `form:"lang" json:"lang" validate:"omitempty,max=10"`
}

type AdminContentPageUpsertReq struct {
	AppID          string `json:"app_id" validate:"required,uuid"`      // apps.id
	Lang           string `json:"lang" validate:"required,max=10"`      // zh-CN/en-US
	Title          string `json:"title" validate:"omitempty,max=255"`
	ContentMarkdown string `json:"content_markdown" validate:"omitempty,max=100000"`
}

type AdminContentPageResp struct {
	AppID          string `json:"app_id"`
	ContentKey     string `json:"content_key"`
	Published      bool   `json:"published"`
	PublishedAt    *string `json:"published_at"`
	Lang           string `json:"lang"`
	Title          string `json:"title"`
	ContentMarkdown string `json:"content_markdown"`
	ContentHTML     string `json:"content_html"`
	UpdatedAt       string `json:"updated_at"`
}

type AdminContentPagePublishReq struct {
	AppID string `json:"app_id" validate:"required,uuid"`
}

type AdminFaqI18nPayload struct {
	Question       string `json:"question" validate:"omitempty,max=500"`
	AnswerMarkdown string `json:"answer_markdown" validate:"omitempty,max=100000"`
}

type AdminFaqCreateReq struct {
	AppID     string                         `json:"app_id" validate:"required,uuid"`
	IsPinned  bool                           `json:"is_pinned"`
	Sort      int                            `json:"sort" validate:"gte=0,lte=1000000"`
	Published bool                           `json:"published"`
	I18n      map[string]AdminFaqI18nPayload `json:"i18n" validate:"omitempty"`
}

type AdminFaqUpdateReq struct {
	IsPinned  *bool                          `json:"is_pinned" validate:"omitempty"`
	Sort      *int                           `json:"sort" validate:"omitempty,gte=0,lte=1000000"`
	Published *bool                          `json:"published" validate:"omitempty"`
	I18n      map[string]AdminFaqI18nPayload `json:"i18n" validate:"omitempty"`
}

type AdminFaqListReq struct {
	PageReq
	AppID     string  `form:"app_id" json:"app_id" validate:"required,uuid"`
	Lang      *string `form:"lang" json:"lang" validate:"omitempty,max=10"`
	Keyword   *string `form:"keyword" json:"keyword" validate:"omitempty,max=100"`
	Published *bool   `form:"published" json:"published" validate:"omitempty"`
}

type AdminFaqListItemResp struct {
	ID        string `json:"id"`
	Question  string `json:"question"`
	IsPinned  bool   `json:"is_pinned"`
	Sort      int    `json:"sort"`
	Published bool   `json:"published"`
	UpdatedAt string `json:"updated_at"`
}

type AdminFaqListResp struct {
	List     []AdminFaqListItemResp `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

type AdminFaqDetailResp struct {
	ID        string                         `json:"id"`
	AppID     string                         `json:"app_id"`
	IsPinned  bool                           `json:"is_pinned"`
	Sort      int                            `json:"sort"`
	Published bool                           `json:"published"`
	I18n      map[string]AdminFaqI18nPayload `json:"i18n"`
	UpdatedAt string                         `json:"updated_at"`
}

type AdminFeedbackListReq struct {
	PageReq
	AppID   string  `form:"app_id" json:"app_id" validate:"required,uuid"`
	Status  *string `form:"status" json:"status" validate:"omitempty,max=20"`
	Keyword *string `form:"keyword" json:"keyword" validate:"omitempty,max=100"`
}

type AdminFeedbackListItemResp struct {
	ID        string  `json:"id"`
	UserID    *string `json:"user_id"`
	Phone     *string `json:"phone"`
	Email     *string `json:"email"`
	Content   string  `json:"content"`
	ImageCnt  int     `json:"image_cnt"`
	Status    string  `json:"status"`
	Reply     *string `json:"reply"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type AdminFeedbackListResp struct {
	List     []AdminFeedbackListItemResp `json:"list"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
}

type AdminFeedbackDetailResp struct {
	ID        string   `json:"id"`
	AppID     string   `json:"app_id"`
	AppIDText string   `json:"appid"`
	UserID    *string  `json:"user_id"`
	Phone     *string  `json:"phone"`
	Email     *string  `json:"email"`
	Content   string   `json:"content"`
	Images    []string `json:"images"`
	Platform  *string  `json:"platform"`
	AppVersion *string `json:"app_version"`
	DeviceModel *string `json:"device_model"`
	OSVersion *string  `json:"os_version"`
	Status    string   `json:"status"`
	Reply     *string  `json:"reply"`
	RepliedAt *string  `json:"replied_at"`
	HandleNote *string `json:"handle_note"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type AdminFeedbackUpdateReq struct {
	Status     *string `json:"status" validate:"omitempty,max=20"`
	Reply      *string `json:"reply" validate:"omitempty,max=10000"`
	HandleNote *string `json:"handle_note" validate:"omitempty,max=10000"`
}

