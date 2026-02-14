package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project/internal/dal"
	"project/pkg/utils"
)

// OpenAPIAppSecretAuth 第三方 OpenAPI 认证（x-app-id + x-secret-key）
func OpenAPIAppSecretAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("X-Request-ID")
		appID := strings.TrimSpace(c.GetHeader("x-app-id"))
		secretKey := strings.TrimSpace(c.GetHeader("x-secret-key"))

		if appID == "" || secretKey == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Code:      ErrCodeNoAuth,
				Message:   "missing authentication (x-app-id and x-secret-key required)",
				RequestID: requestID,
			})
			c.Abort()
			return
		}

		tenantID, createdID, keyID, err := dal.VerifyOpenAPICredentials(context.Background(), appID, secretKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Code:      ErrCodeInvalidAPIKey,
				Message:   "invalid app_id or secret_key",
				RequestID: requestID,
			})
			c.Abort()
			return
		}

		if strings.TrimSpace(createdID) == "" {
			createdID = "openapi"
		}

		claims := utils.UserClaims{
			TenantID:  tenantID,
			Authority: "TENANT_ADMIN",
			ID:        createdID,
		}
		c.Set("claims", &claims)
		c.Set("open_api_key_id", keyID)
		c.Next()
	}
}
