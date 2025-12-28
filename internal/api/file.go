package api

import (
	"net/http"
	"strings"

	dal "project/internal/dal"
	model "project/internal/model"
	service "project/internal/service"
	"project/pkg/errcode"
	utils "project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type FileApi struct{}

// GetFileStorageConfig 获取文件存储配置（SYS_ADMIN）
// @Router   /api/v1/file/storage/config [get]
func (*FileApi) GetFileStorageConfig(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	if userClaims.Authority != dal.SYS_ADMIN {
		c.Error(errcode.WithData(errcode.CodeNoPermission, map[string]interface{}{
			"authority": "authority is not sys admin",
		}))
		return
	}

	data, err := service.GroupApp.File.GetFileStorageConfig(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// UpsertFileStorageConfig 创建/修改文件存储配置（SYS_ADMIN）
// @Router   /api/v1/file/storage/config [put]
func (*FileApi) UpsertFileStorageConfig(c *gin.Context) {
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	if userClaims.Authority != dal.SYS_ADMIN {
		c.Error(errcode.WithData(errcode.CodeNoPermission, map[string]interface{}{
			"authority": "authority is not sys admin",
		}))
		return
	}

	var req model.UpsertFileStorageConfigReq
	if !BindAndValidate(c, &req) {
		return
	}
	if err := service.GroupApp.File.UpsertFileStorageConfig(c.Request.Context(), &req); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// GetFileListByPage 获取当前租户文件列表
// @Router   /api/v1/file/list [get]
func (*FileApi) GetFileListByPage(c *gin.Context) {
	var req model.GetFileListByPageReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.File.GetFileListByPage(c.Request.Context(), userClaims, &req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// CreateCloudUploadCredential 获取云存储直传参数（当前租户）
// @Router   /api/v1/file/cloud/credential [post]
func (*FileApi) CreateCloudUploadCredential(c *gin.Context) {
	var req model.CreateCloudUploadCredentialReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.File.CreateCloudUploadCredential(c.Request.Context(), userClaims, &req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// RegisterCloudFile 云直传完成后登记文件记录（当前租户）
// @Router   /api/v1/file/cloud/register [post]
func (*FileApi) RegisterCloudFile(c *gin.Context) {
	var req model.RegisterCloudFileReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.File.RegisterCloudFile(c.Request.Context(), userClaims, &req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// ServeFilesCloudRedirect /files-cloud/:id 302重定向到云存储URL（公开）
func (*FileApi) ServeFilesCloudRedirect(c *gin.Context) {
	id := c.Param("id")
	id = strings.TrimSpace(id)
	if id == "" {
		c.Status(http.StatusNotFound)
		return
	}
	f, err := dal.GetFileByID(c.Request.Context(), id)
	if err != nil || f == nil || f.FullURL == "" {
		c.Status(http.StatusNotFound)
		return
	}
	c.Redirect(http.StatusFound, f.FullURL)
}

