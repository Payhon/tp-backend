package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"project/internal/model"
	"project/internal/query"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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

	got := mergeCurrentTelemetryIntoAppBatterySnapshot(snapshot, current, 0)
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

	got := mergeCurrentTelemetryIntoAppBatterySnapshot(snapshot, nil, 0)
	cell := got["cell"].(map[string]interface{})
	voltages := cell["voltagesMv"].([]interface{})
	if len(voltages) != 0 {
		t.Fatalf("expected invalid snapshot voltages to be cleared, got %#v", voltages)
	}
}

func TestMergeCurrentTelemetryIntoAppBatterySnapshot_DoesNotApplyOlderCurrentRows(t *testing.T) {
	snapshot := map[string]interface{}{
		"energy": map[string]interface{}{
			"socPct": float64(80),
		},
		"electrical": map[string]interface{}{
			"currentA": float64(5),
		},
	}
	current := map[string]model.AppBatteryCurrentTelemetryValue{
		"soc": {
			Value: float64(20),
			Ts:    1_000,
		},
		"currentA": {
			Value: float64(6),
			Ts:    3_000,
		},
	}

	got := mergeCurrentTelemetryIntoAppBatterySnapshot(snapshot, current, 2_000)
	energy := got["energy"].(map[string]interface{})
	if energy["socPct"].(float64) != 80 {
		t.Fatalf("older current SOC must not overwrite newer snapshot: %#v", energy["socPct"])
	}
	electrical := got["electrical"].(map[string]interface{})
	if electrical["currentA"].(float64) != 6 {
		t.Fatalf("newer current must still overlay snapshot: %#v", electrical["currentA"])
	}
}

func TestIsFourGBatteryDetail(t *testing.T) {
	commBLE := 1
	comm4G := 2
	commDual := 3
	chipID := "867123456789012"

	cases := []struct {
		name   string
		detail *model.AppBatteryDetailResp
		want   bool
	}{
		{name: "nil", detail: nil, want: false},
		{name: "ble only", detail: &model.AppBatteryDetailResp{BmsCommType: &commBLE}, want: false},
		{name: "4g", detail: &model.AppBatteryDetailResp{BmsCommType: &comm4G}, want: true},
		{name: "ble and 4g", detail: &model.AppBatteryDetailResp{BmsCommType: &commDual}, want: true},
		{name: "chip id fallback", detail: &model.AppBatteryDetailResp{CommChipID: &chipID}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFourGBatteryDetail(tc.detail); got != tc.want {
				t.Fatalf("isFourGBatteryDetail() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMqttInteractiveSnapshotOwnerMatches(t *testing.T) {
	nowMs := time.Now().UnixMilli()
	claims := &utils.UserClaims{ID: "user-1", TenantID: "tenant-1"}
	valid := &AppBatteryMqttSocketOwnerState{
		DeviceID:    "device-1",
		SessionID:   "session-1",
		UserID:      "user-1",
		TenantID:    "tenant-1",
		ExpiresAtTs: nowMs + 1_000,
	}
	if !mqttInteractiveSnapshotOwnerMatches(valid, "device-1", "session-1", claims, nowMs) {
		t.Fatal("matching live owner should be accepted")
	}

	cases := []struct {
		name      string
		owner     *AppBatteryMqttSocketOwnerState
		deviceID  string
		sessionID string
		claims    *utils.UserClaims
		nowMs     int64
	}{
		{name: "missing owner", owner: nil, deviceID: "device-1", sessionID: "session-1", claims: claims, nowMs: nowMs},
		{name: "wrong device", owner: valid, deviceID: "device-2", sessionID: "session-1", claims: claims, nowMs: nowMs},
		{name: "wrong session", owner: valid, deviceID: "device-1", sessionID: "session-2", claims: claims, nowMs: nowMs},
		{name: "wrong user", owner: valid, deviceID: "device-1", sessionID: "session-1", claims: &utils.UserClaims{ID: "user-2", TenantID: "tenant-1"}, nowMs: nowMs},
		{name: "wrong tenant", owner: valid, deviceID: "device-1", sessionID: "session-1", claims: &utils.UserClaims{ID: "user-1", TenantID: "tenant-2"}, nowMs: nowMs},
		{name: "expired", owner: valid, deviceID: "device-1", sessionID: "session-1", claims: claims, nowMs: nowMs + 2_000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if mqttInteractiveSnapshotOwnerMatches(tc.owner, tc.deviceID, tc.sessionID, tc.claims, tc.nowMs) {
				t.Fatal("mismatched or expired owner must be rejected")
			}
		})
	}
}

func TestBuildAppBatteryCurrentTelemetryRespKeepsInteractiveSnapshotAtomic(t *testing.T) {
	interactiveRaw := `{"energy":{"socPct":91},"electrical":{"currentA":4.5}}`
	reportedRaw := `{"energy":{"socPct":42},"electrical":{"currentA":1.5}}`
	currentSoc := float64(10)
	rows := []*model.TelemetryCurrentData{
		{
			DeviceID: "device-1",
			Key:      appBatteryMqttInteractiveSnapshotKey,
			T:        time.UnixMilli(2_000),
			StringV:  &interactiveRaw,
		},
		{
			DeviceID: "device-1",
			Key:      appBatterySnapshotKey,
			T:        time.UnixMilli(3_000),
			StringV:  &reportedRaw,
		},
		{
			DeviceID: "device-1",
			Key:      "soc",
			T:        time.UnixMilli(4_000),
			NumberV:  &currentSoc,
		},
	}

	resp := buildAppBatteryCurrentTelemetryResp("device-1", 0, rows)
	if resp.LastReportTs != 4_000 {
		t.Fatalf("last report timestamp = %d, want 4000", resp.LastReportTs)
	}
	if resp.InteractiveSnapshotTs != 2_000 {
		t.Fatalf("interactive snapshot timestamp = %d, want 2000", resp.InteractiveSnapshotTs)
	}
	energy := resp.InteractiveSnapshot["energy"].(map[string]interface{})
	if energy["socPct"].(float64) != 91 {
		t.Fatalf("interactive snapshot was mixed with ordinary current telemetry: %#v", resp.InteractiveSnapshot)
	}
	if _, exists := resp.Current[appBatteryMqttInteractiveSnapshotKey]; exists {
		t.Fatal("interactive snapshot storage key must not leak into ordinary current telemetry map")
	}
	reportedEnergy := resp.Snapshot["energy"].(map[string]interface{})
	if reportedEnergy["socPct"].(float64) != 10 {
		t.Fatalf("reported snapshot should preserve existing current merge behavior: %#v", resp.Snapshot)
	}
}

func setupAppBatteryInteractiveSnapshotTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:app-battery-interactive-%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE telemetry_current_datas (
			device_id TEXT NOT NULL,
			key TEXT NOT NULL,
			ts DATETIME NOT NULL,
			bool_v BOOLEAN,
			number_v REAL,
			string_v TEXT,
			tenant_id TEXT,
			UNIQUE(device_id, key)
		)
	`).Error; err != nil {
		t.Fatalf("create telemetry current table failed: %v", err)
	}

	oldDB := global.DB
	global.DB = db
	query.SetDefault(db)
	t.Cleanup(func() {
		global.DB = oldDB
		if oldDB != nil {
			query.SetDefault(oldDB)
		}
	})
	return db
}

func TestSaveAppBatteryMqttInteractiveSnapshotUsesMonotonicCurrentUpsert(t *testing.T) {
	db := setupAppBatteryInteractiveSnapshotTestDB(t)
	newerAt := time.UnixMilli(2_000).UTC()
	if err := saveAppBatteryMqttInteractiveSnapshot(
		context.Background(),
		"device-1",
		"tenant-1",
		`{"energy":{"socPct":88}}`,
		newerAt,
	); err != nil {
		t.Fatalf("save newer interactive snapshot failed: %v", err)
	}
	if err := saveAppBatteryMqttInteractiveSnapshot(
		context.Background(),
		"device-1",
		"tenant-1",
		`{"energy":{"socPct":20}}`,
		time.UnixMilli(1_000).UTC(),
	); err != nil {
		t.Fatalf("save older interactive snapshot failed: %v", err)
	}

	var got model.TelemetryCurrentData
	if err := db.Where("device_id = ? AND key = ?", "device-1", appBatteryMqttInteractiveSnapshotKey).First(&got).Error; err != nil {
		t.Fatalf("query interactive snapshot failed: %v", err)
	}
	if got.T.UnixMilli() != newerAt.UnixMilli() {
		t.Fatalf("interactive snapshot timestamp regressed to %d", got.T.UnixMilli())
	}
	if got.StringV == nil || !strings.Contains(*got.StringV, `"socPct":88`) {
		t.Fatalf("interactive snapshot payload regressed: %#v", got.StringV)
	}
}
