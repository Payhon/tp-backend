package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolve4GModuleTenantID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/4g-module/check?tenant_id=query-tenant", nil)
	req.Header.Set("X-Tenant-ID", "header-tenant")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	if got := resolve4GModuleTenantID(c); got != "query-tenant" {
		t.Fatalf("expected query tenant first, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/ota/4g-module/check", nil)
	req.Header.Set("X-Tenant-ID", "header-tenant")
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	if got := resolve4GModuleTenantID(c); got != "header-tenant" {
		t.Fatalf("expected X-Tenant-ID fallback, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/ota/4g-module/check", nil)
	req.Header.Set("X-TenantID", "legacy-tenant")
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	if got := resolve4GModuleTenantID(c); got != "legacy-tenant" {
		t.Fatalf("expected legacy X-TenantID fallback, got %q", got)
	}
}
