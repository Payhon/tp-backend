package model

// DealerOverviewResp 经销商穿透概览（聚合数字）
type DealerOverviewResp struct {
	DealerID   string `json:"dealer_id"`
	DealerName string `json:"dealer_name"`

	DeviceCount int64 `json:"device_count"`
	ActiveCount int64 `json:"active_count"`

	EndUserCount int64 `json:"end_user_count"`

	WarrantyTotal      int64 `json:"warranty_total"`
	WarrantyPending    int64 `json:"warranty_pending"`
	WarrantyApproved   int64 `json:"warranty_approved"`
	WarrantyRejected   int64 `json:"warranty_rejected"`
	WarrantyProcessing int64 `json:"warranty_processing"`
	WarrantyCompleted  int64 `json:"warranty_completed"`
}
