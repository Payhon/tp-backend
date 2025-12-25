package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type Org struct{}

func (*Org) InitOrg(Router *gin.RouterGroup) {
	// 组织管理路由
	orgApi := Router.Group("org")
	{
		// 增
		orgApi.POST("", api.Controllers.OrgApi.CreateOrg)

		// 删
		orgApi.DELETE(":id", api.Controllers.OrgApi.DeleteOrg)

		// 改
		orgApi.PUT(":id", api.Controllers.OrgApi.UpdateOrg)

		// 详情查询
		orgApi.GET(":id", api.Controllers.OrgApi.GetOrgByID)

		// 分页查询
		orgApi.GET("", api.Controllers.OrgApi.GetOrgList)

		// 树结构查询
		orgApi.GET("tree", api.Controllers.OrgApi.GetOrgTree)
	}
}
