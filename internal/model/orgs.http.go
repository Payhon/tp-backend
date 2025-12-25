package model

// OrgCreateReq 创建组织请求
type OrgCreateReq struct {
	Name          string  `json:"name" binding:"required"`     // 组织名称
	OrgType       string  `json:"org_type" binding:"required"` // 组织类型: BMS_FACTORY, PACK_FACTORY, DEALER, STORE
	ParentID      *string `json:"parent_id"`                   // 上级组织ID
	ContactPerson *string `json:"contact_person"`              // 联系人
	Phone         *string `json:"phone"`                       // 电话
	Email         *string `json:"email"`                       // 邮箱
	Province      *string `json:"province"`                    // 省份
	City          *string `json:"city"`                        // 城市
	District      *string `json:"district"`                    // 区县
	Address       *string `json:"address"`                     // 详细地址
	Remark        *string `json:"remark"`                      // 备注
}

// OrgUpdateReq 更新组织请求
type OrgUpdateReq struct {
	Name          *string `json:"name"`           // 组织名称
	ContactPerson *string `json:"contact_person"` // 联系人
	Phone         *string `json:"phone"`          // 电话
	Email         *string `json:"email"`          // 邮箱
	Province      *string `json:"province"`       // 省份
	City          *string `json:"city"`           // 城市
	District      *string `json:"district"`       // 区县
	Address       *string `json:"address"`        // 详细地址
	Status        *string `json:"status"`         // 状态: N-正常, F-禁用
	Remark        *string `json:"remark"`         // 备注
}

// OrgListReq 组织列表查询请求
type OrgListReq struct {
	Page     int     `form:"page" binding:"required,min=1"`
	PageSize int     `form:"page_size" binding:"required,min=1,max=100"`
	OrgType  *string `form:"org_type"`  // 按类型筛选
	Name     *string `form:"name"`      // 按名称模糊搜索
	Status   *string `form:"status"`    // 按状态筛选
	ParentID *string `form:"parent_id"` // 按父组织筛选
}

// OrgListResp 组织列表响应
type OrgListResp struct {
	List     []*Org `json:"list"`
	Total    int64  `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// OrgTreeNode 组织树节点
type OrgTreeNode struct {
	*Org
	Children []*OrgTreeNode `json:"children"`
}

// OrgTreeReq 获取组织树请求
type OrgTreeReq struct {
	OrgType *string `form:"org_type"` // 按类型筛选（可选）
}

// OrgDetailResp 组织详情响应（可扩展统计信息）
type OrgDetailResp struct {
	*Org
	ChildCount  int64 `json:"child_count"`  // 子组织数量
	DeviceCount int64 `json:"device_count"` // 关联设备数量
	UserCount   int64 `json:"user_count"`   // 关联用户数量
}
