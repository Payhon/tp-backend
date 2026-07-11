package storage

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func setupTelemetryWriterTest(t *testing.T) (*telemetryWriter, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&TelemetryData{}, &TelemetryCurrentData{}); err != nil {
		t.Fatalf("migrate telemetry tables: %v", err)
	}
	writer := newTelemetryWriter(db, logrus.New(), Config{}, newMetricsCollector())
	return writer, db
}

func numberPointer(value float64) *float64 {
	return &value
}

func TestTelemetryCurrentUpsertDoesNotRegressTimestamp(t *testing.T) {
	writer, db := setupTelemetryWriterTest(t)
	newerAt := time.UnixMilli(2_000)
	newer := TelemetryCurrentData{
		DeviceID: "device-a",
		Key:      "soc",
		TS:       newerAt,
		NumberV:  numberPointer(90),
		TenantID: "tenant-a",
	}
	if err := db.Create(&newer).Error; err != nil {
		t.Fatalf("seed newer current value: %v", err)
	}

	olderCurrent := TelemetryCurrentData{
		DeviceID: "device-a",
		Key:      "soc",
		TS:       time.UnixMilli(1_000),
		NumberV:  numberPointer(10),
		TenantID: "tenant-a",
	}
	olderHistory := TelemetryData{
		DeviceID: "device-a",
		Key:      "soc",
		TS:       1_000,
		NumberV:  numberPointer(10),
		TenantID: "tenant-a",
	}
	if _, failed := writer.batchInsert([]TelemetryData{olderHistory}, []TelemetryCurrentData{olderCurrent}); failed != 0 {
		t.Fatalf("older batch failed=%d, want 0", failed)
	}

	var got TelemetryCurrentData
	if err := db.Where("device_id = ? AND key = ?", "device-a", "soc").First(&got).Error; err != nil {
		t.Fatalf("read current value: %v", err)
	}
	if !got.TS.Equal(newerAt) {
		t.Fatalf("current timestamp = %v, want %v", got.TS, newerAt)
	}
	if got.NumberV == nil || *got.NumberV != 90 {
		t.Fatalf("current value = %#v, want 90", got.NumberV)
	}
}

func TestTelemetryFallbackHandlesMultipleHistoryRowsPerCurrentKey(t *testing.T) {
	writer, db := setupTelemetryWriterTest(t)
	history := []TelemetryData{
		{DeviceID: "device-a", Key: "soc", TS: 1_000, NumberV: numberPointer(10), TenantID: "tenant-a"},
		{DeviceID: "device-a", Key: "soc", TS: 2_000, NumberV: numberPointer(20), TenantID: "tenant-a"},
	}
	current := []TelemetryCurrentData{
		{DeviceID: "device-a", Key: "soc", TS: time.UnixMilli(2_000), NumberV: numberPointer(20), TenantID: "tenant-a"},
	}

	written, failed := writer.fallbackInsert(history, current)
	if written != len(history) || failed != 0 {
		t.Fatalf("fallback result written=%d failed=%d, want written=%d failed=0", written, failed, len(history))
	}

	var got TelemetryCurrentData
	if err := db.Where("device_id = ? AND key = ?", "device-a", "soc").First(&got).Error; err != nil {
		t.Fatalf("read current value: %v", err)
	}
	if !got.TS.Equal(time.UnixMilli(2_000)) || got.NumberV == nil || *got.NumberV != 20 {
		t.Fatalf("fallback current value = %#v at %v, want 20 at 2000ms", got.NumberV, got.TS)
	}
}
