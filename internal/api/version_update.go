package api

import (
	"project/internal/model"
	"project/internal/service"

	"github.com/gin-gonic/gin"
)

type VersionUpdateApi struct{}

// CreateVersionUpdate 新增版本更新记录
// @Router /api/v1/version-updates [post]
func (*VersionUpdateApi) CreateVersionUpdate(c *gin.Context) {
	var req model.VersionUpdateCreateReq
	if !BindAndValidate(c, &req) {
		return
	}
	data, err := service.GroupApp.VersionUpdate.Create(c, req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetVersionUpdate 获取版本更新记录详情
// @Router /api/v1/version-updates/{id} [get]
func (*VersionUpdateApi) GetVersionUpdate(c *gin.Context) {
	data, err := service.GroupApp.VersionUpdate.Get(c, c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// UpdateVersionUpdate 更新版本更新记录
// @Router /api/v1/version-updates/{id} [put]
func (*VersionUpdateApi) UpdateVersionUpdate(c *gin.Context) {
	var req model.VersionUpdateUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}
	data, err := service.GroupApp.VersionUpdate.Update(c, c.Param("id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// DeleteVersionUpdate 删除版本更新记录
// @Router /api/v1/version-updates/{id} [delete]
func (*VersionUpdateApi) DeleteVersionUpdate(c *gin.Context) {
	if err := service.GroupApp.VersionUpdate.Delete(c, c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// ListVersionUpdates 版本更新记录列表
// @Router /api/v1/version-updates [get]
func (*VersionUpdateApi) ListVersionUpdates(c *gin.Context) {
	var req model.VersionUpdateListReq
	if !BindAndValidate(c, &req) {
		return
	}
	data, err := service.GroupApp.VersionUpdate.List(c, req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
