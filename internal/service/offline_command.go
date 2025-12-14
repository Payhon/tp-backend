package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"project/internal/model"
	query "project/internal/query"
	"project/pkg/constant"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OfflineCommand 离线指令服务
type OfflineCommand struct{}

const (
	OfflineCmdStatusPending   = "PENDING"
	OfflineCmdStatusSent      = "SENT"
	OfflineCmdStatusSuccess   = "SUCCESS"
	OfflineCmdStatusFailed    = "FAILED"
	OfflineCmdStatusCancelled = "CANCELLED"
)

func (*OfflineCommand) Create(ctx context.Context, req model.OfflineCommandCreateReq, claims *utils.UserClaims) error {
	req.CommandType = strings.TrimSpace(req.CommandType)
	req.Identify = strings.TrimSpace(req.Identify)
	if req.CommandType == "" || req.Identify == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "invalid command"})
	}

	// 校验设备归属与在线状态
	device, err := query.Device.WithContext(ctx).
		Where(query.Device.ID.Eq(req.DeviceID), query.Device.TenantID.Eq(claims.TenantID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "device not found"})
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if device.IsOnline == 1 {
		return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "device is online, use immediate command"})
	}

	now := time.Now().UTC()
	task := &model.OfflineCommandTask{
		ID:           uuid.New(),
		TenantID:     claims.TenantID,
		DeviceID:     req.DeviceID,
		DeviceNumber: device.DeviceNumber,
		CommandType:  req.CommandType,
		Identify:     req.Identify,
		Payload:      req.Value,
		CreatedBy:    &claims.ID,
		CreatedAt:    now,
		Status:       OfflineCmdStatusPending,
	}

	if err := global.DB.WithContext(ctx).Create(task).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func (*OfflineCommand) List(ctx context.Context, req model.OfflineCommandListReq, claims *utils.UserClaims, dealerID string) (*model.OfflineCommandListResp, error) {
	type row struct {
		ID           string     `gorm:"column:id"`
		DeviceID     string     `gorm:"column:device_id"`
		DeviceNumber string     `gorm:"column:device_number"`
		CommandType  string     `gorm:"column:command_type"`
		Identify     string     `gorm:"column:identify"`
		Status       string     `gorm:"column:status"`
		CreatedAt    time.Time  `gorm:"column:created_at"`
		DispatchedAt *time.Time `gorm:"column:dispatched_at"`
		ExecutedAt   *time.Time `gorm:"column:executed_at"`
		ErrorMessage *string    `gorm:"column:error_message"`
		OperatorName *string    `gorm:"column:operator_name"`
	}

	db := global.DB.WithContext(ctx)

	// dealer 隔离：只允许查看名下设备的离线指令
	base := db.Table("offline_command_tasks oct").
		Joins("LEFT JOIN users u ON u.id = oct.created_by").
		Joins("LEFT JOIN device_batteries dbat ON dbat.device_id = oct.device_id").
		Where("oct.tenant_id = ?", claims.TenantID)
	if dealerID != "" {
		base = base.Where("dbat.dealer_id = ?", dealerID)
	}

	if req.DeviceNumber != nil && strings.TrimSpace(*req.DeviceNumber) != "" {
		base = base.Where("oct.device_number ILIKE ?", "%"+strings.TrimSpace(*req.DeviceNumber)+"%")
	}
	if req.CommandType != nil && strings.TrimSpace(*req.CommandType) != "" {
		base = base.Where("oct.command_type ILIKE ?", "%"+strings.TrimSpace(*req.CommandType)+"%")
	}
	if req.Status != nil && *req.Status != "" {
		base = base.Where("oct.status = ?", *req.Status)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (req.Page - 1) * req.PageSize
	var rows []row
	if err := base.
		Select(`oct.id, oct.device_id, oct.device_number, oct.command_type, oct.identify, oct.status, oct.created_at, oct.dispatched_at, oct.executed_at, oct.error_message,
COALESCE(u.name, u.phone_number) AS operator_name`).
		Order("oct.created_at DESC").
		Limit(req.PageSize).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.OfflineCommandListItemResp, 0, len(rows))
	for _, r := range rows {
		var dispatchedAtStr *string
		if r.DispatchedAt != nil {
			s := r.DispatchedAt.In(time.Local).Format("2006-01-02 15:04:05")
			dispatchedAtStr = &s
		}
		var executedAtStr *string
		if r.ExecutedAt != nil {
			s := r.ExecutedAt.In(time.Local).Format("2006-01-02 15:04:05")
			executedAtStr = &s
		}

		list = append(list, model.OfflineCommandListItemResp{
			ID:           r.ID,
			DeviceID:     r.DeviceID,
			DeviceNumber: r.DeviceNumber,
			CommandType:  r.CommandType,
			Identify:     r.Identify,
			Status:       r.Status,
			CreatedAt:    r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
			OperatorName: r.OperatorName,
			DispatchedAt: dispatchedAtStr,
			ExecutedAt:   executedAtStr,
			ErrorMessage: r.ErrorMessage,
		})
	}

	return &model.OfflineCommandListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (*OfflineCommand) Detail(ctx context.Context, id string, claims *utils.UserClaims, dealerID string) (*model.OfflineCommandDetailResp, error) {
	type row struct {
		ID           string     `gorm:"column:id"`
		DeviceID     string     `gorm:"column:device_id"`
		DeviceNumber string     `gorm:"column:device_number"`
		CommandType  string     `gorm:"column:command_type"`
		Identify     string     `gorm:"column:identify"`
		Payload      *string    `gorm:"column:payload"`
		Status       string     `gorm:"column:status"`
		MessageID    *string    `gorm:"column:message_id"`
		CreatedAt    time.Time  `gorm:"column:created_at"`
		DispatchedAt *time.Time `gorm:"column:dispatched_at"`
		ExecutedAt   *time.Time `gorm:"column:executed_at"`
		ErrorMessage *string    `gorm:"column:error_message"`
		OperatorName *string    `gorm:"column:operator_name"`
	}

	db := global.DB.WithContext(ctx)
	q := db.Table("offline_command_tasks oct").
		Joins("LEFT JOIN users u ON u.id = oct.created_by").
		Joins("LEFT JOIN device_batteries dbat ON dbat.device_id = oct.device_id").
		Where("oct.tenant_id = ? AND oct.id = ?", claims.TenantID, id)
	if dealerID != "" {
		q = q.Where("dbat.dealer_id = ?", dealerID)
	}

	var r row
	if err := q.Select(`oct.id, oct.device_id, oct.device_number, oct.command_type, oct.identify, oct.payload, oct.status, oct.message_id, oct.created_at, oct.dispatched_at, oct.executed_at, oct.error_message,
COALESCE(u.name, u.phone_number) AS operator_name`).Scan(&r).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if r.ID == "" {
		return nil, errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "offline command not found"})
	}

	var dispatchedAtStr *string
	if r.DispatchedAt != nil {
		s := r.DispatchedAt.In(time.Local).Format("2006-01-02 15:04:05")
		dispatchedAtStr = &s
	}
	var executedAtStr *string
	if r.ExecutedAt != nil {
		s := r.ExecutedAt.In(time.Local).Format("2006-01-02 15:04:05")
		executedAtStr = &s
	}

	resp := &model.OfflineCommandDetailResp{
		ID:           r.ID,
		DeviceID:     r.DeviceID,
		DeviceNumber: r.DeviceNumber,
		CommandType:  r.CommandType,
		Identify:     r.Identify,
		Payload:      r.Payload,
		Status:       r.Status,
		MessageID:    r.MessageID,
		CreatedAt:    r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
		OperatorName: r.OperatorName,
		DispatchedAt: dispatchedAtStr,
		ExecutedAt:   executedAtStr,
		ErrorMessage: r.ErrorMessage,
	}

	// 补充命令日志（返回内容/错误等）
	if r.MessageID != nil && *r.MessageID != "" {
		logRow, err := query.CommandSetLog.WithContext(ctx).
			Where(query.CommandSetLog.MessageID.Eq(*r.MessageID)).
			First()
		if err == nil && logRow != nil {
			resp.CommandLogStatus = logRow.Status
			resp.CommandLogRspData = logRow.RspDatum
			resp.CommandLogErrorMsg = logRow.ErrorMessage
		}
	}

	return resp, nil
}

func (*OfflineCommand) Cancel(ctx context.Context, id string, claims *utils.UserClaims, dealerID string) error {
	db := global.DB.WithContext(ctx)

	// 经销商隔离：只能取消名下设备的离线指令
	q := db.Table("offline_command_tasks oct").
		Joins("LEFT JOIN device_batteries dbat ON dbat.device_id = oct.device_id").
		Where("oct.tenant_id = ? AND oct.id = ? AND oct.status = ?", claims.TenantID, id, OfflineCmdStatusPending)
	if dealerID != "" {
		q = q.Where("dbat.dealer_id = ?", dealerID)
	}

	res := q.Updates(map[string]interface{}{
		"status":      OfflineCmdStatusCancelled,
		"executed_at": time.Now().UTC(),
	})
	if res.Error != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "cannot cancel (not found or already executed)"})
	}
	return nil
}

// ExecutePendingForDevice 设备上线时触发：执行该设备待执行离线指令
func (c *OfflineCommand) ExecutePendingForDevice(ctx context.Context, deviceID string) {
	// 保护：命令总线未初始化时直接跳过
	if GroupApp.CommandData.downlinkBus == nil {
		return
	}

	db := global.DB.WithContext(ctx)
	_ = db.Transaction(func(tx *gorm.DB) error {
		var tasks []model.OfflineCommandTask
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("device_id = ? AND status = ?", deviceID, OfflineCmdStatusPending).
			Order("created_at ASC").
			Limit(20).
			Find(&tasks).Error; err != nil {
			logrus.WithError(err).WithField("device_id", deviceID).Warn("offline command: load pending failed")
			return nil
		}
		if len(tasks) == 0 {
			return nil
		}

		now := time.Now().UTC()
		for _, t := range tasks {
			// 再次确认设备在线
			dev, err := query.Device.WithContext(ctx).Where(query.Device.ID.Eq(deviceID)).First()
			if err != nil || dev == nil || dev.IsOnline != 1 {
				return nil
			}

			// 调用现有命令下发（返回 message_id）
			put := &model.PutMessageForCommand{
				DeviceID: t.DeviceID,
				Value:    t.Payload,
				Identify: t.Identify,
			}
			operatorID := ""
			if t.CreatedBy != nil {
				operatorID = *t.CreatedBy
			}

			messageID, err := GroupApp.CommandData.CommandPutMessageReturnMessageID(ctx, operatorID, put, strconv.Itoa(constant.Manual))
			if err != nil {
				errMsg := err.Error()
				_ = tx.Table(model.TableNameOfflineCommandTask).
					Where("id = ? AND status = ?", t.ID, OfflineCmdStatusPending).
					Updates(map[string]interface{}{
						"status":        OfflineCmdStatusFailed,
						"dispatched_at": now,
						"executed_at":   now,
						"error_message": errMsg,
					}).Error
				continue
			}

			_ = tx.Table(model.TableNameOfflineCommandTask).
				Where("id = ? AND status = ?", t.ID, OfflineCmdStatusPending).
				Updates(map[string]interface{}{
					"status":        OfflineCmdStatusSent,
					"dispatched_at": now,
					"message_id":    messageID,
				}).Error
		}
		return nil
	})
}
