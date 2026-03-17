package model

import "time"

type CreateRoleReq struct {
	Name        string  `json:"name" validate:"required,max=255"`         //角色名称
	Description *string `json:"description" validate:"omitempty,max=500"` //角色描述
	Authority   *string `json:"authority" validate:"omitempty,oneof=TENANT_ADMIN TENANT_USER"` // 角色适用账号类型
	UserKind    *string `json:"user_kind" validate:"omitempty,oneof=ORG_USER END_USER"`        // 角色适用用户类型
	OrgType     *string `json:"org_type" validate:"omitempty,max=50"`                           // 角色适用组织类型
}

type UpdateRoleReq struct {
	Id          string     `json:"id" validate:"required,max=36"`
	Name        string     `json:"name" validate:"required,max=255"`         //角色名称
	Description *string    `json:"description" validate:"omitempty,max=500"` //角色描述
	UpdatedAt   *time.Time `json:"updated_at" validate:"omitempty"`          //修改时间，前端不用传
	Authority   *string    `json:"authority" validate:"omitempty,oneof=TENANT_ADMIN TENANT_USER"` // 角色适用账号类型
	UserKind    *string    `json:"user_kind" validate:"omitempty,oneof=ORG_USER END_USER"`         // 角色适用用户类型
	OrgType     *string    `json:"org_type" validate:"omitempty,max=50"`                            // 角色适用组织类型
}

type GetRoleListByPageReq struct {
	PageReq
	Name *string `json:"name" form:"name" validate:"omitempty,max=255"`
}
