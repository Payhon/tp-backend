package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

type BatteryModel struct{}

func normalizeBatteryModelName(name string) (string, error) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "型号不能为空"})
	}
	if utf8.RuneCountInString(n) > 64 {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "型号长度不能超过64个字符"})
	}
	return n, nil
}

func getOperatorOrgID(claims *utils.UserClaims) (string, error) {
	orgID, err := getUserOrgID(claims.ID)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(orgID), nil
}

func isAdminUser(claims *utils.UserClaims) bool {
	if claims == nil {
		return false
	}
	a := strings.TrimSpace(strings.ToUpper(claims.Authority))
	return a == "TENANT_ADMIN" || a == "SYS_ADMIN"
}

func canOperateBatteryModel(claims *utils.UserClaims, operatorOrgID string, bm *model.BatteryPackModel) bool {
	if bm == nil {
		return false
	}
	if isAdminUser(claims) {
		return true
	}
	if operatorOrgID == "" || strings.TrimSpace(bm.OrgID) == "" {
		return false
	}
	return strings.TrimSpace(bm.OrgID) == operatorOrgID
}

func resolveBatteryModelOrgID(ctx context.Context, claims *utils.UserClaims, operatorOrgID string, targetOrgID *string) (string, error) {
	if isAdminUser(claims) {
		if targetOrgID == nil || strings.TrimSpace(*targetOrgID) == "" {
			if operatorOrgID != "" {
				return operatorOrgID, nil
			}
			return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "请选择PACK厂家"})
		}
	}

	if !isAdminUser(claims) {
		if operatorOrgID == "" {
			return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "当前账号未绑定机构，无法操作电池型号"})
		}
		return operatorOrgID, nil
	}

	orgID := strings.TrimSpace(*targetOrgID)
	var row struct {
		ID      string `gorm:"column:id"`
		OrgType string `gorm:"column:org_type"`
	}
	if err := global.DB.WithContext(ctx).
		Table("orgs").
		Select("id, org_type").
		Where("id = ? AND tenant_id = ?", orgID, claims.TenantID).
		Limit(1).
		Scan(&row).Error; err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if row.ID == "" {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "所选机构不存在"})
	}
	if strings.TrimSpace(strings.ToUpper(row.OrgType)) != model.OrgTypePACKFactory {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "仅支持选择PACK厂家"})
	}
	return orgID, nil
}

func (*BatteryModel) checkCreateConstraints(ctx context.Context, tenantID, orgID, name string, seqNo int16) error {
	var nameExists int64
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryPackModel).
		Where("tenant_id = ? AND org_id = ? AND name = ?", tenantID, orgID, name).
		Count(&nameExists).Error; err != nil {
		return err
	}
	if nameExists > 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "型号已存在"})
	}

	var seqExists int64
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryPackModel).
		Where("tenant_id = ? AND org_id = ? AND seq_no = ?", tenantID, orgID, seqNo).
		Count(&seqExists).Error; err != nil {
		return err
	}
	if seqExists > 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "序号已存在"})
	}
	return nil
}

func (*BatteryModel) checkUpdateConstraints(ctx context.Context, tenantID, orgID, id string, name *string, seqNo *int16) error {
	if name != nil {
		var nameExists int64
		if err := global.DB.WithContext(ctx).
			Table(model.TableNameBatteryPackModel).
			Where("tenant_id = ? AND org_id = ? AND name = ? AND id <> ?", tenantID, orgID, *name, id).
			Count(&nameExists).Error; err != nil {
			return err
		}
		if nameExists > 0 {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "型号已存在"})
		}
	}

	if seqNo != nil {
		var seqExists int64
		if err := global.DB.WithContext(ctx).
			Table(model.TableNameBatteryPackModel).
			Where("tenant_id = ? AND org_id = ? AND seq_no = ? AND id <> ?", tenantID, orgID, *seqNo, id).
			Count(&seqExists).Error; err != nil {
			return err
		}
		if seqExists > 0 {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "序号已存在"})
		}
	}
	return nil
}

// CreateBatteryModel 创建电池型号
func (*BatteryModel) CreateBatteryModel(req model.BatteryModelCreateReq, claims *utils.UserClaims) (*model.BatteryPackModel, error) {
	ctx := context.Background()
	name, err := normalizeBatteryModelName(req.Name)
	if err != nil {
		return nil, err
	}

	orgID, err := getOperatorOrgID(claims)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	targetOrgID, err := resolveBatteryModelOrgID(ctx, claims, orgID, req.OrgID)
	if err != nil {
		return nil, err
	}

	if err := (&BatteryModel{}).checkCreateConstraints(ctx, claims.TenantID, targetOrgID, name, req.SeqNo); err != nil {
		if ec, ok := err.(*errcode.Error); ok {
			return nil, ec
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	now := time.Now().UTC()
	bm := &model.BatteryPackModel{
		ID:        uuid.New(),
		SeqNo:     req.SeqNo,
		Name:      name,
		OrgID:     targetOrgID,
		TenantID:  claims.TenantID,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	if err := global.DB.WithContext(ctx).Create(bm).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return bm, nil
}

// UpdateBatteryModel 更新电池型号
func (*BatteryModel) UpdateBatteryModel(id string, req model.BatteryModelUpdateReq, claims *utils.UserClaims) (*model.BatteryPackModel, error) {
	ctx := context.Background()
	var bm model.BatteryPackModel
	if err := global.DB.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		First(&bm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(404)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	operatorOrgID, err := getOperatorOrgID(claims)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if !isAdminUser(claims) && operatorOrgID == "" {
		return nil, errcode.New(errcode.CodeNoPermission)
	}
	if !canOperateBatteryModel(claims, operatorOrgID, &bm) {
		return nil, errcode.New(errcode.CodeNoPermission)
	}

	updates := map[string]interface{}{}
	targetOrgID := strings.TrimSpace(bm.OrgID)
	if req.OrgID != nil {
		resolvedOrgID, resolveErr := resolveBatteryModelOrgID(ctx, claims, operatorOrgID, req.OrgID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		targetOrgID = resolvedOrgID
		if targetOrgID != strings.TrimSpace(bm.OrgID) {
			updates["org_id"] = targetOrgID
		}
	}
	var normalizedName *string
	if req.Name != nil {
		n, nErr := normalizeBatteryModelName(*req.Name)
		if nErr != nil {
			return nil, nErr
		}
		normalizedName = &n
		updates["name"] = n
	}
	if req.SeqNo != nil {
		updates["seq_no"] = *req.SeqNo
	}

	if err := (&BatteryModel{}).checkUpdateConstraints(ctx, claims.TenantID, targetOrgID, id, normalizedName, req.SeqNo); err != nil {
		if ec, ok := err.(*errcode.Error); ok {
			return nil, ec
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if len(updates) == 0 {
		return &bm, nil
	}

	updates["updated_at"] = time.Now().UTC()
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryPackModel).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		Updates(updates).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if err := global.DB.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		First(&bm).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return &bm, nil
}

// DeleteBatteryModel 删除电池型号
func (*BatteryModel) DeleteBatteryModel(id string, claims *utils.UserClaims) error {
	ctx := context.Background()
	var bm model.BatteryPackModel
	if err := global.DB.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		First(&bm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.New(404)
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	operatorOrgID, err := getOperatorOrgID(claims)
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if !canOperateBatteryModel(claims, operatorOrgID, &bm) {
		return errcode.New(errcode.CodeNoPermission)
	}

	var deviceCount int64
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameDeviceBattery).
		Where("battery_model_id = ?", id).
		Count(&deviceCount).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if deviceCount > 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "battery model has devices, cannot delete"})
	}

	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryPackModel).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		Delete(&model.BatteryPackModel{}).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return nil
}

// GetBatteryModelByID 获取电池型号详情
func (*BatteryModel) GetBatteryModelByID(id string, claims *utils.UserClaims) (*model.BatteryModelResp, error) {
	type row struct {
		ID          string     `gorm:"column:id"`
		SeqNo       *int16     `gorm:"column:seq_no"`
		Name        string     `gorm:"column:name"`
		OrgID       *string    `gorm:"column:org_id"`
		OrgName     *string    `gorm:"column:org_name"`
		CreatedAt   *time.Time `gorm:"column:created_at"`
		DeviceCount int64      `gorm:"column:device_count"`
	}

	ctx := context.Background()
	operatorOrgID, err := getOperatorOrgID(claims)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if !isAdminUser(claims) && operatorOrgID == "" {
		return nil, errcode.New(errcode.CodeNoPermission)
	}

	q := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryPackModel+" AS bm").
		Select(`
			bm.id,
			bm.seq_no,
			bm.name,
			bm.org_id,
			org.name AS org_name,
			bm.created_at,
			COALESCE(cnt.device_count, 0) AS device_count
		`).
		Joins("LEFT JOIN orgs org ON org.id = bm.org_id").
		Joins("LEFT JOIN (SELECT battery_model_id, COUNT(*) AS device_count FROM device_batteries GROUP BY battery_model_id) cnt ON cnt.battery_model_id = bm.id").
		Where("bm.id = ? AND bm.tenant_id = ?", id, claims.TenantID)

	if operatorOrgID != "" {
		q = q.Where("bm.org_id = ?", operatorOrgID)
	}

	var r row
	if err := q.Limit(1).Scan(&r).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if r.ID == "" {
		return nil, errcode.New(404)
	}

	createdAt := ""
	if r.CreatedAt != nil {
		createdAt = r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05")
	}

	return &model.BatteryModelResp{
		ID:          r.ID,
		SeqNo:       r.SeqNo,
		Name:        r.Name,
		OrgID:       r.OrgID,
		OrgName:     r.OrgName,
		DeviceCount: r.DeviceCount,
		CreatedAt:   createdAt,
	}, nil
}

// GetBatteryModelList 获取电池型号列表
func (*BatteryModel) GetBatteryModelList(req model.BatteryModelListReq, claims *utils.UserClaims) (*model.BatteryModelListResp, error) {
	type row struct {
		ID          string     `gorm:"column:id"`
		SeqNo       *int16     `gorm:"column:seq_no"`
		Name        string     `gorm:"column:name"`
		OrgID       *string    `gorm:"column:org_id"`
		OrgName     *string    `gorm:"column:org_name"`
		CreatedAt   *time.Time `gorm:"column:created_at"`
		DeviceCount int64      `gorm:"column:device_count"`
	}

	ctx := context.Background()
	operatorOrgID, err := getOperatorOrgID(claims)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	q := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryPackModel+" AS bm").
		Select(`
			bm.id,
			bm.seq_no,
			bm.name,
			bm.org_id,
			org.name AS org_name,
			bm.created_at,
			COALESCE(cnt.device_count, 0) AS device_count
		`).
		Joins("LEFT JOIN orgs org ON org.id = bm.org_id").
		Joins("LEFT JOIN (SELECT battery_model_id, COUNT(*) AS device_count FROM device_batteries GROUP BY battery_model_id) cnt ON cnt.battery_model_id = bm.id").
		Where("bm.tenant_id = ?", claims.TenantID)

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		q = q.Where("bm.name ILIKE ?", "%"+strings.TrimSpace(*req.Name)+"%")
	}

	if operatorOrgID != "" {
		q = q.Where("bm.org_id = ?", operatorOrgID)
	} else if req.OrgID != nil && strings.TrimSpace(*req.OrgID) != "" {
		q = q.Where("bm.org_id = ?", strings.TrimSpace(*req.OrgID))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = int(total)
		if pageSize < 1 {
			pageSize = 1000
		}
	}

	var rows []row
	if err := q.
		Order("bm.seq_no ASC").
		Order("bm.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.BatteryModelResp, 0, len(rows))
	for _, r := range rows {
		createdAt := ""
		if r.CreatedAt != nil {
			createdAt = r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05")
		}
		list = append(list, model.BatteryModelResp{
			ID:          r.ID,
			SeqNo:       r.SeqNo,
			Name:        r.Name,
			OrgID:       r.OrgID,
			OrgName:     r.OrgName,
			DeviceCount: r.DeviceCount,
			CreatedAt:   createdAt,
		})
	}

	return &model.BatteryModelListResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
