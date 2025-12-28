package model

import "time"

type GetFileListByPageReq struct {
	PageReq
	// Mine=true 时仅返回当前用户上传的文件（uploaded_by=claims.id）
	Mine            *bool      `json:"mine" form:"mine" validate:"omitempty"`
	Keyword         *string    `json:"keyword" form:"keyword" validate:"omitempty,max=255"`
	BizType         *string    `json:"biz_type" form:"biz_type" validate:"omitempty,max=50"`
	StorageLocation *string    `json:"storage_location" form:"storage_location" validate:"omitempty,oneof=local aliyun qiniu"`
	StartTime       *time.Time `json:"start_time" form:"start_time" validate:"omitempty"`
	EndTime         *time.Time `json:"end_time" form:"end_time" validate:"omitempty"`

	UploadedBy *string `json:"-" validate:"omitempty"`
}

type FileListItem struct {
	ID              string     `json:"id"`
	FileName        string     `json:"file_name"`
	FileSize        int64      `json:"file_size"`
	StorageLocation string     `json:"storage_location"`
	BizType         string     `json:"biz_type"`
	MimeType        *string    `json:"mime_type,omitempty"`
	FileExt         *string    `json:"file_ext,omitempty"`
	UploadedAt      time.Time  `json:"uploaded_at"`
	UploadedBy      *string    `json:"uploaded_by,omitempty"`
	Path            string     `json:"path"` // 本地为 ./files/...，云为 ./files-cloud/{id}
	URL             string     `json:"url"`  // full_url
}

type GetFileListByPageRsp struct {
	Total int64          `json:"total"`
	List  []FileListItem `json:"list"`
}

type CreateCloudUploadCredentialReq struct {
	BizType   string  `json:"biz_type" validate:"required,max=50"`
	FileName  string  `json:"file_name" validate:"required,max=255"`
	MimeType  *string `json:"mime_type" validate:"omitempty,max=100"`
	FileSize  *int64  `json:"file_size" validate:"omitempty,gte=0"`
}

type CreateCloudUploadCredentialRsp struct {
	Provider  string                 `json:"provider"`   // aliyun/qiniu
	ObjectKey string                 `json:"object_key"` // 上传到云存储的 key
	AccessURL string                 `json:"access_url"` // 上传后可访问的URL（直链）
	Upload    map[string]interface{} `json:"upload"`     // 直传参数（供应商相关）
}

type RegisterCloudFileReq struct {
	BizType  string  `json:"biz_type" validate:"required,max=50"`
	FileName string  `json:"file_name" validate:"required,max=255"`
	FileSize int64   `json:"file_size" validate:"required,gte=0"`
	MimeType *string `json:"mime_type" validate:"omitempty,max=100"`
	ObjectKey string `json:"object_key" validate:"required,max=500"`
}

type UploadFileRsp struct {
	ID              string `json:"id"`
	StorageLocation string `json:"storage_location"`
	Path            string `json:"path"`
	URL             string `json:"url"`
}
