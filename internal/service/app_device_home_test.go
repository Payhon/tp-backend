package service

import (
	"testing"
	"time"
)

func TestBuildDeviceListRespIncludesImei(t *testing.T) {
	imei := "862415074075885"
	iccid := "898608861925D1163980"
	bmsCommType := 2
	isOwner := true
	bindingTime := time.Date(2026, 6, 12, 9, 0, 0, 0, time.Local)

	resp := buildDeviceListResp([]appDeviceListRow{
		{
			ID:           "binding-1",
			UserID:       "user-1",
			UserPhone:    "13800000000",
			DeviceID:     "device-1",
			DeviceNumber: "Battery_360111611350535934373730300A354F",
			Iccid:        &iccid,
			Imei:         &imei,
			BmsCommType:  &bmsCommType,
			IsOnline:     1,
			IsOwner:      &isOwner,
			BindingTime:  &bindingTime,
			RelationType: "BINDING",
		},
	}, 1, 1, 20)

	if len(resp.List) != 1 {
		t.Fatalf("expected 1 device, got %d", len(resp.List))
	}
	got := resp.List[0]
	if got.Imei == nil || *got.Imei != imei {
		t.Fatalf("expected imei %q, got %#v", imei, got.Imei)
	}
	if got.Iccid == nil || *got.Iccid != iccid {
		t.Fatalf("expected iccid %q, got %#v", iccid, got.Iccid)
	}
	if got.BmsCommType == nil || *got.BmsCommType != bmsCommType {
		t.Fatalf("expected bms_comm_type %d, got %#v", bmsCommType, got.BmsCommType)
	}
}
