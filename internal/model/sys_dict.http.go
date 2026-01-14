package model

// CreateDictReq 创建字典条目
type CreateDictReq struct {
	DictCode  string  `json:"dict_code" validate:"required,max=36"`
	DictValue string  `json:"dict_value" validate:"required,max=255"`
	Category  string  `json:"category" validate:"required,max=100"`
	TenantID  *string `json:"tenant_id" validate:"omitempty,max=36"` // 仅 SYS_ADMIN 可用；0-系统全局
	Remark    *string `json:"remark" validate:"omitempty,max=255"`
}

type DictListReq struct {
	DictCode     string  `json:"dict_code" form:"dict_code" validate:"required,max=36"`
	LanguageCode *string `json:"language_code" form:"language_code" validate:"omitempty,max=36"`
}

type DictListRsp struct {
	DictValue   string `json:"dict_value" form:"dict_value"`
	Translation string `json:"translation" form:"translation"`
}

type GetDictLisyByPageReq struct {
	PageReq
	DictCode  *string `json:"dict_code" form:"dict_code" validate:"omitempty,max=36"`
	DictValue *string `json:"dict_value" form:"dict_value" validate:"omitempty,max=255"`
	Category  *string `json:"category" form:"category" validate:"omitempty,max=100"`
	// scope: all/global/tenant (默认 all). SYS_ADMIN 可额外通过 tenant_id 指定目标租户。
	Scope    *string `json:"scope" form:"scope" validate:"omitempty,oneof=all global tenant"`
	TenantID *string `json:"tenant_id" form:"tenant_id" validate:"omitempty,max=36"`
}

type ProtocolMenuReq struct {
	LanguageCode *string `json:"language_code" form:"language_code" validate:"omitempty,max=36"`
}

type UpdateDictReq struct {
	DictCode  *string `json:"dict_code" validate:"omitempty,max=36"`
	DictValue *string `json:"dict_value" validate:"omitempty,max=255"`
	Category  *string `json:"category" validate:"omitempty,max=100"`
	Remark    *string `json:"remark" validate:"omitempty,max=255"`
}

type DictCategoriesReq struct {
	// scope: all/global/tenant (默认 all). SYS_ADMIN 可额外通过 tenant_id 指定目标租户。
	Scope    *string `json:"scope" form:"scope" validate:"omitempty,oneof=all global tenant"`
	TenantID *string `json:"tenant_id" form:"tenant_id" validate:"omitempty,max=36"`
}

type DictCategoryItem struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

type DictValueReq struct {
	DictCode     string  `json:"dict_code" form:"dict_code" validate:"required,max=36"`
	DictValue    string  `json:"dict_value" form:"dict_value" validate:"required,max=255"`
	LanguageCode *string `json:"language_code" form:"language_code" validate:"omitempty,max=36"`
}
