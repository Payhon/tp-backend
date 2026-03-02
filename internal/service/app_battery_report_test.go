package service

import (
	"strings"
	"testing"
)

func TestNormalizeAppBatteryCore_OK(t *testing.T) {
	core, err := normalizeAppBatteryCore(map[string]interface{}{
		"soc":         88,
		"chargeFetOn": true,
		"seriesCount": "16",
	})
	if err != nil {
		t.Fatalf("normalizeAppBatteryCore() error = %v", err)
	}

	if got, ok := core["soc"].(float64); !ok || got != 88 {
		t.Fatalf("soc mismatch, got=%v", core["soc"])
	}
	if got, ok := core["chargeFetOn"].(bool); !ok || !got {
		t.Fatalf("chargeFetOn mismatch, got=%v", core["chargeFetOn"])
	}
	if got, ok := core["seriesCount"].(string); !ok || got != "16" {
		t.Fatalf("seriesCount mismatch, got=%v", core["seriesCount"])
	}
}

func TestNormalizeAppBatteryCore_InvalidKey(t *testing.T) {
	_, err := normalizeAppBatteryCore(map[string]interface{}{
		"unknown_key": 1,
	})
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}
}

func TestNormalizeAppBatteryCore_InvalidValueType(t *testing.T) {
	_, err := normalizeAppBatteryCore(map[string]interface{}{
		"soc": map[string]interface{}{"v": 1},
	})
	if err == nil {
		t.Fatal("expected error for invalid value type, got nil")
	}
}

func TestNormalizeAppBatterySnapshot_MaxBytes(t *testing.T) {
	_, err := normalizeAppBatterySnapshot(map[string]interface{}{
		"raw": strings.Repeat("a", appBatterySnapshotMaxBytes),
	})
	if err == nil {
		t.Fatal("expected error for oversize snapshot, got nil")
	}
}
