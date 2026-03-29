package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

// OrgTypePermission 机构类型权限配置
type OrgTypePermission struct{}

func (*OrgTypePermission) InitOrgTypePermission(Router *gin.RouterGroup) {
	g := Router.Group("org_type_permissions")
	{
		g.GET("", api.Controllers.OrgTypePermissionApi.ListOrgTypePermissions)
		g.PUT(":org_type", api.Controllers.OrgTypePermissionApi.UpsertOrgTypePermission)
		g.GET("ui_codes/me", api.Controllers.OrgTypePermissionApi.GetCurrentUIPermissions)
		g.GET("mobile_ui_codes/me", api.Controllers.OrgTypePermissionApi.GetCurrentMobileUIPermissions)
		g.GET("device_param_permissions/me", api.Controllers.OrgTypePermissionApi.GetCurrentDeviceParamPermissions)
		g.GET("device_param_options", api.Controllers.OrgTypePermissionApi.GetDeviceParamPermissionOptions)
	}
}
