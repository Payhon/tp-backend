package service

import (
	"context"
	"strings"
	"time"

	"project/pkg/errcode"
	global "project/pkg/global"
)

func touchDeviceLastConnectedAt(ctx context.Context, deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return errcode.NewWithMessage(errcode.CodeParamError, "device_id is required")
	}

	result := global.DB.WithContext(ctx).
		Table("devices").
		Where("id = ?", deviceID).
		UpdateColumn("last_connected_at", time.Now().UTC())
	if result.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": result.Error.Error()})
	}
	if result.RowsAffected == 0 {
		return errcode.NewWithMessage(errcode.CodeParamError, "device not found")
	}
	return nil
}
