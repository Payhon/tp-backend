package service

import (
	"context"
	"strings"
	"time"

	"project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BatteryTag 电池标签服务
type BatteryTag struct{}

func (*BatteryTag) Create(ctx context.Context, req model.BatteryTagCreateReq, claims *utils.UserClaims) error {
	now := time.Now().UTC()
	tag := &model.BatteryTag{
		ID:        uuid.New(),
		TenantID:  claims.TenantID,
		Name:      strings.TrimSpace(req.Name),
		Color:     req.Color,
		Scene:     req.Scene,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if tag.Name == "" {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"name": "required"})
	}
	if err := global.DB.WithContext(ctx).Create(tag).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func (*BatteryTag) Update(ctx context.Context, id string, req model.BatteryTagUpdateReq, claims *utils.UserClaims) error {
	updates := map[string]interface{}{}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{"name": "invalid"})
		}
		updates["name"] = n
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}
	if req.Scene != nil {
		updates["scene"] = *req.Scene
	}
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now().UTC()

	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryTag).
		Where("tenant_id = ? AND id = ?", claims.TenantID, id).
		Updates(updates).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func (*BatteryTag) Delete(ctx context.Context, id string, claims *utils.UserClaims) error {
	if err := global.DB.WithContext(ctx).
		Table(model.TableNameBatteryTag).
		Where("tenant_id = ? AND id = ?", claims.TenantID, id).
		Delete(&model.BatteryTag{}).Error; err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}
	return nil
}

func (*BatteryTag) List(ctx context.Context, req model.BatteryTagListReq, claims *utils.UserClaims) (*model.BatteryTagListResp, error) {
	type row struct {
		ID          string    `gorm:"column:id"`
		Name        string    `gorm:"column:name"`
		Color       *string   `gorm:"column:color"`
		Scene       *string   `gorm:"column:scene"`
		CreatedAt   time.Time `gorm:"column:created_at"`
		DeviceCount int64     `gorm:"column:device_count"`
	}

	q := global.DB.WithContext(ctx).Table("battery_tags bt").
		Select(`
bt.id, bt.name, bt.color, bt.scene, bt.created_at,
COALESCE(COUNT(DISTINCT dbt.device_id), 0) AS device_count
`).
		Joins("LEFT JOIN device_battery_tags dbt ON dbt.tag_id = bt.id").
		Where("bt.tenant_id = ?", claims.TenantID).
		Group("bt.id").
		Order("bt.created_at DESC")

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		q = q.Where("bt.name ILIKE ?", "%"+strings.TrimSpace(*req.Name)+"%")
	}
	if req.Scene != nil && strings.TrimSpace(*req.Scene) != "" {
		q = q.Where("bt.scene ILIKE ?", "%"+strings.TrimSpace(*req.Scene)+"%")
	}

	var total int64
	// count tags (without join/group)
	if err := global.DB.WithContext(ctx).
		Table("battery_tags").
		Where("tenant_id = ?", claims.TenantID).
		Scopes(func(db *gorm.DB) *gorm.DB {
			if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
				db = db.Where("name ILIKE ?", "%"+strings.TrimSpace(*req.Name)+"%")
			}
			if req.Scene != nil && strings.TrimSpace(*req.Scene) != "" {
				db = db.Where("scene ILIKE ?", "%"+strings.TrimSpace(*req.Scene)+"%")
			}
			return db
		}).
		Count(&total).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	offset := (req.Page - 1) * req.PageSize
	var rows []row
	if err := q.Limit(req.PageSize).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	out := make([]model.BatteryTagItemResp, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.BatteryTagItemResp{
			ID:          r.ID,
			Name:        r.Name,
			Color:       r.Color,
			Scene:       r.Scene,
			DeviceCount: r.DeviceCount,
			CreatedAt:   r.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
		})
	}

	return &model.BatteryTagListResp{
		List:     out,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (*BatteryTag) Assign(ctx context.Context, req model.BatteryTagAssignReq, claims *utils.UserClaims, dealerScopeID string) error {
	mode := "REPLACE"
	if req.Mode != nil && *req.Mode != "" {
		mode = *req.Mode
	}

	// 经销商范围校验：只能给名下设备打标签
	if dealerScopeID != "" {
		var cnt int64
		if err := global.DB.WithContext(ctx).
			Table("device_batteries").
			Where("dealer_id = ?", dealerScopeID).
			Where("device_id IN ?", req.DeviceIDs).
			Count(&cnt).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
		if cnt != int64(len(req.DeviceIDs)) {
			return errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "device out of dealer scope"})
		}
	}

	// 先清理（REPLACE）再写入
	if mode == "REPLACE" {
		if err := global.DB.WithContext(ctx).
			Table("device_battery_tags").
			Where("tenant_id = ?", claims.TenantID).
			Where("device_id IN ?", req.DeviceIDs).
			Delete(&model.DeviceBatteryTag{}).Error; err != nil {
			return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
		}
	}

	if len(req.TagIDs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	rows := make([]model.DeviceBatteryTag, 0, len(req.DeviceIDs)*len(req.TagIDs))
	for _, did := range req.DeviceIDs {
		for _, tid := range req.TagIDs {
			rows = append(rows, model.DeviceBatteryTag{
				ID:        uuid.New(),
				TenantID:  claims.TenantID,
				DeviceID:  did,
				TagID:     tid,
				CreatedAt: now,
			})
		}
	}

	if err := global.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "device_id"}, {Name: "tag_id"}},
			DoNothing: true,
		}).
		CreateInBatches(rows, 200).Error; err != nil {
		logrus.WithError(err).Error("assign battery tags failed")
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	return nil
}

