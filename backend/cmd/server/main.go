package main

import (
	"log"

	"github.com/smartcat999/registry-agent/internal/handler"
	"github.com/smartcat999/registry-agent/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
		// 容器相关路由
		api.GET("/containers", containerHandler.ListContainers)
		api.POST("/containers/:id/start", containerHandler.StartContainer)
		api.POST("/containers/:id/stop", containerHandler.StopContainer)
		api.DELETE("/containers/:id", containerHandler.DeleteContainer)
		api.GET("/containers/:id/json", containerHandler.GetContainerDetail)
		api.GET("/containers/:id/logs", containerHandler.GetContainerLogs)
		api.GET("/containers/:id/exec", containerHandler.ExecContainer)

		// 镜像相关路由
		api.GET("/images", imageHandler.GetImages)
		api.DELETE("/images/:id", imageHandler.DeleteImage)
		api.POST("/containers", imageHandler.CreateContainer)
		api.GET("/images/:id/json", imageHandler.GetImageDetail)

		// 网络相关路由
		api.GET("/networks", networkHandler.GetNetworks)
		api.GET("/networks/:id", networkHandler.GetNetworkDetail)
		api.DELETE("/networks/:id", networkHandler.DeleteNetwork)

		// 数据卷相关路由
		api.GET("/volumes", volumeHandler.GetVolumes)
		api.GET("/volumes/:name", volumeHandler.GetVolumeDetail)
		api.DELETE("/volumes/:name", volumeHandler.DeleteVolume)

		// 上下文相关路由
		api.GET("/contexts", contextHandler.ListContexts)
		api.GET("/contexts/current", contextHandler.GetCurrentContext)
		api.POST("/contexts/:name/use", contextHandler.SwitchContext)
		api.POST("/contexts", contextHandler.CreateContext)
		api.GET("/contexts/:name", contextHandler.GetContextConfig)
		api.PUT("/contexts/:name", contextHandler.UpdateContextConfig)
		api.DELETE("/contexts/:name", contextHandler.DeleteContext)
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
