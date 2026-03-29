package service

import (
	"errors"
	"testing"

	"project/internal/model"
)

func TestAuthMQTTRequestIgnoresUnsupportedServerClient(t *testing.T) {
	t.Parallel()

	result, reason := authMQTTRequest(
		&model.MqttHttpAuthReq{
			ClientID: "server-bms-bridge",
			Username: "root",
			Password: "root",
		},
		func(username string) (*model.Device, error) {
			if username != "root" {
				t.Fatalf("unexpected username lookup: %s", username)
			}
			return nil, errors.New("device not found for username: root")
		},
		func(string) (*model.Device, error) {
			t.Fatal("unexpected device-id lookup")
			return nil, nil
		},
		func(string, string) (bool, string) {
			t.Fatal("unexpected user auth")
			return false, ""
		},
	)

	if result != mqttAuthResultIgnore {
		t.Fatalf("expected ignore, got %s", result)
	}
	if reason != "client not handled by http auth" {
		t.Fatalf("unexpected reason: %s", reason)
	}
}

func TestAuthMQTTRequestAllowsKnownDeviceVoucher(t *testing.T) {
	t.Parallel()

	result, reason := authMQTTRequest(
		&model.MqttHttpAuthReq{
			ClientID: "mqtt_abc123",
			Username: "dev-user",
			Password: "dev-pass",
		},
		func(username string) (*model.Device, error) {
			if username != "dev-user" {
				t.Fatalf("unexpected username lookup: %s", username)
			}
			return &model.Device{Voucher: `{"username":"dev-user","password":"dev-pass"}`}, nil
		},
		func(string) (*model.Device, error) {
			t.Fatal("unexpected device-id lookup")
			return nil, nil
		},
		func(string, string) (bool, string) {
			t.Fatal("unexpected user auth")
			return false, ""
		},
	)

	if result != mqttAuthResultAllow {
		t.Fatalf("expected allow, got %s (%s)", result, reason)
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %s", reason)
	}
}

func TestAuthMQTTRequestDeniesManagedUserClientOnPasswordMismatch(t *testing.T) {
	t.Parallel()

	result, reason := authMQTTRequest(
		&model.MqttHttpAuthReq{
			ClientID: "user_ops",
			Username: "ops@example.com",
			Password: "bad-pass",
		},
		func(string) (*model.Device, error) {
			t.Fatal("unexpected device lookup")
			return nil, nil
		},
		func(string) (*model.Device, error) {
			t.Fatal("unexpected device-id lookup")
			return nil, nil
		},
		func(username, password string) (bool, string) {
			if username != "ops@example.com" || password != "bad-pass" {
				t.Fatalf("unexpected auth payload: %s / %s", username, password)
			}
			return false, "password mismatch"
		},
	)

	if result != mqttAuthResultDeny {
		t.Fatalf("expected deny, got %s", result)
	}
	if reason != "password mismatch" {
		t.Fatalf("unexpected reason: %s", reason)
	}
}
