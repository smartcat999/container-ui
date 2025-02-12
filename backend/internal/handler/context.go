package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/smartcat999/container-ui/internal/service"
)

type ContextHandler struct {
	dockerService *service.DockerService
}

func NewContextHandler(dockerService *service.DockerService) *ContextHandler {
	return &ContextHandler{
		dockerService: dockerService,
	}
}

func (h *ContextHandler) ListContexts(c *gin.Context) {
	contexts, err := h.dockerService.ListContexts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, contexts)
}

func (h *ContextHandler) CreateContext(c *gin.Context) {
	var config service.ContextConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.dockerService.CreateContext(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Context created successfully"})
}

func (h *ContextHandler) DeleteContext(c *gin.Context) {
	name := c.Param("context")
	err := h.dockerService.DeleteContext(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Context deleted successfully"})
}

func (h *ContextHandler) GetContextConfig(c *gin.Context) {
	name := c.Param("context")
	host, err := h.dockerService.GetContextConfig(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"host": host})
}

func (h *ContextHandler) UpdateContextConfig(c *gin.Context) {
	name := c.Param("context")
	var config service.ContextConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.dockerService.UpdateContextConfig(name, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Context updated successfully"})
}

// GetServerInfo 获取服务器信息
func (h *ContextHandler) GetServerInfo(c *gin.Context) {
	contextName := c.Param("context")
	info, err := h.dockerService.GetServerInfo(contextName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}
