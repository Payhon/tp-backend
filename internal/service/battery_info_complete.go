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

	"gorm.io/gorm"
)

type batteryInfoCompleteDeviceRow struct {
	DeviceID          string  `gorm:"column:device_id"`
	DeviceNumber      string  `gorm:"column:device_number"`
	OwnerOrgID        *string `gorm:"column:owner_org_id"`
	PackFactoryOrgID  *string `gorm:"column:pack_factory_org_id"`
	CellBrandSeqNo    *int16  `gorm:"column:cell_brand_seq_no"`
	BatteryModelSeqNo *int16  `gorm:"column:battery_model_seq_no"`
}

func validateBatteryInfoCompleteOperator(ctx context.Context, claims *utils.UserClaims) (string, error) {
	operatorOrgID, err := getOperatorOrgID(claims)
	if err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if operatorOrgID == "" {
		return "", errcode.New(errcode.CodeNoPermission)
	}

	operatorOrg, err := getOrgByID(ctx, claims.TenantID, operatorOrgID)
	if err != nil {
		return "", err
	}
	if operatorOrg.OrgType != model.OrgTypePACKFactory {
		return "", errcode.New(errcode.CodeNoPermission)
	}

	return operatorOrgID, nil
}

func resolveBatteryInfoCompleteModel(ctx context.Context, claims *utils.UserClaims, operatorOrgID string, seqNo int16) (model.BatteryModelResp, error) {
	type row struct {
		ID      string  `gorm:"column:id"`
		SeqNo   *int16  `gorm:"column:seq_no"`
		Name    string  `gorm:"column:name"`
		OrgID   *string `gorm:"column:org_id"`
		OrgName *string `gorm:"column:org_name"`
	}

	q := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryPackModel+" AS bm").
		Select("bm.id, bm.seq_no, bm.name, bm.org_id, org.name AS org_name").
		Joins("LEFT JOIN orgs org ON org.id = bm.org_id").
		Where("bm.tenant_id = ? AND bm.seq_no = ?", claims.TenantID, seqNo)

	if operatorOrgID != "" {
		q = q.Where("bm.org_id = ?", operatorOrgID)
	}

	rows := make([]row, 0, 2)
	if err := q.Limit(2).Scan(&rows).Error; err != nil {
		return model.BatteryModelResp{}, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if len(rows) == 0 {
		return model.BatteryModelResp{}, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "所选电池型号不存在"})
	}
	if operatorOrgID == "" && len(rows) > 1 {
		return model.BatteryModelResp{}, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "所选电池型号序号存在多个归属PACK厂家，请使用PACK厂家账号操作"})
	}

	return model.BatteryModelResp{
		ID:      rows[0].ID,
		SeqNo:   rows[0].SeqNo,
		Name:    rows[0].Name,
		OrgID:   rows[0].OrgID,
		OrgName: rows[0].OrgName,
	}, nil
}

func resolveBatteryInfoCompleteCellBrand(ctx context.Context, claims *utils.UserClaims, seqNo int16) (*model.BatteryCellBrandItemResp, error) {
	var row struct {
		ID    string `gorm:"column:id"`
		SeqNo int16  `gorm:"column:seq_no"`
		Name  string `gorm:"column:name"`
	}
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryCellBrand).
		Select("id, seq_no, name").
		Where("tenant_id = ? AND seq_no = ?", claims.TenantID, seqNo).
		Limit(1).
		Scan(&row).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if row.ID == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "所选电芯品牌不存在"})
	}
	return &model.BatteryCellBrandItemResp{ID: row.ID, SeqNo: row.SeqNo, Name: row.Name}, nil
}

// CompleteBatteryInfo 批量补全电池信息
func (*Battery) CompleteBatteryInfo(ctx context.Context, req model.BatteryCompleteInfoReq, claims *utils.UserClaims) (*model.BatteryCompleteInfoResp, error) {
	operatorOrgID, err := validateBatteryInfoCompleteOperator(ctx, claims)
	if err != nil {
		return nil, err
	}

	cellBrand, err := resolveBatteryInfoCompleteCellBrand(ctx, claims, req.CellBrandSeqNo)
	if err != nil {
		return nil, err
	}
	packModel, err := resolveBatteryInfoCompleteModel(ctx, claims, operatorOrgID, req.BatteryModelSeqNo)
	if err != nil {
		return nil, err
	}

	deviceIDs := make([]string, 0, len(req.DeviceIDs))
	seen := make(map[string]struct{}, len(req.DeviceIDs))
	for _, id := range req.DeviceIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deviceIDs = append(deviceIDs, id)
	}
	if len(deviceIDs) == 0 {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "请选择至少一条电池记录"})
	}

	rows := make([]batteryInfoCompleteDeviceRow, 0, len(deviceIDs))
	if err := global.DB.WithContext(ctx).
		Table("devices AS d").
		Select(`
			d.id AS device_id,
			d.device_number AS device_number,
			dbat.owner_org_id AS owner_org_id,
			dbat.pack_factory_org_id AS pack_factory_org_id,
			dbat.cell_brand_seq_no AS cell_brand_seq_no,
			dbat.battery_model_seq_no AS battery_model_seq_no
		`).
		Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
		Where("d.tenant_id = ? AND d.id IN ?", claims.TenantID, deviceIDs).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if len(rows) != len(deviceIDs) {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "存在无效的电池记录，请刷新列表后重试"})
	}

	rowMap := make(map[string]batteryInfoCompleteDeviceRow, len(rows))
	for _, row := range rows {
		rowMap[row.DeviceID] = row
	}

	if operatorOrgID != "" {
		for _, deviceID := range deviceIDs {
			if accessErr := checkDeviceOrgAccess(ctx, deviceID, claims.TenantID, operatorOrgID); accessErr != nil {
				return nil, accessErr
			}
		}
	}

	now := time.Now().UTC()
	if err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, deviceID := range deviceIDs {
			row := rowMap[deviceID]

			var exists int64
			if err := tx.Table("device_batteries").Where("device_id = ?", row.DeviceID).Count(&exists).Error; err != nil {
				return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
			}

			updates := map[string]interface{}{
				"cell_brand_seq_no":    req.CellBrandSeqNo,
				"battery_model_seq_no": req.BatteryModelSeqNo,
				"updated_at":           now,
			}

			if exists == 0 {
				insertRow := map[string]interface{}{
					"device_id":            row.DeviceID,
					"cell_brand_seq_no":    req.CellBrandSeqNo,
					"battery_model_seq_no": req.BatteryModelSeqNo,
					"updated_at":           now,
				}
				if operatorOrgID != "" {
					insertRow["pack_factory_org_id"] = operatorOrgID
				}
				if err := tx.Table("device_batteries").Create(insertRow).Error; err != nil {
					return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
				}
			} else {
				if operatorOrgID != "" && row.PackFactoryOrgID == nil {
					updates["pack_factory_org_id"] = operatorOrgID
				}
				if err := tx.Table("device_batteries").Where("device_id = ?", row.DeviceID).Updates(updates).Error; err != nil {
					return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
				}
			}

			desc := fmt.Sprintf("信息补全：电芯品牌=%s（%d），电池型号=%s（%d）", cellBrand.Name, cellBrand.SeqNo, packModel.Name, req.BatteryModelSeqNo)
			extra := map[string]any{
				"old_cell_brand_seq_no":    row.CellBrandSeqNo,
				"new_cell_brand_seq_no":    req.CellBrandSeqNo,
				"old_battery_model_seq_no": row.BatteryModelSeqNo,
				"new_battery_model_seq_no": req.BatteryModelSeqNo,
				"cell_brand_name":          cellBrand.Name,
				"battery_model_name":       packModel.Name,
			}
			_ = CreateBatteryOperationLogTx(tx, claims.TenantID, row.DeviceID, row.DeviceNumber, BatteryOpTypeInfoComplete, &claims.ID, &desc, extra)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &model.BatteryCompleteInfoResp{
		Total:   len(deviceIDs),
		Success: len(deviceIDs),
	}, nil
}
