package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/smartcat999/container-ui/internal/handler"
	"github.com/smartcat999/container-ui/internal/service"
)

func main() {
	// 创建 Docker 服务
	dockerService, err := service.NewDockerService()
	if err != nil {
		log.Fatal(err)
	}
	// 创建处理器
	containerHandler := handler.NewContainerHandler(dockerService)
	imageHandler := handler.NewImageHandler(dockerService)
	networkHandler := handler.NewNetworkHandler(dockerService)
	volumeHandler := handler.NewVolumeHandler(dockerService)
	contextHandler := handler.NewContextHandler(dockerService)

	r := gin.Default()

	// 配置CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// API路由组
	api := r.Group("/api")
	{
		// Context 相关路由 - 不需要 context 参数
		api.GET("/contexts", contextHandler.ListContexts)
		api.POST("/contexts", contextHandler.CreateContext)
		api.GET("/contexts/:context", contextHandler.GetContextConfig)
		api.PUT("/contexts/:context", contextHandler.UpdateContextConfig)
		api.DELETE("/contexts/:context", contextHandler.DeleteContext)
		// 新增：获取服务器信息路由
		api.GET("/contexts/:context/info", contextHandler.GetServerInfo)

		// 需要 context 参数的资源路由组
		contextAPI := api.Group("/contexts/:context")
		{
			// 容器相关路由
			contextAPI.GET("/containers", containerHandler.ListContainers)
			contextAPI.POST("/containers/:id/start", containerHandler.StartContainer)
			contextAPI.POST("/containers/:id/stop", containerHandler.StopContainer)
			contextAPI.DELETE("/containers/:id", containerHandler.DeleteContainer)
			contextAPI.GET("/containers/:id/json", containerHandler.GetContainerDetail)
			contextAPI.GET("/containers/:id/logs", containerHandler.GetContainerLogs)
			contextAPI.GET("/containers/:id/exec", containerHandler.ExecContainer)

			// 镜像相关路由
			contextAPI.GET("/images", imageHandler.GetImages)
			contextAPI.DELETE("/images/:id", imageHandler.DeleteImage)
			contextAPI.POST("/containers", imageHandler.CreateContainer)
			contextAPI.GET("/images/:id/json", imageHandler.GetImageDetail)

			// 网络相关路由
			contextAPI.GET("/networks", networkHandler.GetNetworks)
			contextAPI.GET("/networks/:id", networkHandler.GetNetworkDetail)
			contextAPI.DELETE("/networks/:id", networkHandler.DeleteNetwork)

			// 数据卷相关路由
			contextAPI.GET("/volumes", volumeHandler.GetVolumes)
			contextAPI.GET("/volumes/:name", volumeHandler.GetVolumeDetail)
			contextAPI.DELETE("/volumes/:name", volumeHandler.DeleteVolume)
		}
	}

	// 托管静态文件
	r.Static("/assets", "./dist/assets")
	r.StaticFile("/favicon.ico", "./dist/favicon.ico")

	// 所有其他路由返回 index.html
	r.NoRoute(func(c *gin.Context) {
		c.File("./dist/index.html")
	})

	log.Fatal(r.Run(":8080"))
}
