package service

import (
	"context"
	"testing"

	"project/pkg/errcode"
	"project/pkg/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestResolveRuntimeApp(t *testing.T) {
	oldDB := global.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })

	if err := db.Exec(`
		CREATE TABLE apps (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			appid TEXT NOT NULL,
			created_at DATETIME
		);
		CREATE TABLE wx_mp_apps (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			appid TEXT NOT NULL
		);
	`).Error; err != nil {
		t.Fatalf("create schema failed: %v", err)
	}

	for _, stmt := range []struct {
		query string
		args  []interface{}
	}{
		{
			query: `INSERT INTO apps (id, tenant_id, appid, created_at) VALUES (?, ?, ?, ?)`,
			args:  []interface{}{"app-main", "tenant-1", "__UNI__MAIN", "2026-01-01 00:00:00"},
		},
		{
			query: `INSERT INTO apps (id, tenant_id, appid, created_at) VALUES (?, ?, ?, ?)`,
			args:  []interface{}{"app-pack", "tenant-1", "wx-pack", "2026-02-01 00:00:00"},
		},
		{
			query: `INSERT INTO wx_mp_apps (id, tenant_id, appid) VALUES (?, ?, ?)`,
			args:  []interface{}{"wx-config-1", "tenant-1", "wx-tenant-default"},
		},
	} {
		if err := db.Exec(stmt.query, stmt.args...).Error; err != nil {
			t.Fatalf("insert fixture failed: %v", err)
		}
	}

	tests := []struct {
		name         string
		runtimeAppID string
		wantAppID    string
	}{
		{name: "canonical app id stays direct", runtimeAppID: "__UNI__MAIN", wantAppID: "__UNI__MAIN"},
		{name: "pack WeChat app id stays direct", runtimeAppID: "wx-pack", wantAppID: "wx-pack"},
		{name: "tenant WeChat app id maps to default app", runtimeAppID: "wx-tenant-default", wantAppID: "__UNI__MAIN"},
		{name: "empty app id keeps public default fallback", runtimeAppID: "", wantAppID: "__UNI__MAIN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := resolveRuntimeApp(context.Background(), "tenant-1", tt.runtimeAppID)
			if err != nil {
				t.Fatalf("resolveRuntimeApp() error = %v", err)
			}
			if app.AppID != tt.wantAppID {
				t.Fatalf("resolveRuntimeApp() appid = %q, want %q", app.AppID, tt.wantAppID)
			}
		})
	}

	_, err = resolveRuntimeApp(context.Background(), "tenant-1", "wx-unknown")
	if err == nil {
		t.Fatal("resolveRuntimeApp() expected unknown app id error")
	}
	appErr, ok := err.(*errcode.Error)
	if !ok || appErr.Code != errcode.CodeNotFound {
		t.Fatalf("resolveRuntimeApp() error = %#v, want CodeNotFound", err)
	}
}
