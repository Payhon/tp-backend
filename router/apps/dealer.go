package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type Dealer struct{}

func (*Dealer) InitDealer(Router *gin.RouterGroup) {
	// 经销商路由
	dealerApi := Router.Group("dealer")
	{
		// 增
		dealerApi.POST("", api.Controllers.DealerApi.CreateDealer)

		// 删
		dealerApi.DELETE(":id", api.Controllers.DealerApi.DeleteDealer)

		// 改
		dealerApi.PUT(":id", api.Controllers.DealerApi.UpdateDealer)

		// 详情查询
		dealerApi.GET(":id", api.Controllers.DealerApi.GetDealerByID)

		// 穿透概览
		dealerApi.GET(":id/overview", api.Controllers.DealerApi.GetDealerOverview)

		// 权限模板
		dealerApi.GET(":id/permission_template", api.Controllers.DealerApi.GetDealerPermissionTemplate)
		dealerApi.PUT(":id/permission_template", api.Controllers.DealerApi.SetDealerPermissionTemplate)

		// 分页查询
		dealerApi.GET("", api.Controllers.DealerApi.GetDealerList)
	}
}
