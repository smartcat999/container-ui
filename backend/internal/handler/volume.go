package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	
	"github.com/smartcat999/registry-agent/internal/service"
)

type VolumeHandler struct {
	dockerService *service.DockerService
}

func NewVolumeHandler(dockerService *service.DockerService) *VolumeHandler {
	return &VolumeHandler{
		dockerService: dockerService,
	}
}

func (h *VolumeHandler) GetVolumes(c *gin.Context) {
	volumes, err := h.dockerService.ListVolumes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, volumes)
}

func (h *VolumeHandler) GetVolumeDetail(c *gin.Context) {
	name := c.Param("name")
	detail, err := h.dockerService.GetVolumeDetail(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *VolumeHandler) DeleteVolume(c *gin.Context) {
	name := c.Param("name")
	err := h.dockerService.DeleteVolume(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Volume deleted successfully"})
}
