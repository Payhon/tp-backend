package service

import (
	"context"
	"testing"
	"time"

	"project/internal/model"
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
		`CREATE TABLE device_configs (
			id text primary key,
			tenant_id text,
			name text
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
			activation_status text,
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
			device_config_id text,
			voltage_rated real,
			capacity_rated real,
			cell_count integer,
			nominal_power real,
			warranty_months integer,
			description text,
			tenant_id text,
			created_at datetime,
			updated_at datetime
		)`,
		`CREATE TABLE battery_warranty_recalc_jobs (
			id text primary key,
			tenant_id text,
			operator_id text,
			source text,
			scope_model_id text,
			status text,
			total_rows integer default 0,
			processed_rows integer default 0,
			success_rows integer default 0,
			skipped_rows integer default 0,
			failed_rows integer default 0,
			error_message text,
			started_at datetime,
			finished_at datetime,
			created_at datetime,
			updated_at datetime
		)`,
		`CREATE TABLE battery_warranty_recalc_job_logs (
			id integer primary key autoincrement,
			job_id text,
			tenant_id text,
			level text,
			device_id text,
			device_number text,
			battery_model_id text,
			message text,
			created_at datetime
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

func warrantyStringPtr(s string) *string {
	return &s
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
	if resp.WarrantyProfileExists {
		t.Fatalf("expected no saved warranty profile")
	}
	if resp.ContactName != nil || resp.ContactPhone != nil {
		t.Fatalf("expected no account contact fallback, got name=%+v phone=%+v", resp.ContactName, resp.ContactPhone)
	}
}

func TestAppWarrantyProfileReminderState(t *testing.T) {
	tests := []struct {
		name               string
		withBinding        bool
		contactName        *string
		contactPhone       *string
		wantCompleted      bool
		wantProfileExists  bool
		wantReminderNeeded bool
	}{
		{
			name:              "no bound device does not need a reminder",
			contactName:       warrantyStringPtr("张三"),
			contactPhone:      warrantyStringPtr("13800000000"),
			wantCompleted:     true,
			wantProfileExists: true,
		},
		{
			name:               "account fallback is not a completed warranty profile",
			withBinding:        true,
			wantReminderNeeded: true,
		},
		{
			name:               "partial warranty profile remains incomplete",
			withBinding:        true,
			contactName:        warrantyStringPtr("张三"),
			wantProfileExists:  true,
			wantReminderNeeded: true,
		},
		{
			name:               "full warranty profile completes reminder",
			withBinding:        true,
			contactName:        warrantyStringPtr("张三"),
			contactPhone:       warrantyStringPtr("13800000000"),
			wantCompleted:      true,
			wantProfileExists:  true,
			wantReminderNeeded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupUserWarrantyTestDB(t)
			if err := db.Exec(`INSERT INTO users (id, tenant_id, user_kind, name, phone_number) VALUES (?, ?, ?, ?, ?)`, "user-1", "tenant-1", "END_USER", "账号姓名", "13900000000").Error; err != nil {
				t.Fatalf("insert user failed: %v", err)
			}
			if tt.withBinding {
				if err := db.Exec(`INSERT INTO devices (id, device_number, tenant_id) VALUES (?, ?, ?)`, "dev-1", "SN001", "tenant-1").Error; err != nil {
					t.Fatalf("insert device failed: %v", err)
				}
				if err := db.Exec(`INSERT INTO device_user_bindings (id, user_id, device_id, binding_time, is_owner) VALUES (?, ?, ?, ?, ?)`, "bind-1", "user-1", "dev-1", time.Now().UTC(), true).Error; err != nil {
					t.Fatalf("insert binding failed: %v", err)
				}
			}
			if tt.contactName != nil || tt.contactPhone != nil {
				if err := db.Exec(`INSERT INTO user_warranty_infos (id, tenant_id, user_id, contact_name, contact_phone, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, "warranty-1", "tenant-1", "user-1", tt.contactName, tt.contactPhone, time.Now().UTC(), time.Now().UTC()).Error; err != nil {
					t.Fatalf("insert warranty profile failed: %v", err)
				}
			}

			resp, err := (&UserWarrantyInfo{}).GetProfile(context.Background(), &utils.UserClaims{
				ID:       "user-1",
				TenantID: "tenant-1",
				UserKind: model.UserKindEndUser,
			}, "")
			if err != nil {
				t.Fatalf("GetProfile() error = %v", err)
			}
			if resp.WarrantyProfileCompleted != tt.wantCompleted {
				t.Fatalf("WarrantyProfileCompleted = %v, want %v", resp.WarrantyProfileCompleted, tt.wantCompleted)
			}
			if resp.WarrantyProfileExists != tt.wantProfileExists {
				t.Fatalf("WarrantyProfileExists = %v, want %v", resp.WarrantyProfileExists, tt.wantProfileExists)
			}
			if resp.WarrantyProfileReminderNeeded != tt.wantReminderNeeded {
				t.Fatalf("WarrantyProfileReminderNeeded = %v, want %v", resp.WarrantyProfileReminderNeeded, tt.wantReminderNeeded)
			}
		})
	}
}

func TestApplyBatteryWarrantyActivationTxCalculatesExpiry(t *testing.T) {
	db := setupUserWarrantyTestDB(t)
	if err := db.Exec(`INSERT INTO battery_bms_models (id, name, tenant_id, warranty_months) VALUES (?, ?, ?, ?)`, "model-1", "BMS-A", "tenant-1", 18).Error; err != nil {
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
	if err := db.Exec(`INSERT INTO battery_bms_models (id, name, tenant_id, warranty_months) VALUES (?, ?, ?, ?)`, "model-1", "BMS-A", "tenant-1", 18).Error; err != nil {
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

func TestUpdateBatteryBmsModelStartsWarrantyRecalcJob(t *testing.T) {
	db := setupUserWarrantyTestDB(t)
	now := time.Now().UTC()
	if err := db.Exec(`INSERT INTO device_configs (id, tenant_id, name) VALUES (?, ?, ?)`, "cfg-1", "tenant-1", "BMS模板").Error; err != nil {
		t.Fatalf("insert config failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO battery_bms_models (id, name, device_config_id, tenant_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, "model-1", "BMS-A", "cfg-1", "tenant-1", now, now).Error; err != nil {
		t.Fatalf("insert model failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO devices (id, device_number, tenant_id) VALUES (?, ?, ?)`, "dev-1", "SN001", "tenant-1").Error; err != nil {
		t.Fatalf("insert device failed: %v", err)
	}
	activationAt := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, activation_status, activation_date) VALUES (?, ?, ?, ?)`, "dev-1", "model-1", "ACTIVE", activationAt).Error; err != nil {
		t.Fatalf("insert battery failed: %v", err)
	}

	months := int32(18)
	resp, err := (&BatteryBmsModel{}).UpdateBatteryBmsModel("model-1", model.BatteryBmsModelUpdateReq{WarrantyMonths: &months}, &utils.UserClaims{
		ID:       "op-1",
		TenantID: "tenant-1",
	})
	if err != nil {
		t.Fatalf("UpdateBatteryBmsModel() error = %v", err)
	}
	if resp.WarrantyRecalcJobID == nil || *resp.WarrantyRecalcJobID == "" {
		t.Fatalf("expected warranty recalc job id, got %+v", resp.WarrantyRecalcJobID)
	}
	waitForWarrantyRecalcJob(t, db, *resp.WarrantyRecalcJobID)

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

func TestBatteryWarrantyRecalcModelChangeOverwritesAutoButKeepsManualOverride(t *testing.T) {
	db := setupUserWarrantyTestDB(t)
	if err := db.Exec(`INSERT INTO battery_bms_models (id, name, tenant_id, warranty_months) VALUES (?, ?, ?, ?)`, "model-1", "BMS-A", "tenant-1", 24).Error; err != nil {
		t.Fatalf("insert model failed: %v", err)
	}
	for _, row := range []struct {
		id     string
		number string
	}{
		{"dev-auto", "SN-AUTO"},
		{"dev-manual", "SN-MANUAL"},
		{"dev-inactive", "SN-INACTIVE"},
	} {
		if err := db.Exec(`INSERT INTO devices (id, device_number, tenant_id) VALUES (?, ?, ?)`, row.id, row.number, "tenant-1").Error; err != nil {
			t.Fatalf("insert device failed: %v", err)
		}
	}
	autoActivation := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	manualActivation := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	oldAutoExpire := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	manualExpire := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, activation_status, activation_date, warranty_months, warranty_expire_date, warranty_manual_override) VALUES (?, ?, ?, ?, ?, ?, ?)`, "dev-auto", "model-1", "ACTIVE", autoActivation, 12, oldAutoExpire, false).Error; err != nil {
		t.Fatalf("insert auto battery failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, activation_status, activation_date, warranty_months, warranty_expire_date, warranty_manual_override) VALUES (?, ?, ?, ?, ?, ?, ?)`, "dev-manual", "model-1", "ACTIVE", manualActivation, 12, manualExpire, true).Error; err != nil {
		t.Fatalf("insert manual battery failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, activation_status) VALUES (?, ?, ?)`, "dev-inactive", "model-1", "INACTIVE").Error; err != nil {
		t.Fatalf("insert inactive battery failed: %v", err)
	}

	modelID := "model-1"
	jobID, err := createBatteryWarrantyRecalcJobTx(context.Background(), db, "tenant-1", "op-1", batteryWarrantyRecalcSourceModelChange, &modelID)
	if err != nil {
		t.Fatalf("create job failed: %v", err)
	}
	if err := runBatteryWarrantyRecalcJob(context.Background(), jobID); err != nil {
		t.Fatalf("run job failed: %v", err)
	}

	var autoGot struct {
		WarrantyMonths     int       `gorm:"column:warranty_months"`
		WarrantyStartDate  time.Time `gorm:"column:warranty_start_date"`
		WarrantyExpireDate time.Time `gorm:"column:warranty_expire_date"`
	}
	if err := db.Table("device_batteries").Select("warranty_months, warranty_start_date, warranty_expire_date").Where("device_id = ?", "dev-auto").Scan(&autoGot).Error; err != nil {
		t.Fatalf("query auto battery failed: %v", err)
	}
	if autoGot.WarrantyMonths != 24 {
		t.Fatalf("auto warranty_months = %d, want 24", autoGot.WarrantyMonths)
	}
	if autoGot.WarrantyStartDate.Format("2006-01-02") != "2026-01-01" {
		t.Fatalf("auto warranty_start_date = %s", autoGot.WarrantyStartDate.Format("2006-01-02"))
	}
	if autoGot.WarrantyExpireDate.Format("2006-01-02") != "2028-01-01" {
		t.Fatalf("auto warranty_expire_date = %s", autoGot.WarrantyExpireDate.Format("2006-01-02"))
	}

	var manualGot struct {
		WarrantyMonths     int       `gorm:"column:warranty_months"`
		WarrantyExpireDate time.Time `gorm:"column:warranty_expire_date"`
	}
	if err := db.Table("device_batteries").Select("warranty_months, warranty_expire_date").Where("device_id = ?", "dev-manual").Scan(&manualGot).Error; err != nil {
		t.Fatalf("query manual battery failed: %v", err)
	}
	if manualGot.WarrantyMonths != 12 {
		t.Fatalf("manual warranty_months = %d, want 12", manualGot.WarrantyMonths)
	}
	if manualGot.WarrantyExpireDate.Format("2006-01-02") != "2030-01-01" {
		t.Fatalf("manual warranty_expire_date = %s, want 2030-01-01", manualGot.WarrantyExpireDate.Format("2006-01-02"))
	}

	var job batteryWarrantyRecalcJob
	if err := db.Table("battery_warranty_recalc_jobs").Where("id = ?", jobID).Scan(&job).Error; err != nil {
		t.Fatalf("query job failed: %v", err)
	}
	if job.Status != batteryWarrantyRecalcStatusSuccess || job.SuccessRows != 1 || job.SkippedRows != 2 || job.FailedRows != 0 {
		t.Fatalf("unexpected job result: %+v", job)
	}
}

func TestBatteryWarrantyRecalcManualScanOnlyProcessesMissingExpiry(t *testing.T) {
	db := setupUserWarrantyTestDB(t)
	if err := db.Exec(`INSERT INTO battery_bms_models (id, name, tenant_id, warranty_months) VALUES (?, ?, ?, ?)`, "model-1", "BMS-A", "tenant-1", 18).Error; err != nil {
		t.Fatalf("insert model failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO battery_bms_models (id, name, tenant_id, warranty_months) VALUES (?, ?, ?, ?)`, "model-empty", "BMS-empty", "tenant-1", 0).Error; err != nil {
		t.Fatalf("insert empty model failed: %v", err)
	}
	for _, row := range []struct {
		id     string
		number string
	}{
		{"dev-missing", "SN-MISSING"},
		{"dev-existing", "SN-EXISTING"},
		{"dev-no-date", "SN-NODATE"},
		{"dev-no-model", "SN-NOMODEL"},
		{"dev-no-month", "SN-NOMONTH"},
	} {
		if err := db.Exec(`INSERT INTO devices (id, device_number, tenant_id) VALUES (?, ?, ?)`, row.id, row.number, "tenant-1").Error; err != nil {
			t.Fatalf("insert device failed: %v", err)
		}
	}
	activationAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	existingExpire := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, activation_status, activation_date) VALUES (?, ?, ?, ?)`, "dev-missing", "model-1", "ACTIVE", activationAt).Error; err != nil {
		t.Fatalf("insert missing battery failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, activation_status, activation_date, warranty_expire_date) VALUES (?, ?, ?, ?, ?)`, "dev-existing", "model-1", "ACTIVE", activationAt, existingExpire).Error; err != nil {
		t.Fatalf("insert existing battery failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, activation_status) VALUES (?, ?, ?)`, "dev-no-date", "model-1", "ACTIVE").Error; err != nil {
		t.Fatalf("insert no-date battery failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, activation_status, activation_date) VALUES (?, ?, ?)`, "dev-no-model", "ACTIVE", activationAt).Error; err != nil {
		t.Fatalf("insert no-model battery failed: %v", err)
	}
	if err := db.Exec(`INSERT INTO device_batteries (device_id, battery_model_id, activation_status, activation_date) VALUES (?, ?, ?, ?)`, "dev-no-month", "model-empty", "ACTIVE", activationAt).Error; err != nil {
		t.Fatalf("insert no-month battery failed: %v", err)
	}

	jobID, err := createBatteryWarrantyRecalcJobTx(context.Background(), db, "tenant-1", "op-1", batteryWarrantyRecalcSourceManualScan, nil)
	if err != nil {
		t.Fatalf("create job failed: %v", err)
	}
	if err := runBatteryWarrantyRecalcJob(context.Background(), jobID); err != nil {
		t.Fatalf("run job failed: %v", err)
	}

	var missingExpire time.Time
	if err := db.Table("device_batteries").Select("warranty_expire_date").Where("device_id = ?", "dev-missing").Scan(&missingExpire).Error; err != nil {
		t.Fatalf("query missing battery failed: %v", err)
	}
	if missingExpire.Format("2006-01-02") != "2027-09-01" {
		t.Fatalf("missing warranty_expire_date = %s, want 2027-09-01", missingExpire.Format("2006-01-02"))
	}
	var existingGot time.Time
	if err := db.Table("device_batteries").Select("warranty_expire_date").Where("device_id = ?", "dev-existing").Scan(&existingGot).Error; err != nil {
		t.Fatalf("query existing battery failed: %v", err)
	}
	if existingGot.Format("2006-01-02") != "2027-03-01" {
		t.Fatalf("existing warranty_expire_date = %s, want 2027-03-01", existingGot.Format("2006-01-02"))
	}
	var job batteryWarrantyRecalcJob
	if err := db.Table("battery_warranty_recalc_jobs").Where("id = ?", jobID).Scan(&job).Error; err != nil {
		t.Fatalf("query job failed: %v", err)
	}
	if job.TotalRows != 4 || job.SuccessRows != 1 || job.SkippedRows != 3 || job.FailedRows != 0 {
		t.Fatalf("unexpected job result: %+v", job)
	}

	claims := &utils.UserClaims{ID: "op-1", TenantID: "tenant-1"}
	status, err := (&UserWarrantyInfo{}).GetBatteryWarrantyRecalcJobStatus(context.Background(), jobID, claims)
	if err != nil {
		t.Fatalf("GetBatteryWarrantyRecalcJobStatus() error = %v", err)
	}
	if status.JobID != jobID || status.SuccessRows != 1 || status.SkippedRows != 3 {
		t.Fatalf("unexpected status: %+v", status)
	}
	logs, err := (&UserWarrantyInfo{}).GetBatteryWarrantyRecalcJobLogs(context.Background(), jobID, 0, 20, claims)
	if err != nil {
		t.Fatalf("GetBatteryWarrantyRecalcJobLogs() error = %v", err)
	}
	if len(logs.List) != 4 || logs.NextAfterID == 0 {
		t.Fatalf("unexpected logs: %+v", logs)
	}
	if _, err := (&UserWarrantyInfo{}).GetBatteryWarrantyRecalcJobStatus(context.Background(), jobID, &utils.UserClaims{ID: "op-2", TenantID: "tenant-2"}); err == nil {
		t.Fatalf("expected cross-tenant status query to fail")
	}
}

func waitForWarrantyRecalcJob(t *testing.T, db *gorm.DB, jobID string) {
	t.Helper()
	for i := 0; i < 100; i++ {
		var job batteryWarrantyRecalcJob
		if err := db.Table("battery_warranty_recalc_jobs").Where("id = ?", jobID).Scan(&job).Error; err != nil {
			t.Fatalf("query job failed: %v", err)
		}
		if job.Status == batteryWarrantyRecalcStatusSuccess {
			return
		}
		if job.Status == batteryWarrantyRecalcStatusFailed {
			t.Fatalf("job failed: %+v", job)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("job %s did not finish", jobID)
}
