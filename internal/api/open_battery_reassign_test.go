package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"project/internal/model"

	"github.com/gin-gonic/gin"
)

func bindMESPackFactoryReassignRequestForTest(t *testing.T, payload interface{}) (model.MESPackFactoryReassignReq, bool) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/openapi/mes/battery/reassign-pack-factory", bytes.NewReader(raw))
	ctx.Request.Header.Set("Content-Type", "application/json")

	var req model.MESPackFactoryReassignReq
	ok := BindAndValidate(ctx, &req)
	return req, ok
}

func TestMESPackFactoryReassignRequestBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req, ok := bindMESPackFactoryReassignRequestForTest(t, map[string]interface{}{
		"serial_numbers":           []string{"SN-001", "SN-002"},
		"target_pack_factory_name": "目标PACK",
		"remark":                   "MES改派",
	})
	if !ok {
		t.Fatal("valid MES reassign request should bind")
	}
	if len(req.SerialNumbers) != 2 || req.TargetPackFactoryName != "目标PACK" {
		t.Fatalf("unexpected bound request: %+v", req)
	}

	invalidPayloads := []map[string]interface{}{
		{"target_pack_factory_name": "目标PACK"},
		{"serial_numbers": []string{}, "target_pack_factory_name": "目标PACK"},
		{"serial_numbers": []string{"SN-001"}},
	}
	for index, payload := range invalidPayloads {
		if _, ok := bindMESPackFactoryReassignRequestForTest(t, payload); ok {
			t.Fatalf("invalid payload %d should be rejected: %#v", index, payload)
		}
	}

	tooMany := make([]string, 501)
	for index := range tooMany {
		tooMany[index] = "SN"
	}
	if _, ok := bindMESPackFactoryReassignRequestForTest(t, map[string]interface{}{
		"serial_numbers":           tooMany,
		"target_pack_factory_name": "目标PACK",
	}); ok {
		t.Fatal("more than 500 serial numbers should be rejected")
	}
}
