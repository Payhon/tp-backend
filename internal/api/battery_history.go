package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	middleware "project/internal/middleware"
	"project/internal/model"
	"project/internal/service"
	"project/pkg/errcode"
	global "project/pkg/global"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// GetBMSHistoryDeviceList BMS 历史数据设备筛选列表。
func (*BatteryApi) GetBMSHistoryDeviceList(c *gin.Context) {
	var req model.BMSHistoryDeviceListReq
	if !BindAndValidate(c, &req) {
		return
	}

	claims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)
	data, err := service.GroupApp.Battery.ListBMSHistoryDevices(context.Background(), req, claims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBMSHistoryData BMS 历史数据查询（long|wide）。
func (*BatteryApi) GetBMSHistoryData(c *gin.Context) {
	var req model.BMSHistoryQueryReq
	if !BindAndValidate(c, &req) {
		return
	}

	claims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)
	data, err := service.GroupApp.Battery.GetBMSHistory(context.Background(), req, claims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// CreateBMSHistoryExportJob 创建 BMS 历史数据异步导出任务。
func (*BatteryApi) CreateBMSHistoryExportJob(c *gin.Context) {
	var req model.BMSHistoryExportCreateReq
	if !BindAndValidate(c, &req) {
		return
	}

	claims := c.MustGet("claims").(*utils.UserClaims)
	orgID := middleware.GetOrgID(c)
	data, err := service.GroupApp.Battery.CreateBMSHistoryExportJob(context.Background(), req, claims, orgID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetBMSHistoryPendingExportJobs 获取“已完成未下载”的导出任务列表。
func (*BatteryApi) GetBMSHistoryPendingExportJobs(c *gin.Context) {
	var req model.BMSHistoryExportPendingReq
	if !BindAndValidate(c, &req) {
		return
	}

	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.Battery.ListBMSHistoryPendingExportJobs(context.Background(), req, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// DownloadBMSHistoryExport 固定路由下载导出文件，成功后标记已下载。
func (*BatteryApi) DownloadBMSHistoryExport(c *gin.Context) {
	var req model.BMSHistoryExportDownloadReq
	if !BindAndValidate(c, &req) {
		return
	}

	claims := c.MustGet("claims").(*utils.UserClaims)
	filePath, fileName, err := service.GroupApp.Battery.GetBMSHistoryExportDownloadInfo(context.Background(), req.TaskID, claims)
	if err != nil {
		c.Error(err)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		c.Error(errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "导出文件不存在"}))
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		c.Error(errcode.WithData(errcode.CodeNotFound, map[string]interface{}{"message": "导出文件不存在"}))
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Header("Content-Length", strconv.FormatInt(stat.Size(), 10))
	c.Status(http.StatusOK)

	if _, err := io.Copy(c.Writer, file); err != nil {
		logrus.WithError(err).WithField("task_id", req.TaskID).Warn("download bms history export file interrupted")
		return
	}

	if err := service.GroupApp.Battery.MarkBMSHistoryExportDownloaded(context.Background(), req.TaskID, claims); err != nil {
		logrus.WithError(err).WithField("task_id", req.TaskID).Warn("mark bms history export downloaded failed")
	}
}

// ServeBMSHistoryExportWS BMS 历史导出通知 WebSocket。
func (*BatteryApi) ServeBMSHistoryExportWS(c *gin.Context) {
	claims := c.MustGet("claims").(*utils.UserClaims)
	if claims == nil || claims.ID == "" {
		c.Error(errcode.WithData(errcode.CodeUnauthorized, map[string]interface{}{"message": "未授权"}))
		return
	}

	conn, err := Wsupgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Error(errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"message": "WebSocket升级失败"}))
		return
	}
	defer conn.Close()

	if global.BMSHistoryExportWSManager == nil {
		c.Error(errcode.WithData(errcode.CodeSystemError, map[string]interface{}{"message": "通知服务未初始化"}))
		return
	}

	connID := fmt.Sprintf("%s-%d", claims.ID, time.Now().UnixNano())
	var mu sync.Mutex
	client := &global.UserWSClient{
		UserID:  claims.ID,
		Conn:    conn,
		ConnID:  connID,
		MsgType: websocket.TextMessage,
		Mu:      &mu,
	}
	global.BMSHistoryExportWSManager.Subscribe(claims.ID, connID, client)
	defer global.BMSHistoryExportWSManager.Unsubscribe(claims.ID, connID)

	for {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
				continue
			}
			return
		}

		if string(msg) == "ping" {
			mu.Lock()
			_ = conn.WriteMessage(msgType, []byte("pong"))
			mu.Unlock()
		}
	}
}
