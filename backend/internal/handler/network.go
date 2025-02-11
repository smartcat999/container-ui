package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smartcat999/container-ui/internal/service"
)

type NetworkHandler struct {
	dockerService *service.DockerService
}

func NewNetworkHandler(dockerService *service.DockerService) *NetworkHandler {
	return &NetworkHandler{
		dockerService: dockerService,
	}
}

// GetNetworks 获取网络列表
func (h *NetworkHandler) GetNetworks(c *gin.Context) {
	contextName := c.Param("context")
	networks, err := h.dockerService.ListNetworks(contextName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, networks)
}

// GetNetworkDetail 获取网络详情
func (h *NetworkHandler) GetNetworkDetail(c *gin.Context) {
	contextName := c.Param("context")
	id := c.Param("id")
	detail, err := h.dockerService.GetNetworkDetail(contextName, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// DeleteNetwork 删除网络
func (h *NetworkHandler) DeleteNetwork(c *gin.Context) {
	contextName := c.Param("context")
	id := c.Param("id")
	err := h.dockerService.DeleteNetwork(contextName, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Network deleted successfully"})
}
