package service

import (
	"context"
	"strings"
	"unicode"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// DeviceProvision 移动端设备开通（扫码/蓝牙绑定）
type DeviceProvision struct{}

type deviceProvisionRow struct {
	DeviceID     string  `gorm:"column:device_id"`
	DeviceNumber string  `gorm:"column:device_number"`
	DeviceName   *string `gorm:"column:device_name"`
	BleMac       *string `gorm:"column:ble_mac"`
	CommChipID   *string `gorm:"column:comm_chip_id"`
	BmsCommType  *int    `gorm:"column:bms_comm_type"`
}

func normalizeMac12(input string) (string, error) {
	s := strings.TrimSpace(input)
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		s = s[2:]
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	out := b.String()
	if len(out) != 12 {
		return "", errcode.NewWithMessage(errcode.CodeParamError, "invalid ble_mac, expected 12 hex chars")
	}
	return out, nil
}

func getDTUDomainPortFromConfig() string {
	// 兼容两种命名，避免后续改配置造成老版本不可用
	if v := strings.TrimSpace(viper.GetString("bms.provision.dtu_domain_port")); v != "" {
		return v
	}
	if v := strings.TrimSpace(viper.GetString("bms.dtu_domain_port")); v != "" {
		return v
	}
	return ""
}

// GetProvisionConfig 获取移动端开通配置
func (*DeviceProvision) GetProvisionConfig(_ context.Context, _ string) (*model.DeviceProvisionConfigResp, error) {
	dtu := getDTUDomainPortFromConfig()
	if dtu == "" {
		return nil, errcode.NewWithMessage(errcode.CodeNotFound, "dtu_domain_port not configured")
	}
	return &model.DeviceProvisionConfigResp{DTUDomainPort: dtu}, nil
}

func (*DeviceProvision) findDeviceByItemUUID(ctx context.Context, itemUUID string, claims *utils.UserClaims) (*deviceProvisionRow, error) {
	itemUUID = strings.TrimSpace(itemUUID)
	if itemUUID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "item_uuid is required")
	}
	if claims == nil || claims.TenantID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeParamError, "claims.tenant_id is required")
	}

	var row deviceProvisionRow
	err := global.DB.WithContext(ctx).
		Table("device_batteries AS dbat").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			d.name AS device_name,
			dbat.ble_mac AS ble_mac,
			dbat.comm_chip_id AS comm_chip_id,
			dbat.bms_comm_type AS bms_comm_type
		`).
		Joins("JOIN devices AS d ON d.id = dbat.device_id").
		Where("dbat.item_uuid = ? AND d.tenant_id = ?", itemUUID, claims.TenantID).
		Scan(&row).Error
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if row.DeviceID == "" {
		return nil, errcode.NewWithMessage(errcode.CodeNotFound, "device not found by item_uuid")
	}
	return &row, nil
}

// GetProvisionInfo 按 item_uuid 查询设备信息（用于“扫码 UUID”路径）
func (*DeviceProvision) GetProvisionInfo(ctx context.Context, req model.DeviceProvisionInfoReq, claims *utils.UserClaims) (*model.DeviceProvisionInfoResp, error) {
	svc := &DeviceProvision{}
	row, err := svc.findDeviceByItemUUID(ctx, req.ItemUUID, claims)
	if err != nil {
		return nil, err
	}

	var cnt int64
	if claims != nil && claims.ID != "" {
		if err := global.DB.WithContext(ctx).
			Table("device_user_bindings").
			Where("device_id = ? AND user_id = ?", row.DeviceID, claims.ID).
			Count(&cnt).Error; err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
	}

	return &model.DeviceProvisionInfoResp{
		DeviceID:     row.DeviceID,
		DeviceNumber: row.DeviceNumber,
		DeviceName:   row.DeviceName,
		BleMac:       row.BleMac,
		CommChipID:   row.CommChipID,
		BmsCommType:  row.BmsCommType,
		IsBound:      cnt > 0,
	}, nil
}

// BindByItemUUID 按 item_uuid 将设备绑定到当前账号
func (*DeviceProvision) BindByItemUUID(ctx context.Context, req model.DeviceProvisionBindReq, claims *utils.UserClaims) (*model.DeviceProvisionBindResp, error) {
	svc := &DeviceProvision{}
	row, err := svc.findDeviceByItemUUID(ctx, req.ItemUUID, claims)
	if err != nil {
		return nil, err
	}

	// 记录 ble_mac（可选，便于后续 BLE 优先连接与排错）
	if req.BleMac != nil && strings.TrimSpace(*req.BleMac) != "" {
		newMac, err := normalizeMac12(*req.BleMac)
		if err != nil {
			return nil, err
		}

		if row.BleMac != nil && strings.TrimSpace(*row.BleMac) != "" {
			existingMac, err := normalizeMac12(*row.BleMac)
			if err == nil && existingMac != newMac {
				return nil, errcode.NewWithMessage(errcode.CodeParamError, "ble_mac mismatch with existing record")
			}
		} else {
			// 只在空值时写入，避免覆盖后台维护数据
			if err := global.DB.WithContext(ctx).
				Exec("UPDATE device_batteries SET ble_mac = ? WHERE device_id = ? AND (ble_mac IS NULL OR ble_mac = '')", newMac, row.DeviceID).Error; err != nil {
				return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}
		}
	}

	// 复用现有绑定逻辑（device_user_bindings + activation_status 等）
	if err := GroupApp.DeviceBinding.BindDevice(model.DeviceBindReq{DeviceNumber: row.DeviceNumber}, claims); err != nil {
		// 针对“数据库错误”做更可读的提示（用于测试环境快速定位迁移/表缺失问题）
		if e, ok := err.(*errcode.Error); ok && e.Code == errcode.CodeDBError {
			sqlErr := ""
			if m, ok := e.Data.(map[string]interface{}); ok {
				if v, ok := m["sql_error"].(string); ok {
					sqlErr = v
				}
			}
			if strings.Contains(sqlErr, "device_user_bindings") && strings.Contains(sqlErr, "does not exist") {
				return nil, errcode.NewWithMessage(errcode.CodeDBError, "数据库缺少表 device_user_bindings，请先执行迁移脚本 backend/sql/13.sql")
			}
			if sqlErr != "" {
				msg := sqlErr
				if len(msg) > 180 {
					msg = msg[:180] + "..."
				}
				return nil, errcode.NewWithMessage(errcode.CodeDBError, "数据库错误: "+msg)
			}
		}
		return nil, err
	}

	// 同步设备激活状态（devices 表 activate_flag/is_enabled/activate_at）
	// NOTE: DeviceBinding 仅更新了 device_batteries 的 activation_status；这里补齐 devices 主表字段，便于后台/统计统一。
	now := utils.GetUTCTime()
	if err := global.DB.WithContext(ctx).
		Exec(
			`UPDATE devices SET activate_flag = 'active', is_enabled = 'enabled', activate_at = ?, update_at = ? WHERE id = ? AND tenant_id = ?`,
			now, now, row.DeviceID, claims.TenantID,
		).Error; err != nil {
		// 绑定已成功，这里尽量不影响用户绑定结果；激活字段后续可通过修复脚本/补偿任务同步。
		// TODO: 若后续需要强一致（激活字段必须成功写入），再改回返回错误并在部署前确认 DB schema 一致。
		logrus.WithError(err).Warn("[device_provision] bind succeeded but sync devices activation failed")
	}

	return &model.DeviceProvisionBindResp{
		DeviceID:     row.DeviceID,
		DeviceNumber: row.DeviceNumber,
	}, nil
}
