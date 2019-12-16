package routers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"tgin/controllers/api"
	"tgin/middleware"
	"tgin/pkg/setting"
	"tgin/pkg/upload"
)

func InitRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	gin.SetMode(setting.ServerSetting.RunMode)
	r.GET("/auth", api.GetAuth)
	r.POST("/upload", api.UploadImage)
	//静态访问图片资源
	r.StaticFS("/upload/images", http.Dir(upload.GetImageFullPath()))

	apiv1 := r.Group("/api/v1").Use(middleware.JWT())
	{
		//获取标签列表
		apiv1.GET("/tags", api.GetTags)
		//新建标签
		apiv1.POST("/tags", api.AddTag)
		//更新指定标签
		apiv1.PUT("/tags/:id", api.EditTag)
		//删除指定标签
		apiv1.DELETE("/tags/:id", api.DeleteTag)

		//获取文章列表
		apiv1.GET("/articles", api.GetArticles)
		//获取指定文章
		apiv1.GET("/articles/:id", api.GetArticle)
		//新建文章
		apiv1.POST("/articles", api.AddArticle)
		//更新指定文章
		apiv1.PUT("/articles/:id", api.EditArticle)
		//删除指定文章
		apiv1.DELETE("/articles/:id", api.DeleteArticle)
	}

	return r
}
