package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/smartcat999/registry-agent/internal/service"
)

type ContainerHandler struct {
	dockerService *service.DockerService
}

func NewContainerHandler(dockerService *service.DockerService) *ContainerHandler {
	return &ContainerHandler{
		dockerService: dockerService,
	}
}

func (h *ContainerHandler) GetContainers(c *gin.Context) {
	containers, err := h.dockerService.ListContainers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, containers)
}

func (h *ContainerHandler) StartContainer(c *gin.Context) {
	id := c.Param("id")
	err := h.dockerService.StartContainer(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Container started successfully"})
}

func (h *ContainerHandler) StopContainer(c *gin.Context) {
	id := c.Param("id")
	err := h.dockerService.StopContainer(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Container stopped successfully"})
}

func (h *ContainerHandler) GetContainerDetail(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.dockerService.GetContainerDetail(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *ContainerHandler) GetContainerLogs(c *gin.Context) {
	id := c.Param("id")
	logs, err := h.dockerService.GetContainerLogs(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.String(http.StatusOK, logs)
}

func (h *ContainerHandler) DeleteContainer(c *gin.Context) {
	id := c.Param("id")
	force := c.Query("force") == "true"

	err := h.dockerService.DeleteContainer(id, force)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Container deleted successfully"})
}

func (h *ContainerHandler) ListContainers(c *gin.Context) {
	containers, err := h.dockerService.ListContainers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, containers)
}
