package model

// DeviceTransferReq 设备转移请求
type DeviceTransferReq struct {
	DeviceIDs    []string `json:"device_ids" binding:"required"`
	ToDealerID   *string  `json:"to_dealer_id"` // 为空表示转移给厂家
	Remark       *string  `json:"remark"`
}

// DeviceTransferListReq 设备转移记录查询请求
type DeviceTransferListReq struct {
	Page         int     `form:"page" binding:"required,min=1"`
	PageSize     int     `form:"page_size" binding:"required,min=1,max=100"`
	DeviceNumber *string `form:"device_number"`
	FromDealerID *string `form:"from_dealer_id"`
	ToDealerID   *string `form:"to_dealer_id"`
	StartTime    *string `form:"start_time"`
	EndTime      *string `form:"end_time"`
}

// DeviceTransferResp 设备转移记录响应
type DeviceTransferResp struct {
	ID             string  `json:"id"`
	DeviceID       string  `json:"device_id"`
	DeviceNumber   string  `json:"device_number"`
	DeviceModel    string  `json:"device_model"`
	FromDealerID   *string `json:"from_dealer_id"`
	FromDealerName *string `json:"from_dealer_name"`
	ToDealerID     *string `json:"to_dealer_id"`
	ToDealerName   *string `json:"to_dealer_name"`
	OperatorID     *string `json:"operator_id"`
	OperatorName   *string `json:"operator_name"`
	TransferTime   string  `json:"transfer_time"`
	Remark         *string `json:"remark"`
}

// DeviceTransferListResp 设备转移记录列表响应
type DeviceTransferListResp struct {
	List     []DeviceTransferResp `json:"list"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}
