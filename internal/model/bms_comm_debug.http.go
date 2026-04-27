package model

type BmsCommDebugLogListReq struct {
	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=1,max=100"`

	DeviceID     *string `form:"device_id"`
	DeviceNumber *string `form:"device_number"`
	EventType    *string `form:"event_type"`
	Status       *string `form:"status"`
	StartTime    *string `form:"start_time"`
	EndTime      *string `form:"end_time"`
}

type BmsCommDebugLogItemResp struct {
	ID            int64   `json:"id"`
	OccurredAt    string  `json:"occurred_at"`
	DeviceID      string  `json:"device_id"`
	DeviceNumber  string  `json:"device_number"`
	Source        string  `json:"source"`
	AccessMode    string  `json:"access_mode"`
	EventType     string  `json:"event_type"`
	Direction     string  `json:"direction"`
	MQTTTopic     *string `json:"mqtt_topic"`
	QoS           *int    `json:"qos"`
	MessageID     *string `json:"message_id"`
	PayloadRaw    *string `json:"payload_raw"`
	PayloadFormat *string `json:"payload_format"`
	ParsedSummary any     `json:"parsed_summary"`
	Status        string  `json:"status"`
	ErrorMessage  *string `json:"error_message"`
}

type BmsCommDebugLogListResp struct {
	List     []BmsCommDebugLogItemResp `json:"list"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}
