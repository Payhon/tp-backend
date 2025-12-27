package service

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDryRunPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Discard,
	})
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}
	return db.Session(&gorm.Session{DryRun: true})
}

func TestBuildLatestDeviceAlarmsBase_NoDevicesAliasD(t *testing.T) {
	db := newDryRunPostgresDB(t)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	var rows []map[string]interface{}
	res := buildLatestDeviceAlarmsBase(db, "tenant-1", "", start).
		Select("lda.alarm_status AS alarm_status, COUNT(1) AS cnt").
		Group("lda.alarm_status").
		Find(&rows)

	if res.Error != nil {
		t.Fatalf("build sql: %v", res.Error)
	}

	sql := res.Statement.SQL.String()
	if sql == "" {
		t.Fatalf("expected non-empty sql")
	}
	if strings.Contains(sql, "devices AS d") {
		t.Fatalf("unexpected devices alias d in sql: %s", sql)
	}
	if strings.Contains(sql, "org_dev") {
		t.Fatalf("unexpected org filter join in sql: %s", sql)
	}
}

func TestBuildLatestDeviceAlarmsBase_WithOrgFilter_NoDevicesAliasD(t *testing.T) {
	db := newDryRunPostgresDB(t)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	var rows []map[string]interface{}
	res := buildLatestDeviceAlarmsBase(db, "tenant-1", "org-1", start).
		Select("to_char(lda.create_at::date, 'YYYY-MM-DD') AS day, COUNT(1) AS cnt").
		Group("lda.create_at::date").
		Order("day ASC").
		Find(&rows)

	if res.Error != nil {
		t.Fatalf("build sql: %v", res.Error)
	}

	sql := res.Statement.SQL.String()
	if sql == "" {
		t.Fatalf("expected non-empty sql")
	}
	if strings.Contains(sql, "devices AS d") {
		t.Fatalf("unexpected devices alias d in sql: %s", sql)
	}
	if !strings.Contains(sql, "org_dev") {
		t.Fatalf("expected org filter join in sql: %s", sql)
	}
	if !strings.Contains(sql, "device_batteries AS dbat") {
		t.Fatalf("expected device_batteries in org filter subquery: %s", sql)
	}
}
