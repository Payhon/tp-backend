package model

type AppUserSourceType string

const (
	AppUserSourceTypeAPP  AppUserSourceType = "APP"
	AppUserSourceTypeWXMP AppUserSourceType = "WXMP"
)

type AppUserSourceOptionResp struct {
	Key        string                    `json:"key"`
	Label      string                    `json:"label"`
	SourceType string                    `json:"source_type"`
	WxAppID    *string                   `json:"wx_appid,omitempty"`
	Children   []AppUserSourceOptionResp `json:"children,omitempty"`
}

type AppUserListReq struct {
	PageReq
	SourceType string  `json:"source_type" form:"source_type" validate:"required,oneof=APP WXMP"`
	WxAppID    *string `json:"wx_appid" form:"wx_appid" validate:"omitempty,max=100"`
	Keyword    *string `json:"keyword" form:"keyword" validate:"omitempty,max=100"`
	Status     *string `json:"status" form:"status" validate:"omitempty,oneof=N F"`
}

type AppUserListItemResp struct {
	ID            string  `json:"id"`
	PhoneNumber   string  `json:"phone_number"`
	Email         string  `json:"email"`
	Username      *string `json:"username"`
	Name          *string `json:"name"`
	Status        *string `json:"status"`
	UserKind      *string `json:"user_kind"`
	SourceType    string  `json:"source_type"`
	SourceName    string  `json:"source_name"`
	WxAppID       *string `json:"wx_appid"`
	IdentityTypes string  `json:"identity_types"`
	DeviceCount   int64   `json:"device_count"`
	LastBindAt    *string `json:"last_bind_at"`
	LastVisitTime *string `json:"last_visit_time"`
	CreatedAt     *string `json:"created_at"`
}

type AppUserListResp struct {
	List     []AppUserListItemResp `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}
