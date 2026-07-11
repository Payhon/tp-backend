package bmsbridge

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestShouldIgnoreRetainedUplink(t *testing.T) {
	if !shouldIgnoreRetainedUplink(true) {
		t.Fatal("retained bridge uplink must be ignored")
	}
	if shouldIgnoreRetainedUplink(false) {
		t.Fatal("live bridge uplink must be processed")
	}
}

func TestIncomingShardKeepsSameDeviceFIFO(t *testing.T) {
	bridge := &Bridge{queues: newIncomingShards(4, 16)}
	first := incoming{rawDeviceID: "topic-device-a", deviceID: "device-a", messageID: "1"}
	second := incoming{rawDeviceID: "topic-device-a", deviceID: "device-a-after-remap", messageID: "2"}
	if !bridge.enqueueIncoming(first) || !bridge.enqueueIncoming(second) {
		t.Fatal("expected both messages to be enqueued")
	}

	index := incomingShardIndex(first.rawDeviceID, len(bridge.queues))
	if got := (<-bridge.queues[index]).messageID; got != "1" {
		t.Fatalf("first dequeued message = %q, want 1", got)
	}
	if got := (<-bridge.queues[index]).messageID; got != "2" {
		t.Fatalf("second dequeued message = %q, want 2", got)
	}
}

func TestNextReceivedAtIsStrictlyIncreasingPerRawDevice(t *testing.T) {
	bridge := &Bridge{lastReceivedAtMs: make(map[string]int64)}
	observed := time.UnixMilli(1_720_000_000_123)
	first := bridge.nextReceivedAt("raw-device-a", observed)
	second := bridge.nextReceivedAt("raw-device-a", observed)
	third := bridge.nextReceivedAt("raw-device-a", observed.Add(-time.Second))
	other := bridge.nextReceivedAt("raw-device-b", observed)

	if first.UnixMilli() != observed.UnixMilli() || second.UnixMilli() != first.UnixMilli()+1 || third.UnixMilli() != second.UnixMilli()+1 {
		t.Fatalf("same-device timestamps = %d, %d, %d; want strictly increasing milliseconds", first.UnixMilli(), second.UnixMilli(), third.UnixMilli())
	}
	if other.UnixMilli() != observed.UnixMilli() {
		t.Fatalf("other device timestamp = %d, want independent %d", other.UnixMilli(), observed.UnixMilli())
	}
}

func TestIncomingShardMappingAllowsCrossDeviceParallelism(t *testing.T) {
	const shardCount = 8
	firstIndex := incomingShardIndex("device-a", shardCount)
	secondDevice := ""
	for i := 0; i < 100; i++ {
		candidate := string(rune('b' + i))
		if incomingShardIndex(candidate, shardCount) != firstIndex {
			secondDevice = candidate
			break
		}
	}
	if secondDevice == "" {
		t.Fatal("expected hash routing to use more than one shard")
	}
	if incomingShardIndex("device-a", shardCount) != firstIndex {
		t.Fatal("same device must always map to the same shard")
	}
}

func TestNewTelemetryPayloadCarriesBridgeReceiveTimestamp(t *testing.T) {
	receivedAt := time.UnixMilli(1_720_000_000_123)
	payload := newTelemetryPayload("device-a", map[string]any{"soc": 88}, receivedAt)
	if got := payload["source"]; got != "bms_bridge" {
		t.Fatalf("source = %#v, want bms_bridge", got)
	}
	if got := payload["timestamp"]; got != receivedAt.UnixMilli() {
		t.Fatalf("timestamp = %#v, want %d", got, receivedAt.UnixMilli())
	}
}

func setupBridgeResolveTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	stmts := []string{
		`CREATE TABLE devices (
			id varchar(36) PRIMARY KEY,
			device_number varchar(100),
			tenant_id varchar(36)
		)`,
		`CREATE TABLE device_batteries (
			device_id varchar(36) PRIMARY KEY,
			item_uuid varchar(64),
			comm_chip_id varchar(64),
			imei varchar(32),
			iccid varchar(32)
		)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}
	return db
}

func TestResolvePlatformDeviceIDRequeriesAfterSameSNRecreate(t *testing.T) {
	db := setupBridgeResolveTestDB(t)
	const sn = "36011161145053593437373030124A57"

	if err := db.Exec(`INSERT INTO devices (id, device_number, tenant_id) VALUES ('old-dev', ?, 'tenant-a')`, sn).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, item_uuid, comm_chip_id) VALUES ('old-dev', ?, 'comm-old')`, sn).Error; err != nil {
		t.Fatal(err)
	}

	bridge := &Bridge{db: db}
	if got := bridge.resolvePlatformDeviceID(context.Background(), sn); got != "old-dev" {
		t.Fatalf("first resolve = %q, want old-dev", got)
	}

	// Simulate deleting the device row in the admin backend and recreating a new
	// device with the same SN while an old battery extension row is left behind.
	if err := db.Exec(`DELETE FROM devices WHERE id = 'old-dev'`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO devices (id, device_number, tenant_id) VALUES ('new-dev', ?, 'tenant-a')`, sn).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, item_uuid, comm_chip_id) VALUES ('new-dev', ?, 'comm-new')`, sn).Error; err != nil {
		t.Fatal(err)
	}

	if got := bridge.resolvePlatformDeviceID(context.Background(), sn); got != "new-dev" {
		t.Fatalf("resolve after recreate = %q, want new-dev", got)
	}
}
