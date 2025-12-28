package apps

import (
	"project/internal/api"

	"github.com/gin-gonic/gin"
)

type UpLoad struct{}

func (*UpLoad) Init(Router *gin.RouterGroup) {
	uploadapi := Router.Group("file")
	{
		// 文件上传
		uploadapi.POST("up", api.Controllers.UpLoadApi.UpFile)

		// 文件列表（当前租户）
		uploadapi.GET("list", api.Controllers.FileApi.GetFileListByPage)

		// 云存储直传（获取参数 + 登记）
		uploadapi.POST("cloud/credential", api.Controllers.FileApi.CreateCloudUploadCredential)
		uploadapi.POST("cloud/register", api.Controllers.FileApi.RegisterCloudFile)

		// 文件存储配置（系统设置，SYS_ADMIN）
		uploadapi.GET("storage/config", api.Controllers.FileApi.GetFileStorageConfig)
		uploadapi.PUT("storage/config", api.Controllers.FileApi.UpsertFileStorageConfig)
	}
}
