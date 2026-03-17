package service

import (
	"context"
	"strings"
	"time"

	global "project/pkg/global"

	"github.com/go-basic/uuid"
	"github.com/sirupsen/logrus"
)

type Casbin struct{}

type userRoleRow struct {
	ID        string     `gorm:"column:id"`
	UserID    string     `gorm:"column:user_id"`
	RoleID    string     `gorm:"column:role_id"`
	TenantID  string     `gorm:"column:tenant_id"`
	CreatedAt *time.Time `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

type rolePermissionRow struct {
	ID            string     `gorm:"column:id"`
	RoleID        string     `gorm:"column:role_id"`
	TenantID      string     `gorm:"column:tenant_id"`
	PermissionKey string     `gorm:"column:permission_key"`
	CreatedAt     *time.Time `gorm:"column:created_at"`
	UpdatedAt     *time.Time `gorm:"column:updated_at"`
}

func syncRolePoliciesToCasbin(ctx context.Context, role string) bool {
	if _, err := global.CasbinEnforcer.RemoveFilteredPolicy(0, role); err != nil {
		logrus.WithError(err).WithField("role", role).Error("remove role policies failed")
		return false
	}

	var permissionIDs []string
	if err := global.DB.WithContext(ctx).
		Table("role_permissions").
		Select("permission_key").
		Where("role_id = ?", role).
		Order("created_at ASC NULLS LAST, permission_key ASC").
		Scan(&permissionIDs).Error; err != nil {
		logrus.WithError(err).WithField("role", role).Error("query role permissions failed")
		return false
	}

	if len(permissionIDs) == 0 {
		return true
	}

	rules := make([][]string, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		permissionID = strings.TrimSpace(permissionID)
		if permissionID == "" {
			continue
		}
		rules = append(rules, []string{role, permissionID, "allow"})
	}

	if len(rules) == 0 {
		return true
	}

	ok, err := global.CasbinEnforcer.AddNamedPolicies("p", rules)
	if err != nil {
		logrus.WithError(err).WithField("role", role).Error("add role policies failed")
		return false
	}
	return ok || len(rules) > 0
}

func syncUserRolesToCasbin(ctx context.Context, user string) bool {
	if _, err := global.CasbinEnforcer.RemoveFilteredNamedGroupingPolicy("g", 0, user); err != nil {
		logrus.WithError(err).WithField("user", user).Error("remove user roles failed")
		return false
	}

	var roleIDs []string
	if err := global.DB.WithContext(ctx).
		Table("user_roles").
		Select("role_id").
		Where("user_id = ?", user).
		Order("created_at ASC NULLS LAST, role_id ASC").
		Scan(&roleIDs).Error; err != nil {
		logrus.WithError(err).WithField("user", user).Error("query user roles failed")
		return false
	}

	if len(roleIDs) == 0 {
		return true
	}

	rules := make([][]string, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		roleID = strings.TrimSpace(roleID)
		if roleID == "" {
			continue
		}
		rules = append(rules, []string{user, roleID})
	}

	if len(rules) == 0 {
		return true
	}

	ok, err := global.CasbinEnforcer.AddNamedGroupingPolicies("g", rules)
	if err != nil {
		logrus.WithError(err).WithField("user", user).Error("add user roles failed")
		return false
	}
	return ok || len(rules) > 0
}

// 角色添加多个权限
func (*Casbin) AddFunctionToRole(role string, functions []string) bool {
	role = strings.TrimSpace(role)
	if role == "" {
		return false
	}

	now := time.Now().UTC()
	var roleTenantID string
	if err := global.DB.WithContext(context.Background()).
		Table("roles").
		Select("tenant_id").
		Where("id = ?", role).
		Limit(1).
		Scan(&roleTenantID).Error; err != nil {
		logrus.WithError(err).WithField("role", role).Error("query role tenant failed")
		return false
	}

	if len(functions) == 0 {
		return syncRolePoliciesToCasbin(context.Background(), role)
	}

	rows := make([]rolePermissionRow, 0, len(functions))
	for _, functionID := range functions {
		functionID = strings.TrimSpace(functionID)
		if functionID == "" {
			continue
		}
		rows = append(rows, rolePermissionRow{
			ID:            uuid.New(),
			RoleID:        role,
			TenantID:      roleTenantID,
			PermissionKey: functionID,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		})
	}

	if len(rows) == 0 {
		return syncRolePoliciesToCasbin(context.Background(), role)
	}

	if err := global.DB.WithContext(context.Background()).
		Table("role_permissions").
		Create(&rows).Error; err != nil {
		logrus.WithError(err).WithField("role", role).Error("create role permissions failed")
		return false
	}

	return syncRolePoliciesToCasbin(context.Background(), role)
}

// 查询角色的功能
func (*Casbin) GetFunctionFromRole(role string) ([]string, bool) {
	var functions []string
	err := global.DB.WithContext(context.Background()).
		Table("role_permissions").
		Select("permission_key").
		Where("role_id = ?", role).
		Order("created_at ASC NULLS LAST, permission_key ASC").
		Scan(&functions).Error
	if err != nil {
		logrus.WithError(err).WithField("role", role).Error("get role permissions failed")
		return nil, false
	}
	return functions, true
}

// 删除角色和功能
func (*Casbin) RemoveRoleAndFunction(role string) bool {
	if err := global.DB.WithContext(context.Background()).
		Table("role_permissions").
		Where("role_id = ?", role).
		Delete(nil).Error; err != nil {
		logrus.WithError(err).WithField("role", role).Error("delete role permissions failed")
		return false
	}

	if _, err := global.CasbinEnforcer.RemoveFilteredPolicy(0, role); err != nil {
		logrus.WithError(err).WithField("role", role).Error("remove role policies failed")
		return false
	}
	return true
}

// 用户添加多个角色
func (*Casbin) AddRolesToUser(user string, roles []string) bool {
	user = strings.TrimSpace(user)
	if user == "" {
		return false
	}

	now := time.Now().UTC()
	var tenantID string
	if err := global.DB.WithContext(context.Background()).
		Table("users").
		Select("tenant_id").
		Where("id = ?", user).
		Limit(1).
		Scan(&tenantID).Error; err != nil {
		logrus.WithError(err).WithField("user", user).Error("query user tenant failed")
		return false
	}

	rows := make([]userRoleRow, 0, len(roles))
	for _, roleID := range roles {
		roleID = strings.TrimSpace(roleID)
		if roleID == "" {
			continue
		}
		rows = append(rows, userRoleRow{
			ID:        uuid.New(),
			UserID:    user,
			RoleID:    roleID,
			TenantID:  tenantID,
			CreatedAt: &now,
			UpdatedAt: &now,
		})
	}

	if len(rows) > 0 {
		if err := global.DB.WithContext(context.Background()).
			Table("user_roles").
			Create(&rows).Error; err != nil {
			logrus.WithError(err).WithField("user", user).Error("create user roles failed")
			return false
		}
	}

	return syncUserRolesToCasbin(context.Background(), user)
}

// 查询用户的角色
func (*Casbin) GetRoleFromUser(user string) ([]string, bool) {
	var roles []string
	err := global.DB.WithContext(context.Background()).
		Table("user_roles").
		Select("role_id").
		Where("user_id = ?", user).
		Order("created_at ASC NULLS LAST, role_id ASC").
		Scan(&roles).Error
	if err != nil {
		logrus.WithError(err).WithField("user", user).Error("get user roles failed")
		return nil, false
	}
	return roles, true
}

// 删除用户和角色
func (*Casbin) RemoveUserAndRole(user string) bool {
	if err := global.DB.WithContext(context.Background()).
		Table("user_roles").
		Where("user_id = ?", user).
		Delete(nil).Error; err != nil {
		logrus.WithError(err).WithField("user", user).Error("delete user roles failed")
		return false
	}

	if _, err := global.CasbinEnforcer.RemoveFilteredNamedGroupingPolicy("g", 0, user); err != nil {
		logrus.WithError(err).WithField("user", user).Error("remove user grouping failed")
		return false
	}
	return true
}

// 查询是否存在某个资源
func (*Casbin) GetUrl(url string) bool {
	stringList := global.CasbinEnforcer.GetFilteredNamedGroupingPolicy("g2", 0, url)
	return len(stringList) != 0
}

// 查询用户角色中是否存在某个角色
func (*Casbin) HasRole(role string) bool {
	var count int64
	err := global.DB.WithContext(context.Background()).
		Table("user_roles").
		Where("role_id = ?", role).
		Count(&count).Error
	if err != nil {
		logrus.WithError(err).WithField("role", role).Error("check role bindings failed")
		return false
	}
	return count > 0
}

// 校验
func (*Casbin) Verify(user string, url string) bool {
	isTrue, _ := global.CasbinEnforcer.Enforce(user, url, "allow")
	return isTrue
}
