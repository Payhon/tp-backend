package middleware

import (
	"project/internal/model"
	"project/pkg/global"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// 组织权限相关的 Context Key
const (
	OrgIDContextKey    = "org_id"    // 当前用户归属的组织ID
	UserKindContextKey = "user_kind" // 用户类型: ORG_USER | END_USER
	TenantIDContextKey = "tenant_id" // 租户ID
)

// OrgUserInfo 组织用户信息（从 users 表加载）
type OrgUserInfo struct {
	OrgID    string // 归属组织ID
	UserKind string // 用户类型
	TenantID string // 租户ID
}

// OrgAuthMiddleware
// 为当前请求加载用户的组织信息（org_id, user_kind），并注入到 Gin Context 中。
// - 若用户为 ORG_USER（业务账号），则 org_id 表示其归属的组织；
// - 若用户为 END_USER（终端用户），则 org_id 可能为空，权限通过 device_user_bindings 控制；
// - 若查询失败，仅记录日志，不拦截请求。
func OrgAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("claims")
		if !exists {
			c.Next()
			return
		}

		claims, ok := claimsVal.(*utils.UserClaims)
		if !ok || claims == nil || claims.ID == "" {
			c.Next()
			return
		}

		var userInfo OrgUserInfo
		if err := global.DB.Table("users").
			Select("org_id, user_kind, tenant_id").
			Where("id = ?", claims.ID).
			Scan(&userInfo).Error; err != nil {
			logrus.WithError(err).WithField("user_id", claims.ID).Error("failed to load org info for user")
			c.Next()
			return
		}

		// 设置默认 user_kind
		if userInfo.UserKind == "" {
			userInfo.UserKind = model.UserKindEndUser
		}

		c.Set(OrgIDContextKey, userInfo.OrgID)
		c.Set(UserKindContextKey, userInfo.UserKind)
		c.Set(TenantIDContextKey, userInfo.TenantID)

		// 同时保留 dealer_id 兼容（逐步废弃）
		c.Set(DealerIDContextKey, userInfo.OrgID)

		c.Next()
	}
}

// GetOrgID 从 Gin Context 获取当前用户的组织ID
func GetOrgID(c *gin.Context) string {
	if val, exists := c.Get(OrgIDContextKey); exists {
		if orgID, ok := val.(string); ok {
			return orgID
		}
	}
	return ""
}

// GetUserKind 从 Gin Context 获取当前用户类型
func GetUserKind(c *gin.Context) string {
	if val, exists := c.Get(UserKindContextKey); exists {
		if kind, ok := val.(string); ok {
			return kind
		}
	}
	return model.UserKindEndUser
}

// IsOrgUser 判断当前用户是否为组织用户（业务账号）
func IsOrgUser(c *gin.Context) bool {
	return GetUserKind(c) == model.UserKindOrgUser
}

// IsEndUser 判断当前用户是否为终端用户
func IsEndUser(c *gin.Context) bool {
	return GetUserKind(c) == model.UserKindEndUser
}

// OrgScopeSubQuery 返回子树过滤的 SQL 子查询（用于 WHERE owner_org_id IN (...)）
// 参数 tenantID 和 orgID 为当前用户的租户和组织
// 返回格式示例: SELECT descendant_id FROM org_closure WHERE tenant_id = ? AND ancestor_id = ?
func OrgScopeSubQuery(tenantID, orgID string) string {
	return `SELECT descendant_id FROM org_closure WHERE tenant_id = '` + tenantID + `' AND ancestor_id = '` + orgID + `'`
}

// GetOrgDescendants 获取指定组织的所有后代组织ID（包含自身）
// 返回的列表可用于 IN 查询
func GetOrgDescendants(tenantID, orgID string) ([]string, error) {
	var descendants []string
	err := global.DB.Table("org_closure").
		Select("descendant_id").
		Where("tenant_id = ? AND ancestor_id = ?", tenantID, orgID).
		Pluck("descendant_id", &descendants).Error
	if err != nil {
		return nil, err
	}
	// 如果结果为空，至少包含自身
	if len(descendants) == 0 {
		descendants = []string{orgID}
	}
	return descendants, nil
}

// GetOrgAncestors 获取指定组织的所有祖先组织ID（包含自身）
func GetOrgAncestors(tenantID, orgID string) ([]string, error) {
	var ancestors []string
	err := global.DB.Table("org_closure").
		Select("ancestor_id").
		Where("tenant_id = ? AND descendant_id = ?", tenantID, orgID).
		Pluck("ancestor_id", &ancestors).Error
	if err != nil {
		return nil, err
	}
	if len(ancestors) == 0 {
		ancestors = []string{orgID}
	}
	return ancestors, nil
}

// CanAccessOrg 检查 accessorOrgID 是否有权访问 targetOrgID（target 在 accessor 的子树内）
func CanAccessOrg(tenantID, accessorOrgID, targetOrgID string) bool {
	if accessorOrgID == "" {
		// 无组织归属的用户（如系统管理员）默认可访问所有
		return true
	}
	if accessorOrgID == targetOrgID {
		return true
	}

	var count int64
	global.DB.Table("org_closure").
		Where("tenant_id = ? AND ancestor_id = ? AND descendant_id = ?", tenantID, accessorOrgID, targetOrgID).
		Count(&count)
	return count > 0
}
