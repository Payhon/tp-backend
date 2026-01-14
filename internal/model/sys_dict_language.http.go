package model

type CreateDictLanguageReq struct {
	DictId       string `json:"dict_id" validate:"required,max=36"`
	LanguageCode string `json:"language_code"  validate:"required,max=36"`
	Translation  string `json:"translation" validate:"required,max=255"`
}

// UpsertDictLanguageReq 新增/更新字典多语言（dict_id+language_code 唯一）
type UpsertDictLanguageReq struct {
	DictId       string `json:"dict_id" validate:"required,max=36"`
	LanguageCode string `json:"language_code" validate:"required,max=36"`
	Translation  string `json:"translation" validate:"required,max=255"`
}
