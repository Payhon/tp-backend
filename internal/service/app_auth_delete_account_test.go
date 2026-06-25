package service

import (
	"context"
	"testing"
	"time"

	"project/internal/dal"
	"project/internal/model"
	"project/internal/query"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupDeleteAccountTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	for _, sql := range []string{
		`CREATE TABLE users (
			id TEXT PRIMARY KEY,
			name TEXT,
			username TEXT,
			phone_number TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			status TEXT,
			authority TEXT,
			password TEXT NOT NULL DEFAULT '',
			tenant_id TEXT,
			dealer_id TEXT,
			org_id TEXT,
			user_kind TEXT,
			is_main INTEGER DEFAULT 0,
			remark TEXT,
			additional_info TEXT DEFAULT '{}',
			created_at DATETIME,
			updated_at DATETIME,
			password_last_updated DATETIME,
			last_visit_time DATETIME,
			last_visit_ip TEXT,
			last_visit_device TEXT,
			organization TEXT,
			timezone TEXT,
			default_language TEXT,
			password_fail_count INTEGER DEFAULT 0,
			avatar_url TEXT
		)`,
		`CREATE TABLE user_identities (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			identity_type TEXT NOT NULL,
			identifier TEXT NOT NULL,
			credential_type TEXT NOT NULL,
			password_hash TEXT,
			verified_at DATETIME,
			is_primary BOOLEAN DEFAULT FALSE,
			status TEXT NOT NULL,
			extra TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE pack_wxmp_configs (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			org_id TEXT NOT NULL,
			app_id TEXT NOT NULL,
			wx_appid TEXT NOT NULL,
			app_secret TEXT NOT NULL,
			status TEXT NOT NULL,
			home_banner_url TEXT,
			login_logo_url TEXT,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE app_device_added_records (id TEXT PRIMARY KEY, tenant_id TEXT, user_id TEXT)`,
		`CREATE TABLE device_user_bindings (id TEXT PRIMARY KEY, user_id TEXT)`,
		`CREATE TABLE message_push_manage (id TEXT PRIMARY KEY, user_id TEXT)`,
		`CREATE TABLE message_push_log (id TEXT PRIMARY KEY, user_id TEXT)`,
		`CREATE TABLE user_roles (id TEXT PRIMARY KEY, user_id TEXT)`,
		`CREATE TABLE user_address (id TEXT PRIMARY KEY, user_id TEXT)`,
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("create test schema failed: %v", err)
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

func insertDeleteAccountUser(t *testing.T, db *gorm.DB, tenantID, userID, password string) {
	t.Helper()
	now := time.Now().UTC()
	if err := db.Exec(
		`INSERT INTO users (id, tenant_id, user_kind, status, password, created_at, updated_at)
		 VALUES (?, ?, ?, 'N', ?, ?, ?)`,
		userID,
		tenantID,
		model.UserKindEndUser,
		utils.BcryptHash(password),
		now,
		now,
	).Error; err != nil {
		t.Fatalf("insert user failed: %v", err)
	}
}

func insertDeleteAccountPackWxmpConfig(t *testing.T, db *gorm.DB, tenantID, wxAppID, status string) {
	t.Helper()
	now := time.Now().UTC()
	if err := db.Exec(
		`INSERT INTO pack_wxmp_configs (
			id, tenant_id, org_id, app_id, wx_appid, app_secret, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 'secret', ?, ?, ?)`,
		"pack-cfg-"+wxAppID,
		tenantID,
		"pack-org-1",
		"app-"+wxAppID,
		wxAppID,
		status,
		now,
		now,
	).Error; err != nil {
		t.Fatalf("insert pack wxmp config failed: %v", err)
	}
}

func insertDeleteAccountWxmpIdentity(t *testing.T, db *gorm.DB, tenantID, userID, identifier, status string) {
	t.Helper()
	now := time.Now().UTC()
	if err := db.Exec(
		`INSERT INTO user_identities (
			id, user_id, tenant_id, identity_type, identifier, credential_type,
			verified_at, is_primary, status, extra, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, TRUE, ?, '{}', ?, ?)`,
		"identity-"+identifier,
		userID,
		tenantID,
		dal.IdentityTypeWxmpOpenID,
		identifier,
		dal.CredentialTypeCode,
		now,
		status,
		now,
		now,
	).Error; err != nil {
		t.Fatalf("insert wxmp identity failed: %v", err)
	}
}

func assertDeleteAccountUserExists(t *testing.T, db *gorm.DB, userID string, wantExists bool) {
	t.Helper()
	var count int64
	if err := db.Table("users").Where("id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count user failed: %v", err)
	}
	if gotExists := count > 0; gotExists != wantExists {
		t.Fatalf("user exists mismatch: got=%v want=%v", gotExists, wantExists)
	}
}

func TestAppAuthDeleteAccountRequiresPasswordForNormalUser(t *testing.T) {
	db := setupDeleteAccountTestDB(t)
	insertDeleteAccountUser(t, db, "tenant-1", "user-1", "CorrectPass1")

	err := (&AppAuth{}).DeleteAccount(context.Background(), "tenant-1", "user-1", "", "")
	if err == nil {
		t.Fatal("expected empty password to fail for normal user")
	}
	assertDeleteAccountUserExists(t, db, "user-1", true)
}

func TestAppAuthDeleteAccountAllowsPasswordForNormalUser(t *testing.T) {
	db := setupDeleteAccountTestDB(t)
	insertDeleteAccountUser(t, db, "tenant-1", "user-1", "CorrectPass1")

	err := (&AppAuth{}).DeleteAccount(context.Background(), "tenant-1", "user-1", "CorrectPass1", "")
	if err != nil {
		t.Fatalf("expected correct password to delete account, got %v", err)
	}
	assertDeleteAccountUserExists(t, db, "user-1", false)
}

func TestAppAuthDeleteAccountSkipsPasswordForBoundPackWxmp(t *testing.T) {
	db := setupDeleteAccountTestDB(t)
	insertDeleteAccountUser(t, db, "tenant-1", "user-1", "CorrectPass1")
	insertDeleteAccountPackWxmpConfig(t, db, "tenant-1", "wx-pack-1", "OPEN")
	insertDeleteAccountWxmpIdentity(t, db, "tenant-1", "user-1", "wx-pack-1:openid-1", "ACTIVE")

	err := (&AppAuth{}).DeleteAccount(context.Background(), "tenant-1", "user-1", "", "wx-pack-1")
	if err != nil {
		t.Fatalf("expected bound PACK wxmp account to delete without password, got %v", err)
	}
	assertDeleteAccountUserExists(t, db, "user-1", false)
}

func TestAppAuthDeleteAccountDoesNotSkipPasswordForUntrustedPackWxmp(t *testing.T) {
	cases := []struct {
		name              string
		configAppID       string
		configStatus      string
		requestAppID      string
		identityAppID     string
		identityStatus    string
		shouldAddConfig   bool
		shouldAddIdentity bool
	}{
		{
			name:              "unknown appid",
			requestAppID:      "wx-pack-1",
			shouldAddConfig:   false,
			shouldAddIdentity: false,
		},
		{
			name:              "appid not bound to user",
			configAppID:       "wx-pack-1",
			configStatus:      "OPEN",
			requestAppID:      "wx-pack-1",
			identityAppID:     "wx-pack-2",
			identityStatus:    "ACTIVE",
			shouldAddConfig:   true,
			shouldAddIdentity: true,
		},
		{
			name:              "disabled pack config",
			configAppID:       "wx-pack-1",
			configStatus:      "CLOSED",
			requestAppID:      "wx-pack-1",
			identityAppID:     "wx-pack-1",
			identityStatus:    "ACTIVE",
			shouldAddConfig:   true,
			shouldAddIdentity: true,
		},
		{
			name:              "inactive wxmp identity",
			configAppID:       "wx-pack-1",
			configStatus:      "OPEN",
			requestAppID:      "wx-pack-1",
			identityAppID:     "wx-pack-1",
			identityStatus:    "INACTIVE",
			shouldAddConfig:   true,
			shouldAddIdentity: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupDeleteAccountTestDB(t)
			insertDeleteAccountUser(t, db, "tenant-1", "user-1", "CorrectPass1")
			if tc.shouldAddConfig {
				insertDeleteAccountPackWxmpConfig(t, db, "tenant-1", tc.configAppID, tc.configStatus)
			}
			if tc.shouldAddIdentity {
				insertDeleteAccountWxmpIdentity(t, db, "tenant-1", "user-1", tc.identityAppID+":openid-1", tc.identityStatus)
			}

			err := (&AppAuth{}).DeleteAccount(context.Background(), "tenant-1", "user-1", "", tc.requestAppID)
			if err == nil {
				t.Fatal("expected untrusted PACK wxmp request without password to fail")
			}
			assertDeleteAccountUserExists(t, db, "user-1", true)
		})
	}
}
