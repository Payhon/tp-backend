package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"project/internal/model"
	"project/internal/query"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func setupDeviceLastConnectedTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:last-connected-%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	schema := []string{
		`CREATE TABLE devices (
			id TEXT PRIMARY KEY,
			device_number TEXT,
			name TEXT,
			tenant_id TEXT,
			is_online INTEGER,
			current_version TEXT,
			remark1 TEXT,
			last_connected_at DATETIME,
			last_offline_time DATETIME
		)`,
		`CREATE TABLE device_batteries (
			device_id TEXT PRIMARY KEY,
			bms_comm_type INTEGER,
			battery_model_id TEXT,
			item_uuid TEXT,
			batch_number TEXT,
			product_spec TEXT,
			order_number TEXT,
			ble_mac TEXT,
			identity_ble_mac TEXT,
			comm_chip_id TEXT,
			production_date DATETIME,
			warranty_expire_date DATETIME,
			soc REAL,
			soh REAL,
			updated_at DATETIME
		)`,
		`CREATE TABLE battery_bms_models (
			id TEXT PRIMARY KEY,
			name TEXT
		)`,
	}
	for _, statement := range schema {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create device last connected test schema failed: %v", err)
		}
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

func useDeviceLastConnectedServiceDependencies(t *testing.T) {
	t.Helper()

	oldRedis := global.REDIS
	oldStatusRedis := global.STATUS_REDIS
	failingRedis := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 5 * time.Millisecond,
		ReadTimeout: 5 * time.Millisecond,
		MaxRetries:  -1,
	})
	global.REDIS = failingRedis
	global.STATUS_REDIS = failingRedis

	t.Cleanup(func() {
		_ = failingRedis.Close()
		global.REDIS = oldRedis
		global.STATUS_REDIS = oldStatusRedis
	})
}

func TestTouchDeviceLastConnectedAtUsesServerTime(t *testing.T) {
	db := setupDeviceLastConnectedTestDB(t)
	if err := db.Exec(`INSERT INTO devices (id) VALUES (?)`, "device-1").Error; err != nil {
		t.Fatalf("insert device failed: %v", err)
	}

	before := time.Now().UTC().Add(-time.Second)
	if err := touchDeviceLastConnectedAt(context.Background(), "device-1"); err != nil {
		t.Fatalf("touch last connected failed: %v", err)
	}
	after := time.Now().UTC().Add(time.Second)

	var saved struct {
		LastConnectedAt *time.Time `gorm:"column:last_connected_at"`
	}
	if err := db.Table("devices").Select("last_connected_at").Where("id = ?", "device-1").Scan(&saved).Error; err != nil {
		t.Fatalf("query last connected failed: %v", err)
	}
	if saved.LastConnectedAt == nil {
		t.Fatal("expected last_connected_at to be populated")
	}
	if saved.LastConnectedAt.Before(before) || saved.LastConnectedAt.After(after) {
		t.Fatalf("last_connected_at %s is outside server time window [%s, %s]", saved.LastConnectedAt, before, after)
	}
}

func TestTouchDeviceLastConnectedAtRejectsInvalidDevice(t *testing.T) {
	setupDeviceLastConnectedTestDB(t)

	if err := touchDeviceLastConnectedAt(context.Background(), ""); err == nil {
		t.Fatal("expected empty device id to fail")
	}
	if err := touchDeviceLastConnectedAt(context.Background(), "missing-device"); err == nil {
		t.Fatal("expected missing device to fail")
	}
}

func TestSuccessfulBleConnectionReportTouchesLastConnectedAt(t *testing.T) {
	db := setupDeviceLastConnectedTestDB(t)
	useDeviceLastConnectedServiceDependencies(t)
	if err := db.Exec(
		`INSERT INTO devices (id, device_number, tenant_id, is_online) VALUES (?, ?, ?, ?)`,
		"ble-device",
		"BLE-001",
		"tenant-1",
		1,
	).Error; err != nil {
		t.Fatalf("insert BLE device failed: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO device_batteries (device_id, bms_comm_type, ble_mac) VALUES (?, ?, ?)`,
		"ble-device",
		1,
		"AC1122334455",
	).Error; err != nil {
		t.Fatalf("insert BLE battery failed: %v", err)
	}

	resp, err := new(AppBattery).ReportBatteryConnectionStatusForApp(
		context.Background(),
		model.AppBatteryConnectionStatusReq{
			DeviceID:     "ble-device",
			ConnType:     "bluetooth",
			BleConnected: true,
		},
		&utils.UserClaims{ID: "user-1", TenantID: "tenant-1", Authority: "TENANT_ADMIN"},
	)
	if err != nil {
		t.Fatalf("report successful BLE connection failed: %v", err)
	}
	if resp == nil || !resp.Accepted {
		t.Fatalf("successful BLE connection was not accepted: %#v", resp)
	}

	var lastConnectedAt *time.Time
	if err := db.Table("devices").
		Select("last_connected_at").
		Where("id = ?", "ble-device").
		Scan(&lastConnectedAt).Error; err != nil {
		t.Fatalf("query BLE last connected failed: %v", err)
	}
	if lastConnectedAt == nil {
		t.Fatal("successful BLE connection should update last_connected_at")
	}
}

func TestFourGInteractionTouchesLastConnectedAt(t *testing.T) {
	db := setupDeviceLastConnectedTestDB(t)
	useDeviceLastConnectedServiceDependencies(t)
	if err := db.Exec(
		`INSERT INTO devices (id, device_number, tenant_id, is_online) VALUES (?, ?, ?, ?)`,
		"4g-device",
		"4G-001",
		"tenant-1",
		1,
	).Error; err != nil {
		t.Fatalf("insert 4G device failed: %v", err)
	}

	commType := 2
	changed, err := new(AppBattery).MarkFourGBatteryOnlineByInteraction(
		context.Background(),
		&model.AppBatteryDetailResp{DeviceID: "4g-device", BmsCommType: &commType},
		"mqtt_socket_uplink",
	)
	if err != nil {
		t.Fatalf("mark 4G interaction failed: %v", err)
	}
	if changed {
		t.Fatal("already-online 4G device should not report a status transition")
	}

	var lastConnectedAt *time.Time
	if err := db.Table("devices").
		Select("last_connected_at").
		Where("id = ?", "4g-device").
		Scan(&lastConnectedAt).Error; err != nil {
		t.Fatalf("query 4G last connected failed: %v", err)
	}
	if lastConnectedAt == nil {
		t.Fatal("real 4G interaction should update last_connected_at")
	}
}

func TestAppDeviceOrderClause(t *testing.T) {
	tests := []struct {
		name     string
		viewMode string
		want     string
	}{
		{
			name:     "self bound uses binding fallback",
			viewMode: model.AppDeviceViewModeSelfBound,
			want:     "d.last_connected_at DESC NULLS LAST, CASE WHEN d.last_connected_at IS NULL THEN dub.binding_time END DESC, d.id ASC",
		},
		{
			name:     "org added uses added fallback",
			viewMode: model.AppDeviceViewModeOrgAdded,
			want:     "d.last_connected_at DESC NULLS LAST, CASE WHEN d.last_connected_at IS NULL THEN ar.added_at END DESC, d.id ASC",
		},
		{
			name:     "end user bound uses binding fallback",
			viewMode: model.AppDeviceViewModeEndUserBind,
			want:     "d.last_connected_at DESC NULLS LAST, CASE WHEN d.last_connected_at IS NULL THEN lb.binding_time END DESC, d.id ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appDeviceOrderClause(tt.viewMode); got != tt.want {
				t.Fatalf("appDeviceOrderClause(%q) = %q, want %q", tt.viewMode, got, tt.want)
			}
		})
	}
}

func TestAppDeviceOrderClauseSortsConnectedBeforeFallbackRows(t *testing.T) {
	tests := []struct {
		name      string
		viewMode  string
		joinAlias string
	}{
		{name: "self bound", viewMode: model.AppDeviceViewModeSelfBound, joinAlias: "dub"},
		{name: "org added", viewMode: model.AppDeviceViewModeOrgAdded, joinAlias: "ar"},
		{name: "end user bound", viewMode: model.AppDeviceViewModeEndUserBind, joinAlias: "lb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := fmt.Sprintf("file:device-order-%s?mode=memory&cache=shared", t.Name())
			db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
			if err != nil {
				t.Fatalf("open sqlite failed: %v", err)
			}
			for _, statement := range []string{
				`CREATE TABLE devices (id TEXT PRIMARY KEY, last_connected_at DATETIME)`,
				`CREATE TABLE relations (
					device_id TEXT PRIMARY KEY,
					binding_time DATETIME,
					added_at DATETIME
				)`,
			} {
				if err := db.Exec(statement).Error; err != nil {
					t.Fatalf("create order test schema failed: %v", err)
				}
			}

			fixtures := []struct {
				id              string
				lastConnectedAt interface{}
				fallbackAt      string
			}{
				{id: "connected-new", lastConnectedAt: "2026-07-05T00:00:00Z", fallbackAt: "2026-07-01T00:00:00Z"},
				{id: "connected-tie-a", lastConnectedAt: "2026-07-04T00:00:00Z", fallbackAt: "2026-07-02T00:00:00Z"},
				{id: "connected-tie-b", lastConnectedAt: "2026-07-04T00:00:00Z", fallbackAt: "2026-07-03T00:00:00Z"},
				{id: "connected-old", lastConnectedAt: "2026-07-03T00:00:00Z", fallbackAt: "2026-07-04T00:00:00Z"},
				{id: "never-new", lastConnectedAt: nil, fallbackAt: "2026-07-25T00:00:00Z"},
				{id: "never-old", lastConnectedAt: nil, fallbackAt: "2026-07-20T00:00:00Z"},
			}
			for _, fixture := range fixtures {
				if err := db.Exec(
					`INSERT INTO devices (id, last_connected_at) VALUES (?, ?)`,
					fixture.id,
					fixture.lastConnectedAt,
				).Error; err != nil {
					t.Fatalf("insert device fixture failed: %v", err)
				}
				if err := db.Exec(
					`INSERT INTO relations (device_id, binding_time, added_at) VALUES (?, ?, ?)`,
					fixture.id,
					fixture.fallbackAt,
					fixture.fallbackAt,
				).Error; err != nil {
					t.Fatalf("insert relation fixture failed: %v", err)
				}
			}

			type orderedRow struct {
				ID string `gorm:"column:id"`
			}
			var rows []orderedRow
			sql := "SELECT d.id FROM devices AS d JOIN relations AS " + tt.joinAlias +
				" ON " + tt.joinAlias + ".device_id = d.id ORDER BY " + appDeviceOrderClause(tt.viewMode)
			if err := db.Raw(sql).Scan(&rows).Error; err != nil {
				t.Fatalf("execute order clause failed: %v", err)
			}

			want := []string{
				"connected-new",
				"connected-tie-a",
				"connected-tie-b",
				"connected-old",
				"never-new",
				"never-old",
			}
			if len(rows) != len(want) {
				t.Fatalf("ordered row count = %d, want %d", len(rows), len(want))
			}
			for index, expectedID := range want {
				if rows[index].ID != expectedID {
					t.Fatalf("ordered row %d = %q, want %q", index, rows[index].ID, expectedID)
				}
			}
		})
	}
}
