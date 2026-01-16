package model

// MqttHttpAuthReq EMQX HTTP 认证请求
// 兼容 EMQX 4.x/5.x 默认字段
type MqttHttpAuthReq struct {
	ClientID   string `json:"clientid" form:"clientid"`
	Username   string `json:"username" form:"username"`
	Password   string `json:"password" form:"password"`
	Protocol   string `json:"protocol" form:"protocol"`
	Peername   string `json:"peername" form:"peername"`
	Sockname   string `json:"sockname" form:"sockname"`
	Mountpoint string `json:"mountpoint" form:"mountpoint"`
}

// MqttHttpAuthResp EMQX HTTP 认证响应
type MqttHttpAuthResp struct {
	Result      string `json:"result"`                 // allow | deny | ignore
	IsSuperuser bool   `json:"is_superuser,omitempty"` // 默认 false
	Reason      string `json:"reason,omitempty"`
}
