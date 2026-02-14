// internal/model/open_api_keys.http.go
package model

import "time"

// OpenAPIKeyListReq 查询API密钥列表请求
type OpenAPIKeyListReq struct {
	PageReq
	Status   *int16  `json:"status" form:"status" validate:"omitempty,oneof=0 1"` // 状态: 0-禁用 1-启用
	TenantID *string `json:"tenant_id" form:"tenant_id" validate:"omitempty,max=36"`
	Keyword  *string `json:"keyword" form:"keyword" validate:"omitempty,max=100"` // app_id / 备注
}

// CreateOpenAPIKeyReq 创建API密钥请求
type CreateOpenAPIKeyReq struct {
	TenantID  *string `json:"tenant_id" validate:"omitempty,max=36"`  // SYS_ADMIN 可指定，租户管理员自动取 claims tenant_id
	Remark    *string `json:"remark" validate:"omitempty,max=255"`    // 备注
	ExpiredAt *string `json:"expired_at" validate:"omitempty,max=40"` // 过期时间，支持 RFC3339 或 2006-01-02 15:04:05
	Name      *string `json:"name" validate:"omitempty,max=200"`      // 兼容旧前端（映射到备注）
	Status    *int16  `json:"status" validate:"omitempty,oneof=0 1"`  // 默认 1
}

// UpdateOpenAPIKeyReq 更新API密钥请求
type UpdateOpenAPIKeyReq struct {
	ID           string  `json:"id" validate:"required,max=36"`          // 主键ID
	Status       *int16  `json:"status" validate:"omitempty,oneof=0 1"`  // 状态: 0-禁用 1-启用
	Remark       *string `json:"remark" validate:"omitempty,max=255"`    // 备注
	ExpiredAt    *string `json:"expired_at" validate:"omitempty,max=40"` // 过期时间，空串表示清空
	RotateSecret *bool   `json:"rotate_secret"`                          // 是否重置 SecretKey
	Name         *string `json:"name" validate:"omitempty,max=200"`      // 兼容旧前端（映射到备注）
}

// OpenAPIKeyListRsp API密钥列表响应
type OpenAPIKeyListRsp struct {
	ID        string  `json:"id" gorm:"column:id"`
	TenantID  string  `json:"tenant_id" gorm:"column:tenant_id"`
	AppID     string  `json:"app_id" gorm:"column:api_key"`  // 对外字段
	APIKey    string  `json:"api_key" gorm:"column:api_key"` // 兼容旧前端字段
	SecretKey *string `json:"secret_key" gorm:"column:secret_key"`
	Remark    *string `json:"remark" gorm:"column:remark"`
	// 兼容旧前端字段
	Name       *string    `json:"name" gorm:"column:name"`
	Status     *int16     `json:"status" gorm:"column:status"`
	ExpiredAt  *time.Time `json:"expired_at" gorm:"column:expired_at"`
	LastUsedAt *time.Time `json:"last_used_at" gorm:"column:last_used_at"`
	CreatedAt  *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt  *time.Time `json:"updated_at" gorm:"column:updated_at"`
	CreatedID  *string    `json:"created_id" gorm:"column:created_id"`

	UserID   *string `json:"user_id" gorm:"column:user_id"`
	Email    *string `json:"email" gorm:"column:email"`
	UserName *string `json:"user_name" gorm:"column:user_name"`
}
