package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type CellBrand struct{}

func (*CellBrand) InitCellBrand(Router *gin.RouterGroup) {
	cellBrandApi := Router.Group("battery/cell-brand")
	{
		cellBrandApi.POST("", api.Controllers.CellBrandApi.CreateCellBrand)
		cellBrandApi.PUT(":id", api.Controllers.CellBrandApi.UpdateCellBrand)
		cellBrandApi.DELETE(":id", api.Controllers.CellBrandApi.DeleteCellBrand)
		cellBrandApi.GET("", api.Controllers.CellBrandApi.ListCellBrands)
	}
}
