package model

// DeviceTransferReq 设备转移请求
type DeviceTransferReq struct {
	DeviceIDs  []string `json:"device_ids" binding:"required"`
	ToDealerID *string  `json:"to_dealer_id"` // 为空表示转移给厂家
	Remark     *string  `json:"remark"`
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

// ============================================================================
// 组织转移（新版，替代经销商转移）
// ============================================================================

// DeviceOrgTransferReq 设备组织转移请求
type DeviceOrgTransferReq struct {
	DeviceIDs []string `json:"device_ids" binding:"required,min=1"`
	ToOrgID   *string  `json:"to_org_id"` // 目标组织ID，为空表示退回厂家
	Remark    *string  `json:"remark"`
}

// DeviceOrgTransferListReq 组织转移记录查询请求
type DeviceOrgTransferListReq struct {
	Page         int     `form:"page" binding:"required,min=1"`
	PageSize     int     `form:"page_size" binding:"required,min=1,max=100"`
	DeviceNumber *string `form:"device_number"` // 设备编号模糊搜索
	FromOrgID    *string `form:"from_org_id"`   // 转出组织
	ToOrgID      *string `form:"to_org_id"`     // 转入组织
	StartTime    *string `form:"start_time"`    // 开始时间
	EndTime      *string `form:"end_time"`      // 结束时间
}

// DeviceOrgTransferResp 组织转移记录响应
type DeviceOrgTransferResp struct {
	ID           string  `json:"id"`
	DeviceID     string  `json:"device_id"`
	DeviceNumber string  `json:"device_number"`
	DeviceName   *string `json:"device_name"`
	FromOrgID    *string `json:"from_org_id"`
	FromOrgName  *string `json:"from_org_name"`
	FromOrgType  *string `json:"from_org_type"`
	ToOrgID      *string `json:"to_org_id"`
	ToOrgName    *string `json:"to_org_name"`
	ToOrgType    *string `json:"to_org_type"`
	OperatorID   *string `json:"operator_id"`
	OperatorName *string `json:"operator_name"`
	TransferTime *string `json:"transfer_time"`
	Remark       *string `json:"remark"`
}

// DeviceOrgTransferListResp 组织转移记录列表响应
type DeviceOrgTransferListResp struct {
	List     []DeviceOrgTransferResp `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}
