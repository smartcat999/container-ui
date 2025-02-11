package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smartcat999/container-ui/internal/service"
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
		ImageID string   `json:"imageId"`
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Ports   []struct {
			Host      uint16 `json:"host"`
			Container uint16 `json:"container"`
		} `json:"ports"`
		Env []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"env"`
		Volumes []struct {
			Host      string `json:"host"`
			Container string `json:"container"`
			Mode      string `json:"mode"`
		} `json:"volumes"`
		RestartPolicy string `json:"restartPolicy"`
		NetworkMode   string `json:"networkMode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := service.ContainerConfig{
		ImageID:       req.ImageID,
		Name:          req.Name,
		Command:       req.Command,
		Args:          req.Args,
		Ports:         make([]service.PortMapping, len(req.Ports)),
		Env:           make([]service.EnvVar, len(req.Env)),
		Volumes:       make([]service.VolumeMapping, len(req.Volumes)),
		RestartPolicy: req.RestartPolicy,
		NetworkMode:   req.NetworkMode,
	}

	for i, p := range req.Ports {
		config.Ports[i] = service.PortMapping{
			Host:      p.Host,
			Container: p.Container,
		}
	}

	for i, e := range req.Env {
		config.Env[i] = service.EnvVar{
			Key:   e.Key,
			Value: e.Value,
		}
	}

	for i, v := range req.Volumes {
		config.Volumes[i] = service.VolumeMapping{
			Host:      v.Host,
			Container: v.Container,
			Mode:      v.Mode,
		}
	}

	err := h.dockerService.CreateContainer(config)
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
