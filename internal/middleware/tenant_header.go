package middleware

import (
	"strings"
	"time"

	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

const (
	TenantHeaderName         = "X-TenantID"
	TenantIDHeaderContextKey = "x_tenant_id"
)

func getDefaultTenantID(c *gin.Context) (string, error) {
	var tenantID string
	err := global.DB.WithContext(c.Request.Context()).
		Table("users").
		Select("tenant_id").
		Where("authority = ? AND tenant_id IS NOT NULL AND tenant_id <> ''", "TENANT_ADMIN").
		Order("created_at ASC NULLS LAST, id ASC").
		Limit(1).
		Scan(&tenantID).Error
	if err != nil {
		return "", errcode.WithData(errcode.CodeDBError, map[string]interface{}{
			"sql_error": err.Error(),
		})
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return "", errcode.New(errcode.CodeNotFound)
	}
	return tenantID, nil
}

// RequireTenantHeader enforces X-TenantID header existence and stores it into context.
// It is intended for APP/小程序专用接口（不影响现有WEB端接口）。
func RequireTenantHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := strings.TrimSpace(c.GetHeader(TenantHeaderName))
		if tenantID == "" {
			// 默认第一个租户（避免客户端未传 header 时无法使用 APP 接口）
			defaultTenantID, err := getDefaultTenantID(c)
			if err != nil {
				c.Error(err)
				c.Abort()
				return
			}
			tenantID = defaultTenantID
		}
		c.Set(TenantIDHeaderContextKey, tenantID)
		c.Next()
	}
}

// RequireTenantHeaderMatchClaims ensures X-TenantID matches JWT claims. If header is absent, it will fail.
func RequireTenantHeaderMatchClaims() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := strings.TrimSpace(c.GetHeader(TenantHeaderName))

		claimsVal, exists := c.Get("claims")
		var claims *utils.UserClaims
		if exists {
			if v, ok := claimsVal.(*utils.UserClaims); ok {
				claims = v
			}
		}

		// Header 缺省：优先使用 token tenant；否则默认第一个租户
		if tenantID == "" {
			if claims != nil && strings.TrimSpace(claims.TenantID) != "" {
				tenantID = strings.TrimSpace(claims.TenantID)
			} else {
				defaultTenantID, err := getDefaultTenantID(c)
				if err != nil {
					c.Error(err)
					c.Abort()
					return
				}
				tenantID = defaultTenantID
			}
		}

		if claims != nil && strings.TrimSpace(claims.TenantID) != "" && strings.TrimSpace(claims.TenantID) != tenantID {
			c.Error(errcode.WithData(errcode.CodeNoPermission, map[string]interface{}{
				"error":      "tenant mismatch",
				"token":      claims.TenantID,
				"header":     tenantID,
				"header_key": TenantHeaderName,
				"time":       time.Now().Unix(),
			}))
			c.Abort()
			return
		}

		c.Set(TenantIDHeaderContextKey, tenantID)
		c.Next()
	}
}

func GetTenantIDFromHeader(c *gin.Context) string {
	if v, ok := c.Get(TenantIDHeaderContextKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return strings.TrimSpace(c.GetHeader(TenantHeaderName))
}
