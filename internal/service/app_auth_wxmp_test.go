package service

import (
	"context"
	"testing"

	"project/internal/model"
	"project/pkg/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestWxmpIdentityIdentifier(t *testing.T) {
	if got := wxmpIdentityIdentifier("", "openid-1"); got != "openid-1" {
		t.Fatalf("legacy identifier mismatch: got=%q", got)
	}
	if got := wxmpIdentityIdentifier("wx-app-1", "openid-1"); got != "wx-app-1:openid-1" {
		t.Fatalf("scoped identifier mismatch: got=%q", got)
	}
}

func TestGetPackWxMpConfigMissingReturnsEmptyConfig(t *testing.T) {
	oldDB := global.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	global.DB = db
	t.Cleanup(func() {
		global.DB = oldDB
	})

	if err := db.Exec(`
		CREATE TABLE orgs (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			org_type TEXT NOT NULL
		);
		CREATE TABLE pack_wxmp_configs (
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
		);
	`).Error; err != nil {
		t.Fatalf("create schema failed: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO orgs (id, tenant_id, org_type) VALUES (?, ?, ?)`,
		"pack-1",
		"tenant-1",
		model.OrgTypePACKFactory,
	).Error; err != nil {
		t.Fatalf("insert org failed: %v", err)
	}

	resp, err := (&AppAuthConfig{}).GetPackWxMpConfig(context.Background(), "tenant-1", "pack-1")
	if err != nil {
		t.Fatalf("expected empty config without error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected empty config response, got nil")
	}
	if resp.TenantID != "tenant-1" || resp.OrgID != "pack-1" || resp.Status != "OPEN" {
		t.Fatalf("unexpected empty config response: %+v", resp)
	}
	if resp.ID != "" || resp.WxAppID != "" || resp.AppID != "" {
		t.Fatalf("expected no configured ids, got %+v", resp)
	}
}
