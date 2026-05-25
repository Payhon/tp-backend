package service

import (
	"strings"
	"testing"

	"project/internal/model"
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

func TestMergeCurrentTelemetryIntoAppBatterySnapshot_OverridesDynamicCellValues(t *testing.T) {
	snapshot := map[string]interface{}{
		"meta": map[string]interface{}{
			"seriesCount": float64(24),
		},
		"electrical": map[string]interface{}{
			"packCellSumVoltageV": float64(79.2),
			"cellVoltageIndex": map[string]interface{}{
				"highest": float64(3),
				"lowest":  float64(1),
			},
		},
		"cell": map[string]interface{}{
			"voltagesMv": []interface{}{float64(65535), float64(65535)},
		},
	}
	current := map[string]model.AppBatteryCurrentTelemetryValue{
		"cell.voltagesMv": {
			Value: `[3296,3300]`,
		},
		"electrical.packCellSumVoltageV": {
			Value: float64(79.201),
		},
		"electrical.cellVoltageIndex.highest": {
			Value: float64(2),
		},
	}

	got := mergeCurrentTelemetryIntoAppBatterySnapshot(snapshot, current)
	cell := got["cell"].(map[string]interface{})
	voltages := cell["voltagesMv"].([]interface{})
	if voltages[0].(float64) != 3296 || voltages[1].(float64) != 3300 {
		t.Fatalf("cell voltages not overridden: %#v", voltages)
	}
	electrical := got["electrical"].(map[string]interface{})
	if electrical["packCellSumVoltageV"].(float64) != 79.201 {
		t.Fatalf("pack voltage not overridden: %#v", electrical["packCellSumVoltageV"])
	}
	idx := electrical["cellVoltageIndex"].(map[string]interface{})
	if idx["highest"].(float64) != 2 {
		t.Fatalf("highest index not overridden: %#v", idx["highest"])
	}
}

func TestMergeCurrentTelemetryIntoAppBatterySnapshot_ClearsInvalidSnapshotCellVoltages(t *testing.T) {
	snapshot := map[string]interface{}{
		"cell": map[string]interface{}{
			"voltagesMv": []interface{}{float64(65535), float64(65535)},
		},
	}

	got := mergeCurrentTelemetryIntoAppBatterySnapshot(snapshot, nil)
	cell := got["cell"].(map[string]interface{})
	voltages := cell["voltagesMv"].([]interface{})
	if len(voltages) != 0 {
		t.Fatalf("expected invalid snapshot voltages to be cleared, got %#v", voltages)
	}
}
