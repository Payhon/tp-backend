package service

import (
	"context"

	"time"

	"project/internal/model"
	"project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/go-basic/uuid"
	"gorm.io/gorm"
)

type Dealer struct{}

// GetDealerOverview 经销商穿透概览（聚合数字）
func (*Dealer) GetDealerOverview(ctx context.Context, id string, claims *utils.UserClaims, dealerScopeID string) (*model.DealerOverviewResp, error) {
	// 经销商账号只能看自己
	if dealerScopeID != "" && dealerScopeID != id {
		return nil, errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{
			"message": "no permission",
		})
	}

	// 校验经销商存在且属于租户
	dealer, err := query.Dealer.Where(
		query.Dealer.ID.Eq(id),
		query.Dealer.TenantID.Eq(claims.TenantID),
	).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(404)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	// 设备数量/激活数量（沿用详情口径）
	deviceCount, _ := query.DeviceBattery.Where(query.DeviceBattery.DealerID.Eq(id)).Count()
	activeCount, _ := query.DeviceBattery.Where(
		query.DeviceBattery.DealerID.Eq(id),
		query.DeviceBattery.ActivationStatus.Eq("ACTIVE"),
	).Count()

	// 终端用户数量（绑定关系 distinct user）
	var endUserCount int64
	if err := global.DB.Table("device_user_bindings AS dub").
		Joins("LEFT JOIN devices AS d ON d.id = dub.device_id").
		Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
		Where("d.tenant_id = ?", claims.TenantID).
		Where("dbat.dealer_id = ?", id).
		Distinct("dub.user_id").
		Count(&endUserCount).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	// 维保统计（按状态）
	type statusRow struct {
		Status string `gorm:"column:status"`
		Cnt    int64  `gorm:"column:cnt"`
	}
	var statusRows []statusRow
	if err := global.DB.Table("warranty_applications AS wa").
		Joins("LEFT JOIN devices AS d ON d.id = wa.device_id").
		Joins("LEFT JOIN device_batteries AS dbat ON dbat.device_id = d.id").
		Where("wa.tenant_id = ?", claims.TenantID).
		Where("dbat.dealer_id = ?", id).
		Select("wa.status AS status, COUNT(1) AS cnt").
		Group("wa.status").
		Scan(&statusRows).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	var total, pending, approved, rejected, processing, completed int64
	for _, r := range statusRows {
		total += r.Cnt
		switch r.Status {
		case "PENDING":
			pending = r.Cnt
		case "APPROVED":
			approved = r.Cnt
		case "REJECTED":
			rejected = r.Cnt
		case "PROCESSING":
			processing = r.Cnt
		case "COMPLETED":
			completed = r.Cnt
		}
	}

	return &model.DealerOverviewResp{
		DealerID:           dealer.ID,
		DealerName:         dealer.Name,
		DeviceCount:        deviceCount,
		ActiveCount:        activeCount,
		EndUserCount:       endUserCount,
		WarrantyTotal:      total,
		WarrantyPending:    pending,
		WarrantyApproved:   approved,
		WarrantyRejected:   rejected,
		WarrantyProcessing: processing,
		WarrantyCompleted:  completed,
	}, nil
}

// CreateDealer 创建经销商
func (*Dealer) CreateDealer(req model.DealerCreateReq, claims *utils.UserClaims) (*model.Dealer, error) {
	t := time.Now().UTC()

	dealer := &model.Dealer{
		ID:            uuid.New(),
		Name:          req.Name,
		ContactPerson: req.ContactPerson,
		Phone:         req.Phone,
		Email:         req.Email,
		Province:      req.Province,
		City:          req.City,
		District:      req.District,
		Address:       req.Address,
		ParentID:      req.ParentID,
		TenantID:      claims.TenantID,
		CreatedAt:     &t,
		UpdatedAt:     &t,
		Remark:        req.Remark,
	}

	// 验证父经销商是否存在
	if req.ParentID != nil && *req.ParentID != "" {
		parentDealer, err := query.Dealer.Where(query.Dealer.ID.Eq(*req.ParentID)).First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
					"message": "parent dealer not found",
				})
			}
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"sql_error": err.Error(),
			})
		}
		// 确保父经销商属于同一租户
		if parentDealer.TenantID != claims.TenantID {
			return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"message": "parent dealer not in same tenant",
			})
		}
	}

	// 创建经销商
	if err := query.Dealer.Create(dealer); err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return dealer, nil
}

// UpdateDealer 更新经销商
func (*Dealer) UpdateDealer(id string, req model.DealerUpdateReq, claims *utils.UserClaims) (*model.Dealer, error) {
	t := time.Now().UTC()

	// 查询经销商是否存在
	dealer, err := query.Dealer.Where(
		query.Dealer.ID.Eq(id),
		query.Dealer.TenantID.Eq(claims.TenantID),
	).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(404)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ContactPerson != nil {
		updates["contact_person"] = *req.ContactPerson
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Province != nil {
		updates["province"] = *req.Province
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.District != nil {
		updates["district"] = *req.District
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	updates["updated_at"] = t

	// 执行更新
	if _, err := query.Dealer.Where(query.Dealer.ID.Eq(id)).Updates(updates); err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 重新查询返回
	dealer, err = query.Dealer.Where(query.Dealer.ID.Eq(id)).First()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return dealer, nil
}

// DeleteDealer 删除经销商
func (*Dealer) DeleteDealer(id string, claims *utils.UserClaims) error {
	// 检查经销商是否存在
	dealer, err := query.Dealer.Where(
		query.Dealer.ID.Eq(id),
		query.Dealer.TenantID.Eq(claims.TenantID),
	).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.New(404)
		}
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 检查是否有下级经销商
	count, err := query.Dealer.Where(query.Dealer.ParentID.Eq(dealer.ID)).Count()
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	if count > 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "dealer has sub dealers, please delete them first",
		})
	}

	// 检查是否有关联设备
	deviceCount, err := query.DeviceBattery.Where(query.DeviceBattery.DealerID.Eq(id)).Count()
	if err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	if deviceCount > 0 {
		return errcode.WithData(errcode.CodeParamError, map[string]interface{}{
			"message": "dealer has devices, please transfer them first",
		})
	}

	// 删除经销商
	if _, err := query.Dealer.Where(query.Dealer.ID.Eq(id)).Delete(); err != nil {
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	return nil
}

// GetDealerByID 获取经销商详情
func (*Dealer) GetDealerByID(id string, claims *utils.UserClaims) (*model.DealerResp, error) {
	dealer, err := query.Dealer.Where(
		query.Dealer.ID.Eq(id),
		query.Dealer.TenantID.Eq(claims.TenantID),
	).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errcode.New(404)
		}
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 统计设备数量
	deviceCount, _ := query.DeviceBattery.Where(query.DeviceBattery.DealerID.Eq(id)).Count()
	activeCount, _ := query.DeviceBattery.Where(
		query.DeviceBattery.DealerID.Eq(id),
		query.DeviceBattery.ActivationStatus.Eq("ACTIVE"),
	).Count()

	resp := &model.DealerResp{
		ID:            dealer.ID,
		Name:          dealer.Name,
		ContactPerson: dealer.ContactPerson,
		Phone:         dealer.Phone,
		Email:         dealer.Email,
		Province:      dealer.Province,
		City:          dealer.City,
		District:      dealer.District,
		Address:       dealer.Address,
		ParentID:      dealer.ParentID,
		DeviceCount:   deviceCount,
		ActiveCount:   activeCount,
		CreatedAt:     dealer.CreatedAt.Format("2006-01-02 15:04:05"),
		Remark:        dealer.Remark,
	}

	return resp, nil
}

// GetDealerList 获取经销商列表
func (*Dealer) GetDealerList(req model.DealerListReq, claims *utils.UserClaims) (*model.DealerListResp, error) {
	q := query.Dealer
	queryBuilder := q.Where(q.TenantID.Eq(claims.TenantID))

	// 条件筛选
	if req.Name != nil && *req.Name != "" {
		queryBuilder = queryBuilder.Where(q.Name.Like("%" + *req.Name + "%"))
	}
	if req.Phone != nil && *req.Phone != "" {
		queryBuilder = queryBuilder.Where(q.Phone.Like("%" + *req.Phone + "%"))
	}
	if req.Province != nil && *req.Province != "" {
		queryBuilder = queryBuilder.Where(q.Province.Eq(*req.Province))
	}
	if req.City != nil && *req.City != "" {
		queryBuilder = queryBuilder.Where(q.City.Eq(*req.City))
	}

	// 统计总数
	total, err := queryBuilder.Count()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	dealers, err := queryBuilder.Offset(offset).Limit(req.PageSize).Order(q.CreatedAt.Desc()).Find()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}

	// 构建响应
	list := make([]model.DealerResp, 0, len(dealers))
	for _, dealer := range dealers {
		// 统计设备数量
		deviceCount, _ := query.DeviceBattery.Where(query.DeviceBattery.DealerID.Eq(dealer.ID)).Count()
		activeCount, _ := query.DeviceBattery.Where(
			query.DeviceBattery.DealerID.Eq(dealer.ID),
			query.DeviceBattery.ActivationStatus.Eq("ACTIVE"),
		).Count()

		list = append(list, model.DealerResp{
			ID:            dealer.ID,
			Name:          dealer.Name,
			ContactPerson: dealer.ContactPerson,
			Phone:         dealer.Phone,
			Email:         dealer.Email,
			Province:      dealer.Province,
			City:          dealer.City,
			District:      dealer.District,
			Address:       dealer.Address,
			ParentID:      dealer.ParentID,
			DeviceCount:   deviceCount,
			ActiveCount:   activeCount,
			CreatedAt:     dealer.CreatedAt.Format("2006-01-02 15:04:05"),
			Remark:        dealer.Remark,
		})
	}

	return &model.DealerListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
