package service

import (
	"context"
	"fmt"

	"project/internal/model"
	query "project/internal/query"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/sirupsen/logrus"
)

const (
	dealerRoleBasic    = "ROLE_DEALER_BASIC"
	dealerRoleAdvanced = "ROLE_DEALER_ADVANCED"
)

func dealerRoleByTemplate(tpl string) (role string, normalized string) {
	switch tpl {
	case "BASIC":
		return dealerRoleBasic, "BASIC"
	case "ADVANCED":
		return dealerRoleAdvanced, "ADVANCED"
	default:
		return "", "NONE"
	}
}

// EnsureDealerTemplatePolicies 确保模板角色的菜单权限存在（按 element_code 映射 sys_ui_elements.id）
func (*Dealer) EnsureDealerTemplatePolicies(ctx context.Context) error {
	// BASIC：经销商侧建议仅开放（看板/电池列表/维保/终端用户）
	basicCodes := []string{
		"bms",
		"bms_dashboard",
		"bms_battery_list",
		"bms_warranty",
		"bms_end_user",
	}
	// ADVANCED：在 BASIC 基础上增加转移记录（如需要可再扩展）
	advancedCodes := []string{
		"bms",
		"bms_dashboard",
		"bms_battery_list",
		"bms_battery_transfer",
		"bms_warranty",
		"bms_end_user",
	}

	getIDs := func(codes []string) ([]string, error) {
		list, err := query.SysUIElement.WithContext(ctx).
			Where(query.SysUIElement.ElementCode.In(codes...)).
			Find()
		if err != nil {
			return nil, err
		}
		m := make(map[string]string, len(list))
		for _, it := range list {
			m[it.ElementCode] = it.ID
		}
		ids := make([]string, 0, len(codes))
		for _, c := range codes {
			id, ok := m[c]
			if !ok || id == "" {
				return nil, fmt.Errorf("sys_ui_elements missing element_code=%s", c)
			}
			ids = append(ids, id)
		}
		return ids, nil
	}

	// 重建两套 role -> function（sys_ui_elements.id）策略
	basicIDs, err := getIDs(basicCodes)
	if err != nil {
		return err
	}
	advancedIDs, err := getIDs(advancedCodes)
	if err != nil {
		return err
	}

	// 先清理旧策略，再写入新策略（保证与模板定义一致）
	GroupApp.Casbin.RemoveRoleAndFunction(dealerRoleBasic)
	GroupApp.Casbin.RemoveRoleAndFunction(dealerRoleAdvanced)

	if ok := GroupApp.Casbin.AddFunctionToRole(dealerRoleBasic, basicIDs); !ok {
		return fmt.Errorf("add policy failed for %s", dealerRoleBasic)
	}
	if ok := GroupApp.Casbin.AddFunctionToRole(dealerRoleAdvanced, advancedIDs); !ok {
		return fmt.Errorf("add policy failed for %s", dealerRoleAdvanced)
	}

	return nil
}

// GetDealerPermissionTemplate 获取经销商当前模板（从其名下任一用户的 g 角色推断）
func (*Dealer) GetDealerPermissionTemplate(ctx context.Context, dealerID string, claims *utils.UserClaims, dealerScopeID string) (*model.DealerPermissionTemplateResp, error) {
	// 经销商账号只能看自己
	if dealerScopeID != "" && dealerScopeID != dealerID {
		return nil, errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "no permission"})
	}

	// 校验经销商归属租户
	_, err := query.Dealer.WithContext(ctx).
		Where(query.Dealer.ID.Eq(dealerID), query.Dealer.TenantID.Eq(claims.TenantID)).
		First()
	if err != nil {
		return nil, err
	}

	var userIDs []string
	if err := global.DB.WithContext(ctx).
		Table("users").
		Select("id").
		Where("tenant_id = ? AND dealer_id = ?", claims.TenantID, dealerID).
		Scan(&userIDs).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	resp := &model.DealerPermissionTemplateResp{
		DealerID:  dealerID,
		Template:  "NONE",
		RoleName:  "",
		UserCount: int64(len(userIDs)),
	}
	if len(userIDs) == 0 {
		return resp, nil
	}

	roles, _ := GroupApp.Casbin.GetRoleFromUser(userIDs[0])
	for _, r := range roles {
		if r == dealerRoleBasic {
			resp.Template = "BASIC"
			resp.RoleName = dealerRoleBasic
			return resp, nil
		}
		if r == dealerRoleAdvanced {
			resp.Template = "ADVANCED"
			resp.RoleName = dealerRoleAdvanced
			return resp, nil
		}
	}
	return resp, nil
}

// SetDealerPermissionTemplate 设置经销商模板：给该经销商名下所有用户绑定模板角色
func (*Dealer) SetDealerPermissionTemplate(ctx context.Context, dealerID string, req model.DealerPermissionTemplateReq, claims *utils.UserClaims, dealerScopeID string) (*model.DealerPermissionTemplateResp, error) {
	// 经销商账号不允许设置（只能由厂家/租户管理员）
	if dealerScopeID != "" {
		return nil, errcode.WithData(errcode.CodeOpDenied, map[string]interface{}{"message": "dealer cannot set template"})
	}

	// 校验经销商归属租户
	_, err := query.Dealer.WithContext(ctx).
		Where(query.Dealer.ID.Eq(dealerID), query.Dealer.TenantID.Eq(claims.TenantID)).
		First()
	if err != nil {
		return nil, err
	}

	// 确保模板策略存在
	if err := GroupApp.Dealer.EnsureDealerTemplatePolicies(ctx); err != nil {
		logrus.WithError(err).Error("ensure dealer template policies failed")
		return nil, errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"message": err.Error()})
	}

	role, normalized := dealerRoleByTemplate(req.Template)
	if role == "" {
		return nil, errcode.WithData(errcode.CodeParamError, map[string]interface{}{"template": "invalid"})
	}

	var userIDs []string
	if err := global.DB.WithContext(ctx).
		Table("users").
		Select("id").
		Where("tenant_id = ? AND dealer_id = ?", claims.TenantID, dealerID).
		Scan(&userIDs).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{"sql_error": err.Error()})
	}

	for _, uid := range userIDs {
		// 仅移除模板角色，保留用户其它角色
		global.CasbinEnforcer.RemoveFilteredNamedGroupingPolicy("g", 0, uid, dealerRoleBasic)
		global.CasbinEnforcer.RemoveFilteredNamedGroupingPolicy("g", 0, uid, dealerRoleAdvanced)
		GroupApp.Casbin.AddRolesToUser(uid, []string{role})
	}

	return &model.DealerPermissionTemplateResp{
		DealerID:  dealerID,
		Template:  normalized,
		RoleName:  role,
		UserCount: int64(len(userIDs)),
	}, nil
}

