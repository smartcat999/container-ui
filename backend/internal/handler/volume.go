package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/smartcat999/container-ui/internal/service"
)

type VolumeHandler struct {
	dockerService *service.DockerService
}

func NewVolumeHandler(dockerService *service.DockerService) *VolumeHandler {
	return &VolumeHandler{
		dockerService: dockerService,
	}
}

// GetVolumes 获取数据卷列表
func (h *VolumeHandler) GetVolumes(c *gin.Context) {
	contextName := c.Param("context")
	volumes, err := h.dockerService.ListVolumes(contextName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, volumes)
}

// GetVolumeDetail 获取数据卷详情
func (h *VolumeHandler) GetVolumeDetail(c *gin.Context) {
	contextName := c.Param("context")
	name := c.Param("name")
	detail, err := h.dockerService.GetVolumeDetail(contextName, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// DeleteVolume 删除数据卷
func (h *VolumeHandler) DeleteVolume(c *gin.Context) {
	contextName := c.Param("context")
	name := c.Param("name")
	err := h.dockerService.DeleteVolume(contextName, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Volume deleted successfully"})
}
