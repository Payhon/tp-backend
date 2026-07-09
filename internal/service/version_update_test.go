package service

import (
	"context"
	"testing"

	"project/internal/model"
	global "project/pkg/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupVersionUpdateTestDB(t *testing.T) *gorm.DB {
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
		`CREATE TABLE version_update_records (
			id text primary key,
			project text not null,
			version_no text not null,
			release_date datetime not null,
			update_content text not null,
			source text not null,
			source_ref text,
			created_at datetime,
			updated_at datetime
		)`,
		`CREATE UNIQUE INDEX uq_version_update_records_project_version_date
			ON version_update_records (project, version_no, release_date)`,
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

func TestVersionUpdateCreateListUpdateDelete(t *testing.T) {
	setupVersionUpdateTestDB(t)
	ctx := context.Background()
	svc := &VersionUpdate{}

	created, err := svc.Create(ctx, model.VersionUpdateCreateReq{
		Project:       model.VersionUpdateProjectMobile,
		VersionNo:     "1.1.8 / 118",
		ReleaseDate:   "2026-07-01",
		UpdateContent: "新增移动端质保信息页",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Source != model.VersionUpdateSourceManual {
		t.Fatalf("Source = %s, want %s", created.Source, model.VersionUpdateSourceManual)
	}

	if _, err := svc.Create(ctx, model.VersionUpdateCreateReq{
		Project:       model.VersionUpdateProjectMobile,
		VersionNo:     "1.1.8 / 118",
		ReleaseDate:   "2026-07-01",
		UpdateContent: "重复记录",
	}); err == nil {
		t.Fatalf("Create() duplicate error = nil, want error")
	}

	list, err := svc.List(ctx, model.VersionUpdateListReq{
		Page:     1,
		PageSize: 10,
		Project:  stringPtrForVersionUpdateTest(model.VersionUpdateProjectMobile),
		Keyword:  stringPtrForVersionUpdateTest("质保"),
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if list.Total != 1 || len(list.List) != 1 {
		t.Fatalf("List() total/list = %d/%d, want 1/1", list.Total, len(list.List))
	}

	updated, err := svc.Update(ctx, created.ID, model.VersionUpdateUpdateReq{
		Project:       stringPtrForVersionUpdateTest(model.VersionUpdateProjectCloudBackend),
		VersionNo:     stringPtrForVersionUpdateTest("abc1234"),
		ReleaseDate:   stringPtrForVersionUpdateTest("2026-07-08"),
		UpdateContent: stringPtrForVersionUpdateTest("新增版本更新记录 CRUD"),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Project != model.VersionUpdateProjectCloudBackend {
		t.Fatalf("Project = %s, want %s", updated.Project, model.VersionUpdateProjectCloudBackend)
	}
	if updated.ReleaseDate != "2026-07-08" {
		t.Fatalf("ReleaseDate = %s, want 2026-07-08", updated.ReleaseDate)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.UpdateContent != "新增版本更新记录 CRUD" {
		t.Fatalf("UpdateContent = %s", got.UpdateContent)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	empty, err := svc.List(ctx, model.VersionUpdateListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List() after delete error = %v", err)
	}
	if empty.Total != 0 {
		t.Fatalf("Total after delete = %d, want 0", empty.Total)
	}
}

func TestVersionUpdateValidation(t *testing.T) {
	setupVersionUpdateTestDB(t)
	ctx := context.Background()
	svc := &VersionUpdate{}

	if _, err := svc.Create(ctx, model.VersionUpdateCreateReq{
		Project:       "UNKNOWN",
		VersionNo:     "1.0.0",
		ReleaseDate:   "2026-07-01",
		UpdateContent: "更新内容",
	}); err == nil {
		t.Fatalf("Create() invalid project error = nil, want error")
	}

	if _, err := svc.Create(ctx, model.VersionUpdateCreateReq{
		Project:       model.VersionUpdateProjectMobile,
		VersionNo:     "1.0.0",
		ReleaseDate:   "2026/07/01",
		UpdateContent: "更新内容",
	}); err == nil {
		t.Fatalf("Create() invalid date error = nil, want error")
	}
}

func stringPtrForVersionUpdateTest(value string) *string {
	return &value
}
