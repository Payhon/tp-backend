package model

import "time"

// SysDictRecord 用于包含 sys_dict 新增字段（tenant_id/category）的查询/写入
// 注意：gorm/gen 自动生成的 SysDict 结构体不包含新增字段，因此这里单独定义一份。
type SysDictRecord struct {
	ID        string    `gorm:"column:id;primaryKey" json:"id"`
	DictCode  string    `gorm:"column:dict_code" json:"dict_code"`
	DictValue string    `gorm:"column:dict_value" json:"dict_value"`
	TenantID  string    `gorm:"column:tenant_id" json:"tenant_id"`
	Category  string    `gorm:"column:category" json:"category"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	Remark    *string   `gorm:"column:remark" json:"remark"`
}

func (*SysDictRecord) TableName() string { return "sys_dict" }
