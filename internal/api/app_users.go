package api

import (
	"project/internal/model"
	"project/internal/service"
	"project/pkg/utils"

	"github.com/gin-gonic/gin"
)

type AppUsersApi struct{}

// SourceOptions APP用户来源筛选树
// @Summary APP用户来源筛选树
// @Tags APP-Users
// @Produce json
// @Success 200 {array} model.AppUserSourceOptionResp
// @Router /api/v1/app_users/source-options [get]
func (*AppUsersApi) SourceOptions(c *gin.Context) {
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppUsers.SourceOptions(c.Request.Context(), claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}

// List APP用户增强列表
// @Summary APP用户增强列表
// @Tags APP-Users
// @Produce json
// @Param page query int true "页码"
// @Param page_size query int true "每页数量"
// @Param source_type query string true "来源类型 APP/WXMP"
// @Param wx_appid query string false "微信小程序 AppID"
// @Param keyword query string false "关键词"
// @Param status query string false "状态 N/F"
// @Success 200 {object} model.AppUserListResp
// @Router /api/v1/app_users [get]
func (*AppUsersApi) List(c *gin.Context) {
	var req model.AppUserListReq
	if !BindAndValidate(c, &req) {
		return
	}
	claims := c.MustGet("claims").(*utils.UserClaims)
	data, err := service.GroupApp.AppUsers.List(c.Request.Context(), req, claims)
	if err != nil {
		c.Error(err)
		return
	}
	c.Set("data", data)
}
