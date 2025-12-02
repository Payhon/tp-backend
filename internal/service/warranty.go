package service

import (
	"context"
	"encoding/json"
	"time"

	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

// Warranty 维保管理服务
type Warranty struct{}

// CreateWarrantyApplication 创建维保申请
func (*Warranty) CreateWarrantyApplication(req model.WarrantyApplicationCreateReq, claims *utils.UserClaims) (*model.WarrantyApplicationResp, error) {
	ctx := context.Background()

	// 校验设备合法性（存在且属于当前租户）
	device, err := query.Device.WithContext(ctx).
		Where(
			query.Device.ID.Eq(req.DeviceID),
			query.Device.TenantID.Eq(claims.TenantID),
		).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "device not found",
			})
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 组装图片JSON
	var imagesJSON *string
	if len(req.Images) > 0 {
		if b, err := json.Marshal(req.Images); err == nil {
			s := string(b)
			imagesJSON = &s
		} else {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "images format invalid",
			})
		}
	}

	status := "PENDING"
	now := time.Now().UTC()

	app := &model.WarrantyApplication{
		ID:          uuid.New(),
		DeviceID:    req.DeviceID,
		UserID:      claims.ID,
		Type:        &req.Type,
		Description: req.Description,
		Image:       imagesJSON,
		Status:      &status,
		TenantID:    claims.TenantID,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}

	if err := query.WarrantyApplication.WithContext(ctx).Create(app); err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 构建详情响应
	resp, err := buildWarrantyResp(ctx, app, device, claims.TenantID)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// UpdateWarrantyStatus 更新维保申请状态
func (*Warranty) UpdateWarrantyStatus(id string, req model.WarrantyApplicationUpdateReq, claims *utils.UserClaims) error {
	ctx := context.Background()

	// 查询记录是否存在且属于当前租户
	app, err := query.WarrantyApplication.WithContext(ctx).
		Where(
			query.WarrantyApplication.ID.Eq(id),
			query.WarrantyApplication.TenantID.Eq(claims.TenantID),
		).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.New(404)
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	updates := make(map[string]interface{})

	// 状态更新
	if req.Status != nil && *req.Status != "" {
		updates["status"] = *req.Status
	}

	// 处理结果信息
	if req.ResultInfo != nil {
		if b, err := json.Marshal(req.ResultInfo); err == nil {
			updates["result_info"] = string(b)
		} else {
			return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "result_info format invalid",
			})
		}
	}

	// 处理人
	if req.HandlerID != nil && *req.HandlerID != "" {
		updates["handler_id"] = *req.HandlerID
	} else if app.HandlerID == nil || *app.HandlerID == "" {
		// 如未指定处理人且原记录无处理人，默认当前用户为处理人
		updates["handler_id"] = claims.ID
	}

	updates["updated_at"] = time.Now().UTC()

	if len(updates) == 0 {
		// 没有可更新字段，直接返回
		return nil
	}

	if _, err := query.WarrantyApplication.WithContext(ctx).
		Where(
			query.WarrantyApplication.ID.Eq(id),
			query.WarrantyApplication.TenantID.Eq(claims.TenantID),
		).
		Updates(updates); err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return nil
}

// GetWarrantyList 查询维保申请列表
func (*Warranty) GetWarrantyList(req model.WarrantyApplicationListReq, claims *utils.UserClaims) (*model.WarrantyApplicationListResp, error) {
	ctx := context.Background()

	wq := query.WarrantyApplication.WithContext(ctx).
		Where(query.WarrantyApplication.TenantID.Eq(claims.TenantID))

	// 按设备编号筛选 -> 先获取对应设备ID
	if req.DeviceNumber != nil && *req.DeviceNumber != "" {
		devices, err := query.Device.WithContext(ctx).
			Where(
				query.Device.TenantID.Eq(claims.TenantID),
				query.Device.DeviceNumber.Like("%"+*req.DeviceNumber+"%"),
			).
			Find()
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}
		if len(devices) == 0 {
			return &model.WarrantyApplicationListResp{
				List:     []model.WarrantyApplicationResp{},
				Total:    0,
				Page:     req.Page,
				PageSize: req.PageSize,
			}, nil
		}

		deviceIDs := make([]string, 0, len(devices))
		for _, d := range devices {
			deviceIDs = append(deviceIDs, d.ID)
		}
		wq = wq.Where(query.WarrantyApplication.DeviceID.In(deviceIDs...))
	}

	if req.UserID != nil && *req.UserID != "" {
		wq = wq.Where(query.WarrantyApplication.UserID.Eq(*req.UserID))
	}

	if req.Type != nil && *req.Type != "" {
		wq = wq.Where(query.WarrantyApplication.Type.Eq(*req.Type))
	}

	if req.Status != nil && *req.Status != "" {
		wq = wq.Where(query.WarrantyApplication.Status.Eq(*req.Status))
	}

	// 时间范围
	if req.StartTime != nil && *req.StartTime != "" {
		if start, err := time.Parse("2006-01-02 15:04:05", *req.StartTime); err == nil {
			wq = wq.Where(query.WarrantyApplication.CreatedAt.Gte(start))
		}
	}
	if req.EndTime != nil && *req.EndTime != "" {
		if end, err := time.Parse("2006-01-02 15:04:05", *req.EndTime); err == nil {
			wq = wq.Where(query.WarrantyApplication.CreatedAt.Lte(end))
		}
	}

	// 统计总数
	total, err := wq.Count()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	offset := (req.Page - 1) * req.PageSize
	apps, err := wq.
		Offset(offset).
		Limit(req.PageSize).
		Order(query.WarrantyApplication.CreatedAt.Desc()).
		Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	if len(apps) == 0 {
		return &model.WarrantyApplicationListResp{
			List:     []model.WarrantyApplicationResp{},
			Total:    total,
			Page:     req.Page,
			PageSize: req.PageSize,
		}, nil
	}

	// 收集设备ID和用户ID
	deviceIDs := make(map[string]struct{})
	userIDs := make(map[string]struct{})
	for _, a := range apps {
		deviceIDs[a.DeviceID] = struct{}{}
		userIDs[a.UserID] = struct{}{}
		if a.HandlerID != nil && *a.HandlerID != "" {
			userIDs[*a.HandlerID] = struct{}{}
		}
	}

	deviceIDList := make([]string, 0, len(deviceIDs))
	for id := range deviceIDs {
		deviceIDList = append(deviceIDList, id)
	}

	userIDList := make([]string, 0, len(userIDs))
	for id := range userIDs {
		userIDList = append(userIDList, id)
	}

	devices, err := query.Device.WithContext(ctx).
		Where(
			query.Device.ID.In(deviceIDList...),
			query.Device.TenantID.Eq(claims.TenantID),
		).
		Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	users, err := query.User.WithContext(ctx).
		Where(query.User.ID.In(userIDList...)).
		Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	deviceMap := make(map[string]*model.Device, len(devices))
	for _, d := range devices {
		deviceMap[d.ID] = d
	}

	userMap := make(map[string]*model.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 组装响应
	list := make([]model.WarrantyApplicationResp, 0, len(apps))
	for _, a := range apps {
		resp, err := buildWarrantyRespWithMaps(ctx, a, deviceMap, userMap)
		if err != nil {
			return nil, err
		}
		list = append(list, *resp)
	}

	return &model.WarrantyApplicationListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetWarrantyDetail 获取维保申请详情
func (*Warranty) GetWarrantyDetail(id string, claims *utils.UserClaims) (*model.WarrantyApplicationResp, error) {
	ctx := context.Background()

	app, err := query.WarrantyApplication.WithContext(ctx).
		Where(
			query.WarrantyApplication.ID.Eq(id),
			query.WarrantyApplication.TenantID.Eq(claims.TenantID),
		).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(404)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	device, err := query.Device.WithContext(ctx).
		Where(
			query.Device.ID.Eq(app.DeviceID),
			query.Device.TenantID.Eq(claims.TenantID),
		).
		First()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	resp, err := buildWarrantyResp(ctx, app, device, claims.TenantID)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// buildWarrantyResp 使用单个设备/用户查询组装维保详情
func buildWarrantyResp(ctx context.Context, app *model.WarrantyApplication, device *model.Device, tenantID string) (*model.WarrantyApplicationResp, error) {
	// 查询申请人信息
	user, err := query.User.WithContext(ctx).
		Where(query.User.ID.Eq(app.UserID)).
		First()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 查询处理人信息（如果有）
	var handler *model.User
	if app.HandlerID != nil && *app.HandlerID != "" {
		handler, err = query.User.WithContext(ctx).
			Where(query.User.ID.Eq(*app.HandlerID)).
			First()
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}
	}

	deviceMap := map[string]*model.Device{
		device.ID: device,
	}
	userMap := map[string]*model.User{
		user.ID: user,
	}
	if handler != nil {
		userMap[handler.ID] = handler
	}

	return buildWarrantyRespWithMaps(ctx, app, deviceMap, userMap)
}

// buildWarrantyRespWithMaps 使用已加载的设备/用户Map组装响应
func buildWarrantyRespWithMaps(_ context.Context, app *model.WarrantyApplication, deviceMap map[string]*model.Device, userMap map[string]*model.User) (*model.WarrantyApplicationResp, error) {
	resp := &model.WarrantyApplicationResp{
		ID:       app.ID,
		DeviceID: app.DeviceID,
		UserID:   app.UserID,
	}

	if app.Type != nil {
		resp.Type = *app.Type
	}
	if app.Description != nil {
		resp.Description = app.Description
	}
	if app.Status != nil {
		resp.Status = *app.Status
	}
	if app.HandlerID != nil && *app.HandlerID != "" {
		resp.HandlerID = app.HandlerID
	}

	// 反序列化图片列表
	if app.Image != nil && *app.Image != "" {
		var images []string
		if err := json.Unmarshal([]byte(*app.Image), &images); err == nil {
			resp.Images = images
		}
	}

	// 反序列化处理结果
	if app.ResultInfo != nil && *app.ResultInfo != "" {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(*app.ResultInfo), &result); err == nil {
			resp.ResultInfo = result
		}
	}

	if app.CreatedAt != nil {
		resp.CreatedAt = app.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if app.UpdatedAt != nil {
		resp.UpdatedAt = app.UpdatedAt.Format("2006-01-02 15:04:05")
	}

	// 设备信息
	if d, ok := deviceMap[app.DeviceID]; ok {
		resp.DeviceNumber = d.DeviceNumber
		if d.Name != nil {
			resp.DeviceName = *d.Name
		}
	}

	// 申请人信息
	if u, ok := userMap[app.UserID]; ok {
		resp.UserName = u.Name
		resp.UserPhone = u.PhoneNumber
	}

	// 处理人信息
	if app.HandlerID != nil && *app.HandlerID != "" {
		if handler, ok := userMap[*app.HandlerID]; ok && handler.Name != nil {
			resp.HandlerName = handler.Name
		}
	}

	return resp, nil
}
