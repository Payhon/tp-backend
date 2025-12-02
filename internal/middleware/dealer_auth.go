package middleware

import (
	"project/pkg/global"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DealerIDContextKey 在 Gin 上下文中存储经销商ID的键
const DealerIDContextKey = "dealer_id"

// DealerAuthMiddleware
// 为当前请求加载用户归属经销商ID（users.dealer_id），并注入到 Gin Context 中。
// - 若用户未绑定经销商，则 dealer_id 为空字符串；
// - 若查询失败，仅记录日志，不拦截请求，避免影响非经销商账号。
func DealerAuthMiddleware() gin.HandlerFunc {
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

		var dealerID string
		if err := global.DB.Table("users").
			Select("dealer_id").
			Where("id = ?", claims.ID).
			Scan(&dealerID).Error; err != nil {
			logrus.WithError(err).WithField("user_id", claims.ID).Error("failed to load dealer_id for user")
			// 不中断请求流程
			c.Next()
			return
		}

		c.Set(DealerIDContextKey, dealerID)
		c.Next()
	}
}

