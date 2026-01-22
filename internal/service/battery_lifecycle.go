package service

import (
	"context"
	"fmt"
	"time"

	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

func getOrgByID(ctx context.Context, tenantID, orgID string) (*model.Org, error) {
	if orgID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "组织ID不能为空",
		})
	}
	org, err := query.Org.WithContext(ctx).
		Where(query.Org.ID.Eq(orgID), query.Org.TenantID.Eq(tenantID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "组织不存在",
			})
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	return org, nil
}

// FactoryOutBattery 电池出厂（厂家 -> PACK/经销商）
func (*Battery) FactoryOutBattery(ctx context.Context, req model.BatteryFactoryOutReq, claims *utils.UserClaims, operatorOrgID string) error {
	targetOrg, err := getOrgByID(ctx, claims.TenantID, req.ToOrgID)
	if err != nil {
		return err
	}
	if targetOrg.OrgType != model.OrgTypePACKFactory && targetOrg.OrgType != model.OrgTypeDealer {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "出厂目标仅支持 PACK 厂或经销商",
		})
	}

	// 仅厂家可出厂（无组织的超级管理员视为厂家）
	if operatorOrgID != "" {
		operatorOrg, err := getOrgByID(ctx, claims.TenantID, operatorOrgID)
		if err != nil {
			return err
		}
		if operatorOrg.OrgType != model.OrgTypeBMSFactory {
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
				"message": "仅厂家账号可执行出厂操作",
			})
		}
	}

	now := time.Now().UTC()

	return global.DB.Transaction(func(tx *gorm.DB) error {
		// 设备校验
		var device model.Device
		if err := tx.Where("id = ? AND tenant_id = ?", req.DeviceID, claims.TenantID).First(&device).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "设备不存在",
				})
			}
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}

		var dbat model.DeviceBattery
		err := tx.Where("device_id = ?", device.ID).First(&dbat).Error

		var fromOrgID *string
		var fromOrgName string
		if err == nil {
			if dbat.OwnerOrgID != nil && *dbat.OwnerOrgID != "" {
				fromOrgID = dbat.OwnerOrgID
				fromOrg, err := getOrgByID(ctx, claims.TenantID, *dbat.OwnerOrgID)
				if err != nil {
					return err
				}
				fromOrgName = fromOrg.Name
				if fromOrg.OrgType != model.OrgTypeBMSFactory {
					return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
						"message": "当前不在厂家库存，请使用调拨操作",
					})
				}
			}
		} else if err != gorm.ErrRecordNotFound {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}

		// 创建/更新 device_batteries
		if err == gorm.ErrRecordNotFound {
			newBattery := &model.DeviceBattery{
				DeviceID:   device.ID,
				OwnerOrgID: &targetOrg.ID,
				UpdatedAt:  &now,
			}
			if operatorOrgID != "" {
				newBattery.BmsFactoryOrgID = &operatorOrgID
			}
			if targetOrg.OrgType == model.OrgTypePACKFactory {
				newBattery.PackFactoryOrgID = &targetOrg.ID
			}
			if err := tx.Create(newBattery).Error; err != nil {
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"sql_error": err.Error(),
				})
			}
		} else {
			updates := map[string]interface{}{
				"owner_org_id": targetOrg.ID,
				"updated_at":   now,
			}
			if dbat.BmsFactoryOrgID == nil && operatorOrgID != "" {
				updates["bms_factory_org_id"] = operatorOrgID
			}
			if dbat.PackFactoryOrgID == nil && targetOrg.OrgType == model.OrgTypePACKFactory {
				updates["pack_factory_org_id"] = targetOrg.ID
			}
			if err := tx.Model(&model.DeviceBattery{}).Where("device_id = ?", device.ID).Updates(updates).Error; err != nil {
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"sql_error": err.Error(),
				})
			}
		}

		// 记录组织转移日志
		transferLog := model.DeviceOrgTransfer{
			ID:           uuid.New(),
			DeviceID:     device.ID,
			FromOrgID:    fromOrgID,
			ToOrgID:      &targetOrg.ID,
			OperatorID:   &claims.ID,
			TransferTime: &now,
			Remark:       req.Remark,
			TenantID:     claims.TenantID,
			CreatedAt:    &now,
		}
		if err := tx.Create(&transferLog).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}

		fromLabel := "厂家"
		if fromOrgName != "" {
			fromLabel = fromOrgName
		}
		desc := fmt.Sprintf("出厂：%s -> %s", fromLabel, targetOrg.Name)
		_ = CreateBatteryOperationLogTx(tx, claims.TenantID, device.ID, device.DeviceNumber, BatteryOpTypeFactoryOut, &claims.ID, &desc, map[string]any{
			"from_org_id": fromOrgID,
			"to_org_id":   targetOrg.ID,
			"remark":      req.Remark,
		})

		return nil
	})
}

// TransferBattery 电池调拨（组织转移）
func (*Battery) TransferBattery(ctx context.Context, req model.BatteryTransferReq, claims *utils.UserClaims, operatorOrgID string) error {
	targetOrg, err := getOrgByID(ctx, claims.TenantID, req.ToOrgID)
	if err != nil {
		return err
	}

	var operatorOrgType string
	if operatorOrgID == "" {
		operatorOrgType = model.OrgTypeBMSFactory
	} else {
		operatorOrg, err := getOrgByID(ctx, claims.TenantID, operatorOrgID)
		if err != nil {
			return err
		}
		operatorOrgType = operatorOrg.OrgType
	}

	now := time.Now().UTC()

	return global.DB.Transaction(func(tx *gorm.DB) error {
		var device model.Device
		if err := tx.Where("id = ? AND tenant_id = ?", req.DeviceID, claims.TenantID).First(&device).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "设备不存在",
				})
			}
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}

		var dbat model.DeviceBattery
		if err := tx.Where("device_id = ?", device.ID).First(&dbat).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "设备未出厂，无法调拨",
				})
			}
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}

		if dbat.OwnerOrgID == nil || *dbat.OwnerOrgID == "" {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "设备未分配组织，无法调拨",
			})
		}

		fromOrg, err := getOrgByID(ctx, claims.TenantID, *dbat.OwnerOrgID)
		if err != nil {
			return err
		}

		// 权限校验：组织用户需在子树内
		if operatorOrgID != "" {
			var count int64
			if err := tx.Table("org_closure").
				Where("tenant_id = ? AND ancestor_id = ? AND descendant_id = ?", claims.TenantID, operatorOrgID, *dbat.OwnerOrgID).
				Count(&count).Error; err != nil {
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"sql_error": err.Error(),
				})
			}
			if count == 0 {
				return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
					"message": "无权操作该电池",
				})
			}
		}

		switch operatorOrgType {
		case model.OrgTypeBMSFactory:
			if fromOrg.OrgType != model.OrgTypePACKFactory {
				return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
					"message": "厂家仅支持从 PACK 厂调拨",
				})
			}
			if targetOrg.OrgType != model.OrgTypePACKFactory && targetOrg.OrgType != model.OrgTypeDealer {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "厂家调拨目标仅支持 PACK 厂或经销商",
				})
			}
		case model.OrgTypePACKFactory:
			if fromOrg.OrgType != model.OrgTypePACKFactory {
				return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
					"message": "PACK 厂仅支持调拨自身库存",
				})
			}
			if targetOrg.OrgType != model.OrgTypeDealer && targetOrg.OrgType != model.OrgTypeStore {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "PACK 厂调拨目标仅支持经销商或门店",
				})
			}
			if operatorOrgID != "" {
				var count int64
				if err := tx.Table("org_closure").
					Where("tenant_id = ? AND ancestor_id = ? AND descendant_id = ?", claims.TenantID, operatorOrgID, targetOrg.ID).
					Count(&count).Error; err != nil {
					return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
						"sql_error": err.Error(),
					})
				}
				if count == 0 {
					return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
						"message": "目标机构不在当前 PACK 厂范围内",
					})
				}
			}
		case model.OrgTypeDealer:
			if fromOrg.OrgType != model.OrgTypeStore {
				return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
					"message": "经销商仅支持门店间调拨",
				})
			}
			if targetOrg.OrgType != model.OrgTypeStore {
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "经销商调拨目标仅支持门店",
				})
			}
			if operatorOrgID != "" {
				var count int64
				if err := tx.Table("org_closure").
					Where("tenant_id = ? AND ancestor_id = ? AND descendant_id = ?", claims.TenantID, operatorOrgID, targetOrg.ID).
					Count(&count).Error; err != nil {
					return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
						"sql_error": err.Error(),
					})
				}
				if count == 0 {
					return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
						"message": "目标门店不在当前经销商范围内",
					})
				}
			}
		default:
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
				"message": "当前账号无调拨权限",
			})
		}

		updates := map[string]interface{}{
			"owner_org_id": targetOrg.ID,
			"updated_at":   now,
		}
		if err := tx.Model(&model.DeviceBattery{}).Where("device_id = ?", device.ID).Updates(updates).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}

		transferLog := model.DeviceOrgTransfer{
			ID:           uuid.New(),
			DeviceID:     device.ID,
			FromOrgID:    dbat.OwnerOrgID,
			ToOrgID:      &targetOrg.ID,
			OperatorID:   &claims.ID,
			TransferTime: &now,
			Remark:       req.Remark,
			TenantID:     claims.TenantID,
			CreatedAt:    &now,
		}
		if err := tx.Create(&transferLog).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}

		desc := fmt.Sprintf("调拨：%s -> %s", fromOrg.Name, targetOrg.Name)
		_ = CreateBatteryOperationLogTx(tx, claims.TenantID, device.ID, device.DeviceNumber, BatteryOpTypeTransfer, &claims.ID, &desc, map[string]any{
			"from_org_id": dbat.OwnerOrgID,
			"to_org_id":   targetOrg.ID,
			"remark":      req.Remark,
		})

		return nil
	})
}

// ActivateBattery 电池激活（绑定到 APP 用户）
func (*Battery) ActivateBattery(ctx context.Context, req model.BatteryActivateReq, claims *utils.UserClaims, operatorOrgID string) error {
	// 终端用户校验
	user, err := query.User.WithContext(ctx).
		Where(query.User.ID.Eq(req.UserID), query.User.TenantID.Eq(claims.TenantID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "APP 用户不存在",
			})
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	if user.UserKind != nil && *user.UserKind != model.UserKindEndUser {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "请选择终端用户账号",
		})
	}

	device, err := query.Device.WithContext(ctx).
		Where(query.Device.ID.Eq(req.DeviceID), query.Device.TenantID.Eq(claims.TenantID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "设备不存在",
			})
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 操作权限：组织用户需在子树内
	if err := checkDeviceOrgAccess(ctx, device.ID, claims.TenantID, operatorOrgID); err != nil {
		return err
	}

	tx := query.Use(global.DB).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查是否已绑定该用户
	if _, err := tx.DeviceUserBinding.WithContext(ctx).
		Where(
			tx.DeviceUserBinding.DeviceID.Eq(device.ID),
			tx.DeviceUserBinding.UserID.Eq(req.UserID),
		).
		First(); err == nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "该设备已绑定此用户",
		})
	} else if err != gorm.ErrRecordNotFound {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 获取用户归属组织（用于组织校验/补充 owner_org_id）
	userOrgID, err := getUserOrgID(req.UserID)
	if err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 查询现有绑定数量，确定主用户
	existBindings, err := tx.DeviceUserBinding.WithContext(ctx).
		Where(tx.DeviceUserBinding.DeviceID.Eq(device.ID)).
		Find()
	if err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	isFirstBinding := len(existBindings) == 0

	now := time.Now().UTC()

	// 处理 device_batteries 激活状态
	deviceBattery, err := tx.DeviceBattery.WithContext(ctx).
		Where(tx.DeviceBattery.DeviceID.Eq(device.ID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			newBattery := &model.DeviceBattery{
				DeviceID:         device.ID,
				ActivationStatus: StringPtr("ACTIVE"),
				TransferStatus:   StringPtr("USER"),
				ActivationDate:   &now,
				UpdatedAt:        &now,
			}
			if userOrgID != "" {
				newBattery.OwnerOrgID = &userOrgID
			}
			if err := tx.DeviceBattery.WithContext(ctx).Create(newBattery); err != nil {
				tx.Rollback()
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"sql_error": err.Error(),
				})
			}
		} else {
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}
	} else {
		// 若用户有组织信息，校验设备归属组织
		if userOrgID != "" && deviceBattery.OwnerOrgID != nil && *deviceBattery.OwnerOrgID != "" {
			var count int64
			global.DB.Table("org_closure").
				Where("tenant_id = ? AND ancestor_id = ? AND descendant_id = ?",
					claims.TenantID, userOrgID, *deviceBattery.OwnerOrgID).
				Count(&count)
			if count == 0 {
				tx.Rollback()
				return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "该用户不在设备所属组织范围内",
				})
			}
		}

		updates := map[string]interface{}{
			"activation_status": "ACTIVE",
			"transfer_status":   "USER",
			"activation_date":   now,
			"updated_at":        now,
		}
		if deviceBattery.OwnerOrgID == nil && userOrgID != "" {
			updates["owner_org_id"] = userOrgID
		}

		if _, err := tx.DeviceBattery.WithContext(ctx).
			Where(tx.DeviceBattery.DeviceID.Eq(device.ID)).
			Updates(updates); err != nil {
			tx.Rollback()
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}
	}

	// 创建绑定关系
	isOwner := isFirstBinding
	binding := &model.DeviceUserBinding{
		ID:          uuid.New(),
		UserID:      req.UserID,
		DeviceID:    device.ID,
		BindingTime: &now,
		IsOwner:     &isOwner,
		Remark:      req.Remark,
	}

	if err := tx.DeviceUserBinding.WithContext(ctx).Create(binding); err != nil {
		tx.Rollback()
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	if err := tx.Commit(); err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	desc := fmt.Sprintf("激活绑定：%s", user.PhoneNumber)
	_ = CreateBatteryOperationLog(ctx, claims.TenantID, device.ID, device.DeviceNumber, BatteryOpTypeActivate, &claims.ID, &desc, map[string]any{
		"user_id":  req.UserID,
		"remark":   req.Remark,
		"is_owner": isOwner,
	})

	return nil
}
