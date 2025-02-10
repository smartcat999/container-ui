package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/smartcat999/registry-agent/internal/service"
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

func (h *ContextHandler) GetCurrentContext(c *gin.Context) {
	context, err := h.dockerService.GetCurrentContext()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, context)
}

func (h *ContextHandler) SwitchContext(c *gin.Context) {
	name := c.Param("name")
	err := h.dockerService.SwitchContext(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Context switched successfully"})
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

func (h *ContextHandler) GetDefaultContextConfig(c *gin.Context) {
	host, err := h.dockerService.GetDefaultContextConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"host": host})
}

func (h *ContextHandler) UpdateDefaultContext(c *gin.Context) {
	var config struct {
		Host string `json:"host"`
	}
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.dockerService.UpdateDefaultContext(config.Host)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Default context updated successfully"})
}

func (h *ContextHandler) DeleteContext(c *gin.Context) {
	name := c.Param("name")
	if name == "default" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete default context"})
		return
	}

	err := h.dockerService.DeleteContext(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Context deleted successfully"})
}

func (h *ContextHandler) GetContextConfig(c *gin.Context) {
	name := c.Param("name")
	host, err := h.dockerService.GetContextConfig(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"host": host})
}

func (h *ContextHandler) UpdateContextConfig(c *gin.Context) {
	name := c.Param("name")
	var config struct {
		Host string `json:"host"`
	}
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.dockerService.UpdateContextConfig(name, config.Host)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Context updated successfully"})
}
