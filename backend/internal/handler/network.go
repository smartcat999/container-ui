package handler

import (
	"github.com/smartcat999/container-ui/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NetworkHandler struct {
	dockerService *service.DockerService
}

func NewNetworkHandler(dockerService *service.DockerService) *NetworkHandler {
	return &NetworkHandler{
		dockerService: dockerService,
	}
}

func (h *NetworkHandler) GetNetworks(c *gin.Context) {
	networks, err := h.dockerService.ListNetworks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, networks)
}

func (h *NetworkHandler) GetNetworkDetail(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.dockerService.GetNetworkDetail(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *NetworkHandler) DeleteNetwork(c *gin.Context) {
	id := c.Param("id")
	err := h.dockerService.DeleteNetwork(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Network deleted successfully"})
}
