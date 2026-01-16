package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"project/internal/dal"
	"project/internal/model"
)

// MqttHTTPAuth EMQX HTTP 认证服务
type MqttHTTPAuth struct{}

// Auth 校验 MQTT 用户名/密码（支持 ACCESSTOKEN 空密码）
func (*MqttHTTPAuth) Auth(req *model.MqttHttpAuthReq) (bool, string) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return false, "username is required"
	}

	device, err := dal.GetDeviceByMQTTUsername(username)
	if err != nil {
		return false, "device not found"
	}
	if device == nil || strings.TrimSpace(device.Voucher) == "" {
		return false, "voucher not found"
	}
	if !IsJSON(device.Voucher) {
		return false, "voucher is not valid json"
	}

	var voucher model.DeviceVoucher
	if err := json.Unmarshal([]byte(device.Voucher), &voucher); err != nil {
		return false, fmt.Sprintf("voucher parse failed: %v", err)
	}

	if voucher.Username != "" && voucher.Username != username {
		return false, "username mismatch"
	}

	expectedPassword := strings.TrimSpace(voucher.Password)
	providedPassword := strings.TrimSpace(req.Password)

	// ACCESSTOKEN：数据库无 password，则仅允许空密码
	if expectedPassword == "" {
		if providedPassword != "" {
			return false, "password mismatch"
		}
		return true, ""
	}

	if providedPassword != expectedPassword {
		return false, "password mismatch"
	}

	return true, ""
}
