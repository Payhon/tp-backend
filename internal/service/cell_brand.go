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

// CellBrand 电芯品牌服务（租户全局）
type CellBrand struct{}

func normalizeCellBrandName(name string) (string, error) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "品牌名不能为空"})
	}
	if utf8.RuneCountInString(n) > 16 {
		return "", errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "品牌名不能超过16个字符"})
	}
	return n, nil
}

func (*CellBrand) Create(ctx context.Context, req model.BatteryCellBrandCreateReq, claims *utils.UserClaims) (*model.BatteryCellBrand, error) {
	name, err := normalizeCellBrandName(req.Name)
	if err != nil {
		return nil, err
	}

	var seqExists int64
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryCellBrand).
		Where("tenant_id = ? AND seq_no = ?", claims.TenantID, req.SeqNo).
		Count(&seqExists).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if seqExists > 0 {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "序号已存在"})
	}

	var nameExists int64
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryCellBrand).
		Where("tenant_id = ? AND name = ?", claims.TenantID, name).
		Count(&nameExists).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	if nameExists > 0 {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "品牌名已存在"})
	}

	now := time.Now().UTC()
	row := &model.BatteryCellBrand{
		ID:        uuid.New(),
		TenantID:  claims.TenantID,
		SeqNo:     req.SeqNo,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := global.DB.WithContext(ctx).Create(row).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return row, nil
}

func (*CellBrand) Update(ctx context.Context, id string, req model.BatteryCellBrandUpdateReq, claims *utils.UserClaims) (*model.BatteryCellBrand, error) {
	var exists model.BatteryCellBrand
	if err := global.DB.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		First(&exists).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(404)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	updates := map[string]interface{}{}
	if req.SeqNo != nil {
		var seqExists int64
		if err := global.DB.WithContext(ctx).
			Table(model.TableNameBatteryCellBrand).
			Where("tenant_id = ? AND seq_no = ? AND id <> ?", claims.TenantID, *req.SeqNo, id).
			Count(&seqExists).Error; err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if seqExists > 0 {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "序号已存在"})
		}
		updates["seq_no"] = *req.SeqNo
	}

	if req.Name != nil {
		name, err := normalizeCellBrandName(*req.Name)
		if err != nil {
			return nil, err
		}
		var nameExists int64
		if err := global.DB.WithContext(ctx).
			Table(model.TableNameBatteryCellBrand).
			Where("tenant_id = ? AND name = ? AND id <> ?", claims.TenantID, name, id).
			Count(&nameExists).Error; err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if nameExists > 0 {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"message": "品牌名已存在"})
		}
		updates["name"] = name
	}

	if len(updates) == 0 {
		return &exists, nil
	}

	updates["updated_at"] = time.Now().UTC()
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryCellBrand).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		Updates(updates).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	if err := global.DB.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		First(&exists).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return &exists, nil
}

func (*CellBrand) Delete(ctx context.Context, id string, claims *utils.UserClaims) error {
	if err := global.DB.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, claims.TenantID).
		Delete(&model.BatteryCellBrand{}).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func (*CellBrand) List(ctx context.Context, req model.BatteryCellBrandListReq, claims *utils.UserClaims) (*model.BatteryCellBrandListResp, error) {
	type row struct {
		ID        string    `gorm:"column:id"`
		SeqNo     int16     `gorm:"column:seq_no"`
		Name      string    `gorm:"column:name"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}

	q := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryCellBrand).
		Select("id, seq_no, name, created_at").
		Where("tenant_id = ?", claims.TenantID)

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		q = q.Where("name ILIKE ?", "%"+strings.TrimSpace(*req.Name)+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	var rows []row
	if err := q.Order("seq_no ASC").Order("created_at DESC").Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	list := make([]model.BatteryCellBrandItemResp, 0, len(rows))
	for _, r := range rows {
		list = append(list, model.BatteryCellBrandItemResp{
			ID:        r.ID,
			SeqNo:     r.SeqNo,
			Name:      r.Name,
			CreatedAt: r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
		})
	}

	return &model.BatteryCellBrandListResp{List: list, Total: total}, nil
}
