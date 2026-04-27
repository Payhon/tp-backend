package service

import (
	"testing"

	model "project/internal/model"
)

func TestResolveOTADeviceKind(t *testing.T) {
	t.Run("default bms", func(t *testing.T) {
		if got := resolveOTADeviceKind(nil); got != model.OTADeviceKindBMS {
			t.Fatalf("expected default BMS, got %d", got)
		}
	})

	t.Run("meter", func(t *testing.T) {
		kind := model.OTADeviceKindMeter
		if got := resolveOTADeviceKind(&kind); got != model.OTADeviceKindMeter {
			t.Fatalf("expected meter, got %d", got)
		}
	})
}

func TestValidateOTAUpgradePackageReq(t *testing.T) {
	url := "https://example.com/a.bin"

	if err := validateOTAUpgradePackageReq(model.OTADeviceKindMeter, "meter", "", "", &url); err != nil {
		t.Fatalf("meter package should only require name and package_url, got %v", err)
	}

	if err := validateOTAUpgradePackageReq(model.OTADeviceKindBMS, "bms", "", "", &url); err == nil {
		t.Fatalf("bms package should still require version and device_config_id")
	}
}
