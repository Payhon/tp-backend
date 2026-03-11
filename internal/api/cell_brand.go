package api

import (
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type CellBrandApi struct{}

// CreateCellBrand 新增电芯品牌
// @Router /api/v1/battery/cell-brand [post]
func (*CellBrandApi) CreateCellBrand(c *gin.Context) {
	var req model.BatteryCellBrandCreateReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.CellBrand.Create(c, req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// UpdateCellBrand 更新电芯品牌
// @Router /api/v1/battery/cell-brand/{id} [put]
func (*CellBrandApi) UpdateCellBrand(c *gin.Context) {
	id := c.Param("id")
	var req model.BatteryCellBrandUpdateReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.CellBrand.Update(c, id, req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// DeleteCellBrand 删除电芯品牌
// @Router /api/v1/battery/cell-brand/{id} [delete]
func (*CellBrandApi) DeleteCellBrand(c *gin.Context) {
	id := c.Param("id")
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	if err := service.GroupApp.CellBrand.Delete(c, id, userClaims); err != nil {
		c.Error(err)
		return
	}
	c.Set("data", nil)
}

// ListCellBrands 电芯品牌列表
// @Router /api/v1/battery/cell-brand [get]
func (*CellBrandApi) ListCellBrands(c *gin.Context) {
	var req model.BatteryCellBrandListReq
	if !BindAndValidate(c, &req) {
		return
	}
	userClaims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.CellBrand.List(c, req, userClaims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
