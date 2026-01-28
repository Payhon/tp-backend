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

// Auth 校验 MQTT 用户名/密码（支持 ACCESSTOKEN 空密码）
func (*MqttHTTPAuth) Auth(req *model.MqttHttpAuthReq) (bool, string) {
	clientID := strings.TrimSpace(req.ClientID)
	if strings.HasPrefix(clientID, "user_") {
		return authByUserPassword(req.Username, req.Password)
	}
	if strings.HasPrefix(clientID, "bms_") {
		return authByBmsDeviceID(clientID, req.Username, req.Password)
	}

	// device_ 开头或其他：走原有设备凭证逻辑
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return false, "username is required"
	}

	device, err := dal.GetDeviceByMQTTUsername(username)
	if err != nil {
		return false, "device not found"
	}
	return authByDeviceVoucher(device, username, req.Password)
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

func authByBmsDeviceID(clientID, username, password string) (bool, string) {
	deviceID := strings.TrimPrefix(clientID, "bms_")
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false, "device id is required"
	}

	device, err := dal.GetDeviceByID(deviceID)
	if err != nil {
		return false, "device not found"
	}
	return authByDeviceVoucher(device, username, password)
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
