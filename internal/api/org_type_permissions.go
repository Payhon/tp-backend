package api

import (
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type OrgTypePermissionApi struct{}

// ListOrgTypePermissions 获取机构类型权限配置
// @Summary 获取机构类型权限配置
// @Tags 权限管理
// @Accept json
// @Produce json
// @Param tenant_id query string false "租户ID（仅SYS_ADMIN可用）"
// @Success 200 {object} []model.OrgTypePermissionResp
// @Router /api/v1/org_type_permissions [get]
func (*OrgTypePermissionApi) ListOrgTypePermissions(c *gin.Context) {
	claims := c.MustGet("claims").(*utils.UserClaims)
	tenantID := c.Query("tenant_id")

	data, err := service.GroupApp.OrgTypePermission.List(c.Request.Context(), claims, tenantID)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// UpsertOrgTypePermission 更新机构类型权限配置
// @Summary 更新机构类型权限配置
// @Tags 权限管理
// @Accept json
// @Produce json
// @Param org_type path string true "机构类型: PACK_FACTORY, DEALER, STORE"
// @Param tenant_id query string false "租户ID（仅SYS_ADMIN可用）"
// @Param body body model.OrgTypePermissionUpsertReq true "权限配置"
// @Success 200 {object} model.OrgTypePermissionResp
// @Router /api/v1/org_type_permissions/{org_type} [put]
func (*OrgTypePermissionApi) UpsertOrgTypePermission(c *gin.Context) {
	claims := c.MustGet("claims").(*utils.UserClaims)
	orgType := c.Param("org_type")
	tenantID := c.Query("tenant_id")

	var req model.OrgTypePermissionUpsertReq
	if !BindAndValidate(c, &req) {
		return
	}

	data, err := service.GroupApp.OrgTypePermission.Upsert(c.Request.Context(), claims, tenantID, orgType, &req)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// GetDeviceParamPermissionOptions 获取设备参数权限可选项
// @Summary 获取设备参数权限可选项
// @Tags 权限管理
// @Accept json
// @Produce json
// @Success 200 {object} []model.DeviceParamTreeNode
// @Router /api/v1/org_type_permissions/device_param_options [get]
func (*OrgTypePermissionApi) GetDeviceParamPermissionOptions(c *gin.Context) {
	opts, err := service.GroupApp.OrgTypePermission.GetDeviceParamOptions()
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", opts)
}
