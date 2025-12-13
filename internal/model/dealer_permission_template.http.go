package model

// DealerPermissionTemplateResp 经销商权限模板
type DealerPermissionTemplateResp struct {
	DealerID  string `json:"dealer_id"`
	Template  string `json:"template"`  // BASIC/ADVANCED/NONE
	RoleName  string `json:"role_name"` // 对应 casbin 角色名
	UserCount int64  `json:"user_count"`
}

// DealerPermissionTemplateReq 设置经销商权限模板
type DealerPermissionTemplateReq struct {
	Template string `json:"template" binding:"required,oneof=BASIC ADVANCED"`
}
