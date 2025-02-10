package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smartcat999/registry-agent/internal/service"
)

type ImageHandler struct {
	dockerService *service.DockerService
}

func NewImageHandler(dockerService *service.DockerService) *ImageHandler {
	return &ImageHandler{
		dockerService: dockerService,
	}
}

// GetImages 获取镜像列表
func (h *ImageHandler) GetImages(c *gin.Context) {
	images, err := h.dockerService.ListImages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, images)
}

// DeleteImage 删除镜像
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	id := c.Param("id")
	err := h.dockerService.DeleteImage(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
}

// CreateContainer 从镜像创建容器
func (h *ImageHandler) CreateContainer(c *gin.Context) {
	var req struct {
		ImageID string `json:"imageId"`
		Name    string `json:"name"`
		Ports   []struct {
			Host      string `json:"host"`
			Container string `json:"container"`
		} `json:"ports"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.dockerService.CreateContainer(req.ImageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container created successfully"})
}

// GetImageDetail 获取镜像详情
func (h *ImageHandler) GetImageDetail(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.dockerService.GetImageDetail(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}
