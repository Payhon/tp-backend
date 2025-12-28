package api

import (
	"fmt"
	"project/pkg/errcode"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
	service "project/internal/service"
)

type UpLoadApi struct{}

const (
	MaxFileSize   = 500 << 20 // 200MB
)

// UpFile 处理文件上传
// @Tags     文件上传
// @Router   /api/v1/file/up [post]
func (*UpLoadApi) UpFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		c.Error(errcode.New(errcode.CodeFileEmpty))
		return
	}

	fileType := c.PostForm("type")
	if fileType == "" {
		c.Error(errcode.New(errcode.CodeFileEmpty))
		return
	}

	if file.Size > MaxFileSize {
		c.Error(errcode.WithVars(errcode.CodeFileTooLarge, map[string]interface{}{
			"max_size":     "500MB",
			"current_size": fmt.Sprintf("%.2fMB", float64(file.Size)/(1<<20)),
		}))
		return
	}

	userClaims := c.MustGet("claims").(*utils.UserClaims)
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := c.Request.Host

	data, err := service.GroupApp.File.UploadFile(c.Request.Context(), userClaims, scheme, host, file, fileType)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
