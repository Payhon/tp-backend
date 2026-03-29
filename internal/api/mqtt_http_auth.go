package api

import (
	"net/http"
	"strings"

	"project/internal/model"
	"project/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// MqttHTTPAuthApi EMQX HTTP 认证接口
type MqttHTTPAuthApi struct{}

// Auth EMQX HTTP Auth 回调
// @Summary EMQX HTTP 认证
// @Description 供 EMQX HTTP Auth 调用，校验设备 MQTT 用户名密码
// @Tags MQTT
// @Accept json
// @Produce json
// @Param body body model.MqttHttpAuthReq true "认证请求"
// @Success 200 {object} model.MqttHttpAuthResp
// @Router /api/v1/mqtt/auth [post]
func (*MqttHTTPAuthApi) Auth(c *gin.Context) {
	var req model.MqttHttpAuthReq
	_ = c.ShouldBind(&req) // 兼容 JSON / form

	resp := model.MqttHttpAuthResp{
		Result:      "deny",
		IsSuperuser: false,
	}

	if !checkMqttHTTPAuthSecret(c) {
		resp.Reason = "unauthorized"
		c.JSON(http.StatusOK, resp)
		return
	}

	result, reason := service.GroupApp.MqttHTTPAuth.Auth(&req)
	resp.Result = result
	if reason != "" {
		resp.Reason = reason
	}

	c.JSON(http.StatusOK, resp)
}

func checkMqttHTTPAuthSecret(c *gin.Context) bool {
	secret := strings.TrimSpace(viper.GetString("mqtt.http_auth.shared_secret"))
	if secret == "" {
		return true
	}

	// 允许两种方式携带密钥：
	// 1) X-EMQX-API-KEY / X-API-KEY
	// 2) Authorization: Bearer <secret>
	if key := strings.TrimSpace(c.GetHeader("X-EMQX-API-KEY")); key != "" {
		return key == secret
	}
	if key := strings.TrimSpace(c.GetHeader("X-API-KEY")); key != "" {
		return key == secret
	}
	if auth := strings.TrimSpace(c.GetHeader("Authorization")); auth != "" {
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			return strings.TrimSpace(auth[7:]) == secret
		}
	}
	return false
}
