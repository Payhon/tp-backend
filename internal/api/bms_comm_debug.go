package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/errcode"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type BmsCommDebugApi struct{}

// @Router /api/v1/bms/comm-debug/logs [get]
func (*BmsCommDebugApi) GetLogs(c *gin.Context) {
	var req model.BmsCommDebugLogListReq
	if !BindAndValidate(c, &req) {
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	orgIDVal, _ := c.Get(middleware.DealerIDContextKey)
	orgID, _ := orgIDVal.(string)

	data, err := service.GroupApp.BmsCommDebug.GetLogList(c, req, userClaims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// @Router /api/v1/bms/comm-debug/logs/stream [get]
func (*BmsCommDebugApi) StreamLogs(c *gin.Context) {
	userClaims, ok := c.MustGet("claims").(*utils.UserClaims)
	if !ok {
		c.Error(errcode.WithData(errcode.CodeParamError, map[string]interface{}{"error": "UserClaims not found"}))
		return
	}

	afterID := int64(0)
	if v := c.Query("after_id"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			afterID = parsed
		}
	}
	limit := 200
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	deviceID := optionalQueryString(c, "device_id")
	eventType := optionalQueryString(c, "event_type")

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	fmt.Fprintf(c.Writer, "event: ready\ndata: {\"after_id\":%d}\n\n", afterID)
	c.Writer.Flush()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			items, err := service.GroupApp.BmsCommDebug.GetLogsAfterID(context.Background(), userClaims.TenantID, afterID, limit, deviceID, eventType)
			if err != nil {
				fmt.Fprintf(c.Writer, "event: error\ndata: %q\n\n", err.Error())
				c.Writer.Flush()
				continue
			}

			if len(items) > 0 {
				lastID := afterID
				for _, item := range items {
					if item.ID > lastID {
						lastID = item.ID
					}
				}
				payload, _ := toJSON(items)
				fmt.Fprintf(c.Writer, "event: comm-debug\ndata: %s\n\n", payload)
				c.Writer.Flush()
				afterID = lastID
				continue
			}

			fmt.Fprintf(c.Writer, "event: heartbeat\ndata: %d\n\n", time.Now().Unix())
			c.Writer.Flush()
		}
	}
}

func optionalQueryString(c *gin.Context, key string) *string {
	value := c.Query(key)
	if value == "" {
		return nil
	}
	return &value
}

func toJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}
