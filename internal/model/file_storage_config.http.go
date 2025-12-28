package model

type LocalStorageConfig struct {
	BaseDir         string `json:"base_dir" validate:"omitempty,max=255"`
	PublicPathPrefix string `json:"public_path_prefix" validate:"omitempty,max=255"` // 默认 /files
}

type AliyunOSSConfig struct {
	AccessKeyID        string `json:"access_key_id" validate:"omitempty,max=128"`
	AccessKeySecret    string `json:"access_key_secret" validate:"omitempty,max=256"`
	AccessKeySecretSet bool   `json:"access_key_secret_set" validate:"omitempty"`
	Endpoint           string `json:"endpoint" validate:"omitempty,max=255"`
	Bucket             string `json:"bucket" validate:"omitempty,max=128"`
	Domain             string `json:"domain" validate:"omitempty,max=500"`     // 自定义域名/CDN域名（可选）
	DirPrefix          string `json:"dir_prefix" validate:"omitempty,max=255"` // 例如 uploads/
	UseHTTPS           *bool  `json:"use_https" validate:"omitempty"`
}

type QiniuKodoConfig struct {
	AccessKey     string `json:"access_key" validate:"omitempty,max=128"`
	SecretKey     string `json:"secret_key" validate:"omitempty,max=256"`
	SecretKeySet  bool   `json:"secret_key_set" validate:"omitempty"`
	Bucket        string `json:"bucket" validate:"omitempty,max=128"`
	Domain        string `json:"domain" validate:"omitempty,max=500"`
	DirPrefix     string `json:"dir_prefix" validate:"omitempty,max=255"` // 例如 uploads/
	Region        string `json:"region" validate:"omitempty,max=50"`      // huadong/huabei/huanan/beimei/xinjiapo
	UseHTTPS      *bool  `json:"use_https" validate:"omitempty"`
	UploadBaseURL string `json:"upload_base_url" validate:"omitempty,max=500"` // 可选：直传上传域名
}

type UpsertFileStorageConfigReq struct {
	StorageType string             `json:"storage_type" validate:"required,oneof=local cloud"`
	Provider    string             `json:"provider" validate:"omitempty,oneof=aliyun qiniu"`
	Local       LocalStorageConfig `json:"local"`
	Aliyun      AliyunOSSConfig    `json:"aliyun"`
	Qiniu       QiniuKodoConfig    `json:"qiniu"`
	Remark      *string            `json:"remark" validate:"omitempty,max=255"`
}

type GetFileStorageConfigRsp struct {
	ID          string             `json:"id"`
	StorageType string             `json:"storage_type"`
	Provider    string             `json:"provider,omitempty"`
	Local       LocalStorageConfig `json:"local"`
	Aliyun      AliyunOSSConfig    `json:"aliyun"`
	Qiniu       QiniuKodoConfig    `json:"qiniu"`
	UpdatedAt   int64              `json:"updated_at"` // unix seconds
	Remark      *string            `json:"remark,omitempty"`
}

