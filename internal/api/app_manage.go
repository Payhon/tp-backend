package api

import (
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AppManageApi APP管理：应用管理/升级中心
type AppManageApi struct{}

// ListApps 应用列表
// @Summary 应用列表
// @Tags APP-Manage
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param keyword query string false "搜索关键词(AppID/名称)"
// @Success 200 {object} model.AppListResp
// @Router /api/v1/apps [get]
func (*AppManageApi) ListApps(c *gin.Context) {
	var req model.AppListReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppManage.ListApps(c.Request.Context(), req, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetApp 应用详情
// @Summary 应用详情
// @Tags APP-Manage
// @Produce json
// @Param id path string true "应用ID"
// @Success 200 {object} model.AppDetailResp
// @Router /api/v1/apps/{id} [get]
func (*AppManageApi) GetApp(c *gin.Context) {
	id := c.Param("id")
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppManage.GetApp(c.Request.Context(), id, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// CreateApp 创建应用
// @Summary 创建应用
// @Tags APP-Manage
// @Accept json
// @Produce json
// @Param body body model.AppCreateReq true "创建应用"
// @Success 200 {object} model.Response
// @Router /api/v1/apps [post]
func (*AppManageApi) CreateApp(c *gin.Context) {
	var req model.AppCreateReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	id, err := service.GroupApp.AppManage.CreateApp(c.Request.Context(), req, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", map[string]interface{}{"id": id})
}

// UpdateApp 更新应用
// @Summary 更新应用
// @Tags APP-Manage
// @Accept json
// @Produce json
// @Param id path string true "应用ID"
// @Param body body model.AppUpdateReq true "更新应用"
// @Success 200 {object} model.Response
// @Router /api/v1/apps/{id} [put]
func (*AppManageApi) UpdateApp(c *gin.Context) {
	id := c.Param("id")
	var req model.AppUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppManage.UpdateApp(c.Request.Context(), id, req, claims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// DeleteApp 删除应用
// @Summary 删除应用
// @Tags APP-Manage
// @Produce json
// @Param id path string true "应用ID"
// @Success 200 {object} model.Response
// @Router /api/v1/apps/{id} [delete]
func (*AppManageApi) DeleteApp(c *gin.Context) {
	id := c.Param("id")
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppManage.DeleteApp(c.Request.Context(), id, claims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// BatchDeleteApps 批量删除应用
// @Summary 批量删除应用
// @Tags APP-Manage
// @Accept json
// @Produce json
// @Param body body model.BatchDeleteReq true "批量删除"
// @Success 200 {object} model.Response
// @Router /api/v1/apps/batch_delete [post]
func (*AppManageApi) BatchDeleteApps(c *gin.Context) {
	var req model.BatchDeleteReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppManage.BatchDeleteApps(c.Request.Context(), req.IDs, claims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// ListAppVersions 版本列表
// @Summary 版本列表
// @Tags APP-Upgrade
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param app_id query string false "应用ID(apps.id)"
// @Param keyword query string false "关键词(title/version)"
// @Param type query string false "安装包类型(native_app/wgt)"
// @Success 200 {object} model.AppVersionListResp
// @Router /api/v1/app_versions [get]
func (*AppManageApi) ListAppVersions(c *gin.Context) {
	var req model.AppVersionListReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppManage.ListAppVersions(c.Request.Context(), req, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetAppVersion 版本详情
// @Summary 版本详情
// @Tags APP-Upgrade
// @Produce json
// @Param id path string true "版本ID"
// @Success 200 {object} model.AppVersionDetailResp
// @Router /api/v1/app_versions/{id} [get]
func (*AppManageApi) GetAppVersion(c *gin.Context) {
	id := c.Param("id")
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppManage.GetAppVersion(c.Request.Context(), id, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// CreateAppVersion 发布新版
// @Summary 发布新版
// @Tags APP-Upgrade
// @Accept json
// @Produce json
// @Param body body model.AppVersionCreateReq true "发布新版"
// @Success 200 {object} model.Response
// @Router /api/v1/app_versions [post]
func (*AppManageApi) CreateAppVersion(c *gin.Context) {
	var req model.AppVersionCreateReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	id, err := service.GroupApp.AppManage.CreateAppVersion(c.Request.Context(), req, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", map[string]interface{}{"id": id})
}

// UpdateAppVersion 更新版本
// @Summary 更新版本
// @Tags APP-Upgrade
// @Accept json
// @Produce json
// @Param id path string true "版本ID"
// @Param body body model.AppVersionUpdateReq true "更新版本"
// @Success 200 {object} model.Response
// @Router /api/v1/app_versions/{id} [put]
func (*AppManageApi) UpdateAppVersion(c *gin.Context) {
	id := c.Param("id")
	var req model.AppVersionUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppManage.UpdateAppVersion(c.Request.Context(), id, req, claims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// DeleteAppVersion 删除版本
// @Summary 删除版本
// @Tags APP-Upgrade
// @Produce json
// @Param id path string true "版本ID"
// @Success 200 {object} model.Response
// @Router /api/v1/app_versions/{id} [delete]
func (*AppManageApi) DeleteAppVersion(c *gin.Context) {
	id := c.Param("id")
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppManage.DeleteAppVersion(c.Request.Context(), id, claims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// BatchDeleteAppVersions 批量删除版本
// @Summary 批量删除版本
// @Tags APP-Upgrade
// @Accept json
// @Produce json
// @Param body body model.BatchDeleteReq true "批量删除"
// @Success 200 {object} model.Response
// @Router /api/v1/app_versions/batch_delete [post]
func (*AppManageApi) BatchDeleteAppVersions(c *gin.Context) {
	var req model.BatchDeleteReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.AppManage.BatchDeleteAppVersions(c.Request.Context(), req.IDs, claims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

