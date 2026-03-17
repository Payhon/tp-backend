package service

import (
	"context"
	"strings"
	"time"

	dal "project/internal/dal"
	model "project/internal/model"
	"project/pkg/errcode"
	global "project/pkg/global"
	utils "project/pkg/utils"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
)

type UiElements struct{}

func (*UiElements) CreateUiElements(CreateUiElementsReq *model.CreateUiElementsReq) error {

	var UiElements = model.SysUIElement{}

	UiElements.ID = uuid.New()
	UiElements.ParentID = CreateUiElementsReq.ParentID
	UiElements.ElementCode = CreateUiElementsReq.ElementCode
	UiElements.ElementType = int16(CreateUiElementsReq.ElementType)
	aa := int16(CreateUiElementsReq.Orders)
	UiElements.Order_ = &aa
	UiElements.Param1 = CreateUiElementsReq.Param1
	UiElements.Param2 = CreateUiElementsReq.Param2
	UiElements.Param3 = CreateUiElementsReq.Param3
	UiElements.CreatedAt = time.Now().UTC()
	UiElements.Authority = CreateUiElementsReq.Authority
	UiElements.Description = CreateUiElementsReq.Description
	UiElements.Remark = CreateUiElementsReq.Remark
	UiElements.Multilingual = CreateUiElementsReq.Multilingual
	UiElements.RoutePath = CreateUiElementsReq.RoutePath
	err := dal.CreateUiElements(&UiElements)

	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "create_ui_elements",
			"error":     err.Error(),
		})
	}

	return err
}

func (*UiElements) UpdateUiElements(UpdateUiElementsReq *model.UpdateUiElementsReq) error {
	var UiElements = model.SysUIElement{}
	UiElements.ID = UpdateUiElementsReq.Id
	UiElements.ParentID = *UpdateUiElementsReq.ParentID
	UiElements.ElementCode = *UpdateUiElementsReq.ElementCode
	UiElements.ElementType = *UpdateUiElementsReq.ElementType
	UiElements.Order_ = UpdateUiElementsReq.Orders
	UiElements.Param1 = UpdateUiElementsReq.Param1
	UiElements.Param2 = UpdateUiElementsReq.Param2
	UiElements.Param3 = UpdateUiElementsReq.Param3
	UiElements.Authority = *UpdateUiElementsReq.Authority
	UiElements.Description = UpdateUiElementsReq.Description
	UiElements.Multilingual = UpdateUiElementsReq.Multilingual
	UiElements.RoutePath = UpdateUiElementsReq.RoutePath
	UiElements.Remark = UpdateUiElementsReq.Remark

	err := dal.UpdateUiElements(&UiElements)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "update_ui_elements",
			"error":     err.Error(),
		})
	}
	return err
}

func (*UiElements) DeleteUiElements(id string) error {
	err := dal.DeleteUiElements(id)
	if err != nil {
		logrus.Error(err)
		return errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "delete_ui_elements",
			"error":     err.Error(),
		})
	}
	return err
}

func (*UiElements) ServeUiElementsListByPage(Params *model.ServeUiElementsListByPageReq) (map[string]interface{}, error) {

	total, list, err := dal.ServeUiElementsListByPage(Params)
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_ui_elements",
			"error":     err.Error(),
		})
	}
	UiElementsListRsp := make(map[string]interface{})
	UiElementsListRsp["total"] = total
	UiElementsListRsp["list"] = list

	return UiElementsListRsp, err
}

func (*UiElements) ServeUiElementsListByAuthority(ctx context.Context, u *utils.UserClaims) (map[string]interface{}, error) {
	if u.Authority != "SYS_ADMIN" && strings.TrimSpace(u.ID) != "" {
		account, accountErr := dal.GetUsersById(u.ID)
		if accountErr == nil && account != nil && account.IsMain != nil && *account.IsMain == 0 {
			menuIDs, resolveErr := resolveScopedMenuIDs(ctx, account)
			if resolveErr != nil {
				return nil, resolveErr
			}
			total, list, buildErr := dal.ServeUiElementsListByIDs(menuIDs)
			if buildErr != nil {
				return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
					"operation": "query_ui_elements_by_ids",
					"user_id":   u.ID,
					"error":     buildErr.Error(),
				})
			}
			return map[string]interface{}{
				"total": total,
				"list":  list,
			}, nil
		}
	}

	total, list, err := dal.ServeUiElementsListByAuthority(u)
	if err != nil {
		logrus.Error("[ServeUiElementsListByAuthority] query failed:", err)
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_ui_elements",
			"user_id":   u.ID,
			"error":     err.Error(),
		})
	}

	// 非管理员：若命中机构类型菜单配置，则按配置菜单树直接返回（兼容历史 casbin 仅首页问题）
	if u.Authority != "SYS_ADMIN" && u.Authority != "TENANT_ADMIN" &&
		strings.TrimSpace(u.TenantID) != "" && strings.TrimSpace(u.ID) != "" {
		menuOrgType, ok, resolveErr := resolveMenuOrgType(ctx, strings.TrimSpace(u.TenantID), strings.TrimSpace(u.ID))
		if resolveErr != nil {
			logrus.WithError(resolveErr).WithField("user_id", u.ID).Warn("[ServeUiElementsListByAuthority] resolve menu org type failed")
		} else if ok {
			allowed, exists, allowedErr := GroupApp.OrgTypePermission.GetAllowedUICodes(ctx, u.TenantID, menuOrgType)
			if allowedErr != nil {
				logrus.WithError(allowedErr).WithField("org_type", menuOrgType).Warn("[ServeUiElementsListByAuthority] load allowed ui codes failed")
			} else if exists {
				menuTotal, menuList, menuErr := dal.ServeUiElementsListByCodes(allowed)
				if menuErr != nil {
					logrus.WithError(menuErr).Warn("[ServeUiElementsListByAuthority] build menu tree by ui codes failed")
				} else if len(allowed) > 0 && menuTotal == 0 {
					logrus.WithField("org_type", menuOrgType).Warn("[ServeUiElementsListByAuthority] menu tree is empty by ui codes, fallback to casbin result")
				} else {
					total = menuTotal
					list = menuList
				}
			}
		}
	}

	return map[string]interface{}{
		"total": total,
		"list":  list,
	}, nil
}

func resolveScopedMenuIDs(ctx context.Context, account *model.User) ([]string, error) {
	baseIDs, err := resolveBaseMenuIDs(ctx, account)
	if err != nil {
		return nil, err
	}

	if len(baseIDs) == 0 {
		return []string{}, nil
	}

	roleIDs, _ := GroupApp.Casbin.GetRoleFromUser(account.ID)
	if len(roleIDs) == 0 {
		return []string{}, nil
	}

	var rolePermIDs []string
	if err := global.DB.WithContext(ctx).
		Table("role_permissions").
		Select("permission_key").
		Where("role_id IN ?", roleIDs).
		Scan(&rolePermIDs).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_role_permissions",
			"user_id":   account.ID,
			"error":     err.Error(),
		})
	}

	allowedSet := make(map[string]struct{}, len(rolePermIDs))
	for _, id := range rolePermIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		allowedSet[id] = struct{}{}
	}

	finalIDs := make([]string, 0, len(baseIDs))
	for _, id := range baseIDs {
		if _, ok := allowedSet[id]; ok {
			finalIDs = append(finalIDs, id)
		}
	}
	return finalIDs, nil
}

func resolveBaseMenuIDs(ctx context.Context, account *model.User) ([]string, error) {
	if account == nil {
		return []string{}, nil
	}

	if account.Authority != nil && *account.Authority == "TENANT_ADMIN" {
		var ids []string
		if err := global.DB.WithContext(ctx).
			Table("sys_ui_elements").
			Select("id").
			Where("element_type IN (1,2,3)").
			Where("authority::jsonb ? 'TENANT_ADMIN'").
			Scan(&ids).Error; err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_tenant_admin_menu_ids",
				"user_id":   account.ID,
				"error":     err.Error(),
			})
		}
		return ids, nil
	}

	tenantID := strings.TrimSpace(SafeDeref(account.TenantID))
	userID := strings.TrimSpace(account.ID)
	if tenantID == "" || userID == "" {
		return []string{}, nil
	}

	menuOrgType, ok, resolveErr := resolveMenuOrgType(ctx, tenantID, userID)
	if resolveErr != nil {
		return nil, resolveErr
	}
	if !ok {
		return []string{}, nil
	}

	allowedCodes, exists, allowedErr := GroupApp.OrgTypePermission.GetAllowedUICodes(ctx, tenantID, menuOrgType)
	if allowedErr != nil {
		return nil, allowedErr
	}
	if !exists || len(allowedCodes) == 0 {
		return []string{}, nil
	}

	var ids []string
	if err := global.DB.WithContext(ctx).
		Table("sys_ui_elements").
		Select("id").
		Where("element_code IN ?", allowedCodes).
		Scan(&ids).Error; err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_org_menu_ids",
			"user_id":   account.ID,
			"error":     err.Error(),
		})
	}
	return ids, nil
}

// 获取租户下权限配置表单树
func (*UiElements) GetTenantUiElementsList(ctx context.Context, claims *utils.UserClaims, authority, userKind, orgType string) (map[string]interface{}, error) {
	authority = strings.TrimSpace(authority)
	userKind = strings.TrimSpace(userKind)
	orgType = strings.TrimSpace(orgType)

	if authority == "" {
		authority = "TENANT_ADMIN"
	}
	if userKind == "" {
		userKind = model.UserKindOrgUser
	}

	if authority == "TENANT_ADMIN" {
		list, err := dal.GetTenantUiElementsList()
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_ui_elements",
				"error":     err.Error(),
			})
		}
		return map[string]interface{}{"list": list}, nil
	}

	if authority == "TENANT_USER" {
		resolvedOrgType := orgType
		if userKind == model.UserKindEndUser {
			resolvedOrgType = model.OrgTypeAppUser
		}
		if resolvedOrgType == "" {
			return map[string]interface{}{"list": []*model.UiElementsListRsp1{}}, nil
		}

		codes, exists, err := GroupApp.OrgTypePermission.GetAllowedUICodes(ctx, claims.TenantID, resolvedOrgType)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_org_type_permission_codes",
				"org_type":  resolvedOrgType,
				"error":     err.Error(),
			})
		}
		if !exists || len(codes) == 0 {
			return map[string]interface{}{"list": []*model.UiElementsListRsp1{}}, nil
		}

		list, err := dal.ServeUiElementsFormByCodes(codes)
		if err != nil {
			return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
				"operation": "query_ui_elements_form_by_codes",
				"org_type":  resolvedOrgType,
				"error":     err.Error(),
			})
		}
		return map[string]interface{}{"list": list}, nil
	}

	list, err := dal.GetTenantUiElementsList()
	if err != nil {
		return nil, errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"operation": "query_ui_elements",
			"error":     err.Error(),
		})
	}
	return map[string]interface{}{"list": list}, err
}

func resolveMenuOrgType(ctx context.Context, tenantID, userID string) (string, bool, error) {
	userKind, err := GroupApp.OrgTypePermission.GetUserKind(ctx, tenantID, userID)
	if err != nil {
		return "", false, err
	}

	// 终端用户菜单权限走 APP_USER 配置
	if userKind == model.UserKindEndUser {
		return model.OrgTypeAppUser, true, nil
	}

	if userKind != model.UserKindOrgUser {
		return "", false, nil
	}

	orgType, ok, err := GroupApp.OrgTypePermission.GetUserOrgType(ctx, tenantID, userID)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}

	switch strings.TrimSpace(orgType) {
	case model.OrgTypePACKFactory, model.OrgTypeDealer, model.OrgTypeStore:
		return strings.TrimSpace(orgType), true, nil
	default:
		return "", false, nil
	}
}
