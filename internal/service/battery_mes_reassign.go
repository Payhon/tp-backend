package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func normalizeMESPackFactoryReassignSerials(serialNumbers []string) ([]string, error) {
	if len(serialNumbers) == 0 {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "serial_numbers is required"})
	}
	if len(serialNumbers) > 500 {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "serial_numbers最多支持500条"})
	}

	normalized := make([]string, 0, len(serialNumbers))
	seen := make(map[string]struct{}, len(serialNumbers))
	for index, raw := range serialNumbers {
		serialNumber := strings.TrimSpace(raw)
		if serialNumber == "" {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": fmt.Sprintf("serial_numbers第%d项不能为空", index+1),
			})
		}
		if len(serialNumber) > 64 {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": fmt.Sprintf("serial_numbers第%d项长度不能超过64", index+1),
			})
		}
		if _, exists := seen[serialNumber]; exists {
			continue
		}
		seen[serialNumber] = struct{}{}
		normalized = append(normalized, serialNumber)
	}
	return normalized, nil
}

func resolveMESReassignTargetPackFactory(ctx context.Context, tenantID, rawName string) (*model.Org, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "target_pack_factory_name is required"})
	}

	orgs := make([]model.Org, 0, 2)
	if err := global.DB.WithContext(ctx).
		Where("tenant_id = ? AND org_type = ? AND name = ?", tenantID, model.OrgTypePACKFactory, name).
		Limit(2).
		Find(&orgs).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if len(orgs) == 0 {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "目标PACK厂家不存在"})
	}
	if len(orgs) > 1 {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "目标PACK厂家名称不唯一"})
	}
	if orgs[0].Status == nil || strings.TrimSpace(*orgs[0].Status) != model.OrgStatusNormal {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "目标PACK厂家已禁用"})
	}
	return &orgs[0], nil
}

func mesPackFactoryReassignFailureMessage(err error) string {
	if err == nil {
		return ""
	}
	if codeErr, ok := err.(*errcode.Error); ok {
		if codeErr.UseCustomMsg && strings.TrimSpace(codeErr.CustomMsg) != "" {
			return codeErr.CustomMsg
		}
		if data, ok := codeErr.Data.(map[string]interface{}); ok {
			if message, ok := data["message"].(string); ok && strings.TrimSpace(message) != "" {
				return message
			}
		}
		switch codeErr.Code {
		case errcode.CodeDBError:
			return "重新分配PACK厂失败，请稍后重试"
		case errcode.CodeOpDenied, errcode.CodeNoPermission:
			return "无权重新分配该电池的PACK厂"
		case errcode.CodeParamError:
			return "重新分配PACK厂参数无效"
		}
	}
	return err.Error()
}

func normalizeMESReassignRemark(remark *string) *string {
	if remark == nil {
		return nil
	}
	value := strings.TrimSpace(*remark)
	if value == "" {
		return nil
	}
	return &value
}

func (b *Battery) reassignPackFactoryForMESOnce(
	ctx context.Context,
	serialNumber string,
	targetOrg *model.Org,
	remark *string,
	claims *utils.UserClaims,
	openAPIKeyID string,
	requestID string,
) (model.MESPackFactoryReassignResult, error) {
	result := model.MESPackFactoryReassignResult{
		SerialNumber:      serialNumber,
		Status:            model.MESPackFactoryReassignStatusFailed,
		ToPackFactoryName: targetOrg.Name,
	}

	err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var device model.Device
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND device_number = ?", claims.TenantID, serialNumber).
			Take(&device).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "电池不存在"})
			}
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}

		var battery model.DeviceBattery
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("device_id = ?", device.ID).
			Take(&battery).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "设备未出厂，无法重新分配PACK厂"})
			}
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if battery.OwnerOrgID == nil || strings.TrimSpace(*battery.OwnerOrgID) == "" {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "设备未分配PACK厂家"})
		}

		var fromOrg model.Org
		if err := tx.Where("id = ? AND tenant_id = ?", strings.TrimSpace(*battery.OwnerOrgID), claims.TenantID).
			Take(&fromOrg).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "设备当前归属组织不存在"})
			}
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		result.FromPackFactoryName = &fromOrg.Name

		if fromOrg.OrgType != model.OrgTypePACKFactory {
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "设备当前持有方不是PACK厂家"})
		}

		ownerOrgID := strings.TrimSpace(*battery.OwnerOrgID)
		packFactoryOrgID := ""
		if battery.PackFactoryOrgID != nil {
			packFactoryOrgID = strings.TrimSpace(*battery.PackFactoryOrgID)
		}
		if ownerOrgID == targetOrg.ID && packFactoryOrgID == targetOrg.ID {
			result.Status = model.MESPackFactoryReassignStatusUnchanged
			return nil
		}
		if packFactoryOrgID != "" && packFactoryOrgID != ownerOrgID {
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "设备当前持有PACK厂家与PACK组装方数据不一致"})
		}

		if battery.ActivationStatus != nil {
			activationStatus := strings.ToUpper(strings.TrimSpace(*battery.ActivationStatus))
			if activationStatus != "" && activationStatus != "INACTIVE" {
				return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "设备已激活或激活状态异常，不能重新分配PACK厂"})
			}
		}
		if battery.TransferStatus != nil {
			transferStatus := strings.ToUpper(strings.TrimSpace(*battery.TransferStatus))
			if transferStatus != "" && transferStatus != "FACTORY" {
				return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "设备已进入后续流转，不能重新分配PACK厂"})
			}
		}
		if battery.CellBrandSeqNo != nil || battery.BatteryModelSeqNo != nil {
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "设备已完成PACK信息补全，不能重新分配PACK厂"})
		}

		var ownerBindingCount int64
		if err := tx.Model(&model.DeviceUserBinding{}).
			Where("device_id = ? AND is_owner = ?", device.ID, true).
			Count(&ownerBindingCount).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if ownerBindingCount > 0 {
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "设备已绑定用户，不能重新分配PACK厂"})
		}

		now := time.Now().UTC()
		updateResult := tx.Model(&model.DeviceBattery{}).
			Where("device_id = ? AND owner_org_id = ?", device.ID, ownerOrgID).
			Updates(map[string]interface{}{
				"owner_org_id":        targetOrg.ID,
				"pack_factory_org_id": targetOrg.ID,
				"updated_at":          now,
			})
		if updateResult.Error != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": updateResult.Error.Error()})
		}
		if updateResult.RowsAffected != 1 {
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "设备归属已变化，请重试"})
		}

		transferLog := model.DeviceOrgTransfer{
			ID:           uuid.New(),
			DeviceID:     device.ID,
			FromOrgID:    &fromOrg.ID,
			ToOrgID:      &targetOrg.ID,
			OperatorID:   &claims.ID,
			TransferTime: &now,
			Remark:       remark,
			TenantID:     claims.TenantID,
			CreatedAt:    &now,
		}
		if err := tx.Create(&transferLog).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}

		description := fmt.Sprintf("MES重新分配PACK厂：%s -> %s", fromOrg.Name, targetOrg.Name)
		if err := CreateBatteryOperationLogTx(
			tx,
			claims.TenantID,
			device.ID,
			device.DeviceNumber,
			BatteryOpTypePackFactoryReassign,
			&claims.ID,
			&description,
			map[string]interface{}{
				"from_org_id":     fromOrg.ID,
				"from_org_name":   fromOrg.Name,
				"to_org_id":       targetOrg.ID,
				"to_org_name":     targetOrg.Name,
				"source":          "openapi_mes",
				"open_api_key_id": strings.TrimSpace(openAPIKeyID),
				"request_id":      strings.TrimSpace(requestID),
				"remark":          remark,
			},
		); err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}

		result.Status = model.MESPackFactoryReassignStatusReassigned
		return nil
	})
	return result, err
}

// ReassignPackFactoryForMES 批量重新分配尚未投入使用 BMS 板的 PACK 厂。
func (b *Battery) ReassignPackFactoryForMES(
	ctx context.Context,
	req model.MESPackFactoryReassignReq,
	claims *utils.UserClaims,
	openAPIKeyID string,
	requestID string,
) (*model.MESPackFactoryReassignResp, error) {
	if claims == nil || strings.TrimSpace(claims.TenantID) == "" || strings.TrimSpace(claims.ID) == "" {
		return nil, errcode.NewWithMessage(errcode.CodeUnauthorized, "invalid openapi claims")
	}
	serialNumbers, err := normalizeMESPackFactoryReassignSerials(req.SerialNumbers)
	if err != nil {
		return nil, err
	}
	targetOrg, err := resolveMESReassignTargetPackFactory(ctx, claims.TenantID, req.TargetPackFactoryName)
	if err != nil {
		return nil, err
	}
	remark := normalizeMESReassignRemark(req.Remark)

	resp := &model.MESPackFactoryReassignResp{
		RequestID:             strings.TrimSpace(requestID),
		TargetPackFactoryName: targetOrg.Name,
		Total:                 len(serialNumbers),
		Results:               make([]model.MESPackFactoryReassignResult, 0, len(serialNumbers)),
	}
	for _, serialNumber := range serialNumbers {
		result, itemErr := b.reassignPackFactoryForMESOnce(
			ctx,
			serialNumber,
			targetOrg,
			remark,
			claims,
			openAPIKeyID,
			requestID,
		)
		if itemErr != nil {
			message := mesPackFactoryReassignFailureMessage(itemErr)
			result.Status = model.MESPackFactoryReassignStatusFailed
			result.Message = &message
			resp.Failed++
		} else if result.Status == model.MESPackFactoryReassignStatusUnchanged {
			resp.Unchanged++
		} else {
			resp.Success++
		}
		resp.Results = append(resp.Results, result)
	}
	return resp, nil
}
