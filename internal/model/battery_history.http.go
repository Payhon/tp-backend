package model

type BMSHistoryDeviceListReq struct {
	Page     int     `json:"page" form:"page" validate:"required,gte=1"`
	PageSize int     `json:"page_size" form:"page_size" validate:"required,gte=1,lte=200"`
	Keyword  *string `json:"keyword" form:"keyword" validate:"omitempty,max=100"`
}

type BMSHistoryDeviceItem struct {
	DeviceID     string  `json:"device_id"`
	DeviceNumber string  `json:"device_number"`
	DeviceName   *string `json:"device_name,omitempty"`
	ItemUUID     *string `json:"item_uuid,omitempty"`
	BmsCommType  *int    `json:"bms_comm_type,omitempty"`
	BleMac       *string `json:"ble_mac,omitempty"`
	CommChipID   *string `json:"comm_chip_id,omitempty"`
}

type BMSHistoryDeviceListResp struct {
	List     []BMSHistoryDeviceItem `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

type BMSHistoryQueryReq struct {
	DeviceID  string `json:"device_id" form:"device_id" validate:"required,max=36"`
	ViewMode  string `json:"view_mode" form:"view_mode" validate:"required,oneof=long wide"`
	StartTime int64  `json:"start_time" form:"start_time" validate:"required"`
	EndTime   int64  `json:"end_time" form:"end_time" validate:"required"`
	Page      int    `json:"page" form:"page" validate:"required,gte=1"`
	PageSize  int    `json:"page_size" form:"page_size" validate:"required,gte=1,lte=500"`
}

type BMSHistoryWideColumn struct {
	Key        string `json:"key"`
	DataType   string `json:"data_type"`
	Identifier string `json:"identifier"`
	DataName   string `json:"data_name"`
}

type BMSHistoryQueryResp struct {
	ViewMode string                 `json:"view_mode"`
	Columns  []BMSHistoryWideColumn `json:"columns,omitempty"`
	List     []map[string]any       `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

type BMSHistoryExportCreateReq struct {
	DeviceID  string `json:"device_id" validate:"required,max=36"`
	ViewMode  string `json:"view_mode" validate:"required,oneof=long wide"`
	StartTime int64  `json:"start_time" validate:"required"`
	EndTime   int64  `json:"end_time" validate:"required"`
}

type BMSHistoryExportCreateResp struct {
	JobID string `json:"job_id"`
}

type BMSHistoryExportPendingReq struct {
	Limit int `json:"limit" form:"limit" validate:"omitempty,gte=1,lte=100"`
}

type BMSHistoryExportJobItem struct {
	TaskID       string `json:"task_id"`
	DeviceID     string `json:"device_id"`
	DeviceNumber string `json:"device_number"`
	ViewMode     string `json:"view_mode"`
	StartTime    int64  `json:"start_time"`
	EndTime      int64  `json:"end_time"`
	FileName     string `json:"file_name"`
	DownloadURL  string `json:"download_url"`
	FinishedAt   string `json:"finished_at"`
}

type BMSHistoryExportPendingResp struct {
	List []BMSHistoryExportJobItem `json:"list"`
}

type BMSHistoryExportDownloadReq struct {
	TaskID string `json:"task_id" form:"task_id" validate:"required,max=36"`
}

type BMSHistoryExportWSMessage struct {
	Type        string `json:"type"`
	TaskID      string `json:"task_id"`
	DeviceID    string `json:"device_id"`
	FileName    string `json:"file_name"`
	DownloadURL string `json:"download_url"`
	FinishedAt  int64  `json:"finished_at"`
}
