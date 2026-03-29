package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"project/internal/dal"
	"project/internal/model"
	"project/pkg/utils"

	"gorm.io/gorm"
)

// MqttHTTPAuth EMQX HTTP 认证服务
type MqttHTTPAuth struct{}

const (
	mqttAuthResultAllow  = "allow"
	mqttAuthResultDeny   = "deny"
	mqttAuthResultIgnore = "ignore"
)

// Auth 校验 MQTT 用户名/密码（支持 ACCESSTOKEN 空密码）
func (*MqttHTTPAuth) Auth(req *model.MqttHttpAuthReq) (string, string) {
	return authMQTTRequest(req, dal.GetDeviceByMQTTUsername, dal.GetDeviceByID, authByUserPassword)
}

func authMQTTRequest(
	req *model.MqttHttpAuthReq,
	findDeviceByUsername func(string) (*model.Device, error),
	findDeviceByID func(string) (*model.Device, error),
	authUser func(string, string) (bool, string),
) (string, string) {
	clientID := strings.TrimSpace(req.ClientID)
	if strings.HasPrefix(clientID, "user_") {
		return boolResult(authUser(req.Username, req.Password))
	}
	if strings.HasPrefix(clientID, "bms_") {
		return authByBmsDeviceID(clientID, req.Username, req.Password, findDeviceByID)
	}

	// 其他客户端仅在 username 能映射到设备凭证时才由 HTTP 认证处理。
	// 显式返回 ignore，避免把系统内部 MQTT 客户端错误地 deny 掉。
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return mqttAuthResultIgnore, "client not handled by http auth"
	}

	device, err := findDeviceByUsername(username)
	if err != nil {
		if isMQTTDeviceLookupMiss(err) {
			return mqttAuthResultIgnore, "client not handled by http auth"
		}
		return mqttAuthResultDeny, "device lookup failed"
	}
	return boolResult(authByDeviceVoucher(device, username, req.Password))
}

func boolResult(ok bool, reason string) (string, string) {
	if ok {
		return mqttAuthResultAllow, ""
	}
	return mqttAuthResultDeny, reason
}

func isMQTTDeviceLookupMiss(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "device not found")
}

func authByUserPassword(username, password string) (bool, string) {
	u := strings.TrimSpace(username)
	if u == "" {
		return false, "username is required"
	}
	p := strings.TrimSpace(password)
	if p == "" {
		return false, "password is required"
	}

	user, err := dal.GetUsersByEmail(u)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user, err = dal.GetUsersByPhoneNumber(u)
		}
		if err != nil {
			return false, "user not found"
		}
	}
	if user == nil {
		return false, "user not found"
	}
	if user.Status != nil && *user.Status != "N" {
		return false, "user disabled"
	}
	if !utils.BcryptCheck(p, user.Password) {
		return false, "password mismatch"
	}
	return true, ""
}

func authByBmsDeviceID(clientID, username, password string, findDeviceByID func(string) (*model.Device, error)) (string, string) {
	deviceID := strings.TrimPrefix(clientID, "bms_")
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return mqttAuthResultDeny, "device id is required"
	}

	device, err := findDeviceByID(deviceID)
	if err != nil {
		return mqttAuthResultDeny, "device not found"
	}
	return boolResult(authByDeviceVoucher(device, username, password))
}

func authByDeviceVoucher(device *model.Device, username, password string) (bool, string) {
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

	reqUsername := strings.TrimSpace(username)
	if reqUsername == "" {
		return false, "username is required"
	}
	if voucher.Username != "" && voucher.Username != reqUsername {
		return false, "username mismatch"
	}

	expectedPassword := strings.TrimSpace(voucher.Password)
	providedPassword := strings.TrimSpace(password)

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
