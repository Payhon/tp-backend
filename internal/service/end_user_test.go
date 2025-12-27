package service

import (
	"strings"
	"testing"
)

func TestEndUserListSelectSQL_NoMissingAlias(t *testing.T) {
	db := newDryRunPostgresDB(t)

	base := db.Table("device_user_bindings AS dub").
		Joins("LEFT JOIN devices AS d ON d.id = dub.device_id").
		Joins("LEFT JOIN users AS u ON u.id = dub.user_id").
		Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
		Joins("LEFT JOIN orgs AS org ON org.id = dbat.owner_org_id").
		Where("d.tenant_id = ?", "tenant-1")

	selectSQL, selectArgs := endUserListSelectSQL("org-1")

	var rows []map[string]interface{}
	res := base.Select(selectSQL, selectArgs...).
		Group("u.id, u.name, u.phone_number, owner_org_id, owner_org_name").
		Order("last_bind_at DESC").
		Offset(0).
		Limit(10).
		Find(&rows)

	if res.Error != nil {
		t.Fatalf("build sql: %v", res.Error)
	}

	sql := res.Statement.SQL.String()
	if sql == "" {
		t.Fatalf("expected non-empty sql")
	}
	if strings.Contains(sql, "de.") {
		t.Fatalf("unexpected dealer alias de in sql: %s", sql)
	}
	if !strings.Contains(sql, "orgs AS org") {
		t.Fatalf("expected org join in sql: %s", sql)
	}
	if !strings.Contains(sql, "dbat.owner_org_id") {
		t.Fatalf("expected dbat.owner_org_id in sql: %s", sql)
	}
	if !strings.Contains(sql, "org.name") {
		t.Fatalf("expected org.name in sql: %s", sql)
	}
}
