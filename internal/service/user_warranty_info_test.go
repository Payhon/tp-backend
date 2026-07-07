package service

import (
	"context"
	"testing"
	"time"

	"project/pkg/global"
	"project/pkg/utils"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupUserWarrantyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	for _, sql := range []string{
		`CREATE TABLE users (
			id text primary key,
			tenant_id text,
			user_kind text,
			name text,
			username text,
			phone_number text
		)`,
		`CREATE TABLE user_warranty_infos (
			id text primary key,
			tenant_id text,
			user_id text,
			contact_name text,
			contact_phone text,
			created_at datetime,
			updated_at datetime,
			UNIQUE(tenant_id, user_id)
		)`,
		`CREATE TABLE devices (
			id text primary key,
			device_number text,
			tenant_id text
		)`,
		`CREATE TABLE device_user_bindings (
			id text primary key,
			user_id text,
			device_id text,
			binding_time datetime,
			is_owner boolean
		)`,
		`CREATE TABLE device_batteries (
			device_id text primary key,
			battery_model_id text,
			item_uuid text,
			activation_date datetime,
			warranty_months integer,
			warranty_start_date datetime,
			warranty_expire_date datetime,
			warranty_manual_override boolean default false,
			warranty_updated_at datetime,
			warranty_updated_by text,
			updated_at datetime
		)`,
		`CREATE TABLE battery_models (
			id text primary key,
			name text
		)`,
		`CREATE TABLE battery_bms_models (
			id text primary key,
			name text,
			warranty_months integer
		)`,
		`CREATE TABLE pack_wxmp_configs (
			id text primary key,
			tenant_id text,
			org_id text,
			app_id text,
			wx_appid text,
			app_secret text,
			status text,
			home_banner_url text,
			login_logo_url text,
			warranty_cards_enabled boolean default true,
			remark text,
			created_at datetime,
			updated_at datetime
		)`,
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
	return db
}

func TestAppWarrantyProfileHidesBatteryCardsForDisabledPackWxmp(t *testing.T) {
	db := setupUserWarrantyTestDB(t)
	if err := db.Exec(`INSERT INTO users (id, tenant_id, user_kind, name, phone_number) VALUES (?, ?, ?, ?, ?)`, "user-1", "tenant-1", "END_USER", "张三", "13800000000").Error; err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO devices (id, device_number, tenant_id) VALUES (?, ?, ?)`, "dev-1", "SN001", "tenant-1").Error; err != nil {
		t.Fatalf("insert device failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO device_user_bindings (id, user_id, device_id, binding_time, is_owner) VALUES (?, ?, ?, ?, ?)`, "bind-1", "user-1", "dev-1", time.Now().UTC(), true).Error; err != nil {
		t.Fatalf("insert binding failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, item_uuid, warranty_months) VALUES (?, ?, ?)`, "dev-1", "BAT001", 12).Error; err != nil {
		t.Fatalf("insert battery failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO pack_wxmp_configs (id, tenant_id, org_id, app_id, wx_appid, app_secret, status, warranty_cards_enabled) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "cfg-1", "tenant-1", "pack-1", "app-1", "wx-pack", "secret", "OPEN", false).Error; err != nil {
		t.Fatalf("insert pack config failed: %v", err)
	}

	resp, err := (&UserWarrantyInfo{}).GetProfile(context.Background(), &utils.UserClaims{
		ID:       "user-1",
		TenantID: "tenant-1",
		UserKind: "END_USER",
	}, "wx-pack")
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if resp.WarrantyCardsEnabled {
		t.Fatalf("expected warranty cards disabled")
	}
	if len(resp.Batteries) != 0 {
		t.Fatalf("expected no battery cards, got %+v", resp.Batteries)
	}
	if resp.ContactName == nil || *resp.ContactName != "张三" {
		t.Fatalf("expected profile fallback name, got %+v", resp.ContactName)
	}
}

func TestApplyBatteryWarrantyActivationTxCalculatesExpiry(t *testing.T) {
	db := setupUserWarrantyTestDB(t)
	if err := db.Exec(`INSERT INTO battery_bms_models (id, name, warranty_months) VALUES (?, ?, ?)`, "model-1", "BMS-A", 18).Error; err != nil {
		t.Fatalf("insert model failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id) VALUES (?, ?)`, "dev-1", "model-1").Error; err != nil {
		t.Fatalf("insert battery failed: %v", err)
	}
	activationAt := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	if err := applyBatteryWarrantyActivationTx(context.Background(), db, "dev-1", "op-1", activationAt); err != nil {
		t.Fatalf("applyBatteryWarrantyActivationTx() error = %v", err)
	}

	var got struct {
		WarrantyMonths     int       `gorm:"column:warranty_months"`
		WarrantyStartDate  time.Time `gorm:"column:warranty_start_date"`
		WarrantyExpireDate time.Time `gorm:"column:warranty_expire_date"`
	}
	if err := db.Table("device_batteries").Select("warranty_months, warranty_start_date, warranty_expire_date").Where("device_id = ?", "dev-1").Scan(&got).Error; err != nil {
		t.Fatalf("query battery failed: %v", err)
	}
	if got.WarrantyMonths != 18 {
		t.Fatalf("warranty_months = %d, want 18", got.WarrantyMonths)
	}
	if got.WarrantyStartDate.Format("2006-01-02") != "2026-07-01" {
		t.Fatalf("warranty_start_date = %s", got.WarrantyStartDate.Format("2006-01-02"))
	}
	if got.WarrantyExpireDate.Format("2006-01-02") != "2028-01-01" {
		t.Fatalf("warranty_expire_date = %s", got.WarrantyExpireDate.Format("2006-01-02"))
	}
}

func TestApplyBatteryWarrantyActivationTxKeepsManualOverrideExpiry(t *testing.T) {
	db := setupUserWarrantyTestDB(t)
	if err := db.Exec(`INSERT INTO battery_bms_models (id, name, warranty_months) VALUES (?, ?, ?)`, "model-1", "BMS-A", 18).Error; err != nil {
		t.Fatalf("insert model failed: %v", err)
	}
	manualExpire := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, warranty_expire_date, warranty_manual_override) VALUES (?, ?, ?, ?)`, "dev-1", "model-1", manualExpire, true).Error; err != nil {
		t.Fatalf("insert battery failed: %v", err)
	}
	activationAt := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	if err := applyBatteryWarrantyActivationTx(context.Background(), db, "dev-1", "op-1", activationAt); err != nil {
		t.Fatalf("applyBatteryWarrantyActivationTx() error = %v", err)
	}

	var got time.Time
	if err := db.Table("device_batteries").Select("warranty_expire_date").Where("device_id = ?", "dev-1").Scan(&got).Error; err != nil {
		t.Fatalf("query battery failed: %v", err)
	}
	if got.Format("2006-01-02") != "2030-01-01" {
		t.Fatalf("warranty_expire_date = %s, want 2030-01-01", got.Format("2006-01-02"))
	}
}
