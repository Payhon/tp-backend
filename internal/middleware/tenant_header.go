package middleware

import (
	"strings"

	"project/pkg/errcode"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

const (
	TenantHeaderName         = "X-TenantID"
	TenantIDHeaderContextKey = "x_tenant_id"
)

// RequireTenantHeader enforces X-TenantID header existence and stores it into context.
// It is intended for APP/小程序专用接口（不影响现有WEB端接口）。
func RequireTenantHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := strings.TrimSpace(c.GetHeader(TenantHeaderName))
		if tenantID == "" {
			c.Error(errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"header": TenantHeaderName,
				"error":  "missing tenant header",
			}))
			c.Abort()
			return
		}
		c.Set(TenantIDHeaderContextKey, tenantID)
		c.Next()
	}
}

// RequireTenantHeaderMatchClaims ensures X-TenantID matches JWT claims. If header is absent, it will fail.
func RequireTenantHeaderMatchClaims() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := strings.TrimSpace(c.GetHeader(TenantHeaderName))
		if tenantID == "" {
			c.Error(errcode.WithData(errcode.CodeParamError, map[string]interface{}{
				"header": TenantHeaderName,
				"error":  "missing tenant header",
			}))
			c.Abort()
			return
		}
		claimsVal, exists := c.Get("claims")
		if exists {
			if claims, ok := claimsVal.(*utils.UserClaims); ok && claims != nil && claims.TenantID != "" && claims.TenantID != tenantID {
				c.Error(errcode.WithData(errcode.CodeNoPermission, map[string]interface{}{
					"error":      "tenant mismatch",
					"token":      claims.TenantID,
					"header":     tenantID,
					"header_key": TenantHeaderName,
				}))
				c.Abort()
				return
			}
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
