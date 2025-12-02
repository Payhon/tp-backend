package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

// Warranty 维保路由
type Warranty struct{}

func (*Warranty) InitWarranty(Router *gin.RouterGroup) {
	warrantyApi := Router.Group("warranty")
	{
		// 创建维保申请
		warrantyApi.POST("", api.Controllers.WarrantyApi.CreateWarrantyApplication)

		// 维保申请列表
		warrantyApi.GET("", api.Controllers.WarrantyApi.GetWarrantyList)

		// 维保详情
		warrantyApi.GET(":id", api.Controllers.WarrantyApi.GetWarrantyDetail)

		// 更新维保状态
		warrantyApi.PUT(":id", api.Controllers.WarrantyApi.UpdateWarrantyStatus)
	}
}

