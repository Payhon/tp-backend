package model

const (
	MESPackFactoryReassignStatusReassigned = "REASSIGNED"
	MESPackFactoryReassignStatusUnchanged  = "UNCHANGED"
	MESPackFactoryReassignStatusFailed     = "FAILED"
)

// MESPackFactoryReassignReq 第三方 MES 批量重新分配 PACK 厂请求。
type MESPackFactoryReassignReq struct {
	SerialNumbers         []string `json:"serial_numbers" binding:"required,min=1,max=500,dive,required,max=64"`
	TargetPackFactoryName string   `json:"target_pack_factory_name" binding:"required,max=100"`
	Remark                *string  `json:"remark,omitempty" binding:"omitempty,max=255"`
}

// MESPackFactoryReassignResult 单台 BMS 板重新分配结果。
type MESPackFactoryReassignResult struct {
	SerialNumber        string  `json:"serial_number"`
	Status              string  `json:"status"`
	FromPackFactoryName *string `json:"from_pack_factory_name,omitempty"`
	ToPackFactoryName   string  `json:"to_pack_factory_name"`
	Message             *string `json:"message,omitempty"`
}

// MESPackFactoryReassignResp 第三方 MES 批量重新分配 PACK 厂响应。
type MESPackFactoryReassignResp struct {
	RequestID             string                         `json:"request_id,omitempty"`
	TargetPackFactoryName string                         `json:"target_pack_factory_name"`
	Total                 int                            `json:"total"`
	Success               int                            `json:"success"`
	Unchanged             int                            `json:"unchanged"`
	Failed                int                            `json:"failed"`
	Results               []MESPackFactoryReassignResult `json:"results"`
}
