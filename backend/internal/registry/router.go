package registry

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Router 定义API路由器
type Router struct {
	handler *Handler
	engine  *gin.Engine
}

// NewRouter 创建新的路由器
func NewRouter(handler *Handler) *Router {
	router := &Router{
		handler: handler,
		engine:  gin.New(),
	}

	// 使用Gin的Recovery中间件
	router.engine.Use(gin.Recovery())

	// 注册API路由
	router.registerRoutes()
	return router
}

// ServeHTTP 实现http.Handler接口
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 直接委托给Gin的Engine处理
	router.engine.ServeHTTP(w, r)
}

// 注册API路由
func (router *Router) registerRoutes() {
	// 设置通用中间件
	router.engine.Use(func(c *gin.Context) {
		// 标准化路径
		c.Request.URL.Path = normalizePathV2(c.Request.URL.Path)
		// 设置Docker版本响应头
		c.Header("Docker-Distribution-API-Version", "registry/2.0")
		c.Next()
	})

	// 使用单一路由处理所有请求，避免Gin路由冲突
	router.engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 确保是/v2开头的路径
		if !strings.HasPrefix(path, "/v2") {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}

		// 处理API版本检查
		if path == "/v2/" || path == "/v2" {
			router.handler.handleVersionCheck(c)
			return
		}

		// 处理仓库目录
		if path == "/v2/_catalog" || path == "/v2/_catalog/" {
			router.handler.handleCatalog(c)
			return
		}

		// 移除前缀"/v2/"
		subPath := strings.TrimPrefix(path, "/v2/")
		parts := strings.Split(subPath, "/")
		if len(parts) < 2 {
			c.String(http.StatusNotFound, "404 page not found")
			return
		}

		// 解析各种API路径模式
		// 查找操作类型(manifests, tags, blobs)的位置
		manifestsIndex := -1
		tagsIndex := -1
		blobsIndex := -1

		for i, part := range parts {
			if part == "manifests" {
				manifestsIndex = i
			} else if part == "tags" {
				tagsIndex = i
			} else if part == "blobs" {
				blobsIndex = i
			}
		}

		// 处理清单操作: /v2/{name}/manifests/{reference}
		if manifestsIndex > 0 && manifestsIndex < len(parts)-1 {
			// 提取仓库名称和引用
			repository := strings.Join(parts[:manifestsIndex], "/")
			reference := parts[manifestsIndex+1]

			log.Printf("解析清单请求: 仓库=%s, 引用=%s, 方法=%s", repository, reference, c.Request.Method)
			c.Set("repository", repository)
			c.Set("reference", reference)

			router.handler.handleManifests(c)
			return
		}

		// 处理标签列表: /v2/{name}/tags/list
		if tagsIndex > 0 && tagsIndex+1 < len(parts) && parts[tagsIndex+1] == "list" {
			repository := strings.Join(parts[:tagsIndex], "/")

			log.Printf("解析标签列表请求: 仓库=%s", repository)
			c.Set("repository", repository)

			router.handler.handleListTags(c)
			return
		}

		// 处理Blob操作: /v2/{name}/blobs/{digest} 或上传操作
		if blobsIndex > 0 {
			repository := strings.Join(parts[:blobsIndex], "/")

			// 处理上传初始化: /v2/{name}/blobs/uploads/
			if blobsIndex+1 < len(parts) && parts[blobsIndex+1] == "uploads" {
				if (blobsIndex+2 >= len(parts) || parts[blobsIndex+2] == "") && c.Request.Method == http.MethodPost {
					// 上传初始化POST请求
					log.Printf("解析上传初始化请求: 仓库=%s", repository)
					c.Set("repository", repository)

					router.handler.handleInitiateUpload(c)
					return
				} else if blobsIndex+2 < len(parts) {
					// 处理上传操作: /v2/{name}/blobs/uploads/{uuid}
					uuid := parts[blobsIndex+2]

					log.Printf("解析上传请求: 仓库=%s, uuid=%s, 方法=%s", repository, uuid, c.Request.Method)
					c.Set("repository", repository)
					c.Set("uuid", uuid)

					router.handler.handleUpload(c)
					return
				}
			} else if blobsIndex+1 < len(parts) {
				// 处理普通Blob操作: /v2/{name}/blobs/{digest}
				digest := parts[blobsIndex+1]

				log.Printf("解析Blob请求: 仓库=%s, digest=%s, 方法=%s", repository, digest, c.Request.Method)
				c.Set("repository", repository)
				c.Set("digest", digest)

				router.handler.handleBlobs(c)
				return
			}
		}

		// 如果没有匹配的路由，返回404
		c.String(http.StatusNotFound, "404 page not found")
	})
}

// 标准化V2 API路径
func normalizePathV2(path string) string {
	// 确保路径以/v2开头
	if !strings.HasPrefix(path, "/v2") {
		return path
	}

	// 移除连续斜杠
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	// 确保路径以斜杠结尾
	if path != "/v2/" && !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	return path
}
