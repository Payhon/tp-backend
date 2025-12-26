package model

// OrgTypePermissionUpsertReq 机构类型权限更新请求
type OrgTypePermissionUpsertReq struct {
	UICodes               []string `json:"ui_codes"`                 // 菜单权限（sys_ui_elements.element_code）
	DeviceParamPermissions string   `json:"device_param_permissions"` // 设备参数权限（逗号分割字符串）
}

// OrgTypePermissionResp 机构类型权限响应
type OrgTypePermissionResp struct {
	OrgType               string   `json:"org_type"`
	UICodes               []string `json:"ui_codes"`
	DeviceParamPermissions string   `json:"device_param_permissions"`
}

// DeviceParamTreeNode 设备参数权限树节点
type DeviceParamTreeNode struct {
	Label string `json:"label"`
	Value string `json:"value"`
	// Children 为可选，支持层级结构；前端使用树形组件展示
	Children []DeviceParamTreeNode `json:"children,omitempty"`
}
