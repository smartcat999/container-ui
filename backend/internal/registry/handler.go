package registry

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smartcat999/container-ui/internal/storage"
)

// Manifest 定义 Docker 镜像清单结构
type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int64  `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int64  `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

// ManifestList 定义多架构镜像清单列表结构
type ManifestList struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Manifests     []struct {
		MediaType string `json:"mediaType"`
		Size      int64  `json:"size"`
		Digest    string `json:"digest"`
		Platform  struct {
			Architecture string   `json:"architecture"`
			OS           string   `json:"os"`
			OSVersion    string   `json:"os.version,omitempty"`
			OSFeatures   []string `json:"os.features,omitempty"`
			Variant      string   `json:"variant,omitempty"`
			Features     []string `json:"features,omitempty"`
		} `json:"platform,omitempty"`
	} `json:"manifests"`
}

// 媒体类型常量
const (
	MediaTypeManifestV2       = "application/vnd.docker.distribution.manifest.v2+json"
	MediaTypeManifestList     = "application/vnd.docker.distribution.manifest.list.v2+json"
	MediaTypeOCIManifestV1    = "application/vnd.oci.image.manifest.v1+json"
	MediaTypeOCIManifestIndex = "application/vnd.oci.image.index.v1+json"
)

// Handler 处理镜像仓库请求
type Handler struct {
	storage storage.Storage
}

// NewHandler 创建新的处理器
func NewHandler(storage storage.Storage) *Handler {
	return &Handler{
		storage: storage,
	}
}

// 检测清单类型
func detectManifestMediaType(data []byte) string {
	// 尝试解析为标准格式
	var m struct {
		MediaType     string `json:"mediaType"`
		SchemaVersion int    `json:"schemaVersion"`
		Manifests     []any  `json:"manifests"`
	}

	if err := json.Unmarshal(data, &m); err != nil {
		return MediaTypeManifestV2 // 解析失败，默认为v2格式
	}

	// 检查是否包含 manifests 字段
	if m.SchemaVersion == 2 && len(m.Manifests) > 0 {
		// 根据指定的mediaType或类型猜测
		if m.MediaType == MediaTypeManifestList {
			return MediaTypeManifestList
		} else if m.MediaType == MediaTypeOCIManifestIndex {
			return MediaTypeOCIManifestIndex
		}
		return MediaTypeManifestList // 默认为Docker清单列表格式
	}

	// 使用声明的媒体类型，如果有的话
	if m.MediaType != "" {
		return m.MediaType
	}

	return MediaTypeManifestV2 // 默认为清单v2格式
}

// validateManifest 验证清单格式
func (h *Handler) validateManifest(data []byte, mediaType string) error {
	var schemaVersion int
	var manifestError error

	if mediaType == MediaTypeManifestV2 {
		var manifest Manifest
		manifestError = json.Unmarshal(data, &manifest)
		schemaVersion = manifest.SchemaVersion
	} else if mediaType == MediaTypeManifestList || mediaType == MediaTypeOCIManifestIndex {
		var manifestList ManifestList
		manifestError = json.Unmarshal(data, &manifestList)
		schemaVersion = manifestList.SchemaVersion
	} else {
		// 尝试解析为普通JSON
		var genericManifest map[string]interface{}
		manifestError = json.Unmarshal(data, &genericManifest)
		if v, ok := genericManifest["schemaVersion"].(float64); ok {
			schemaVersion = int(v)
		}
	}

	if manifestError != nil {
		return fmt.Errorf("Invalid manifest format: %v", manifestError)
	}

	if schemaVersion != 2 {
		return fmt.Errorf("Unsupported manifest schema version")
	}

	return nil
}

// generateUploadID 生成上传 ID
func generateUploadID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(8))
}

// randomString 生成随机字符串
func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}

// ================ HTTP 处理函数 ================

// handleVersionCheck 处理API版本检查
func (h *Handler) handleVersionCheck(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]string{})
}

// handleCatalog 处理仓库列表
func (h *Handler) handleCatalog(c *gin.Context) {
	repositories, err := h.storage.ListRepositories()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"repositories": repositories,
	})
}

// handleListTags 处理标签列表
func (h *Handler) handleListTags(c *gin.Context) {
	// 获取完整的仓库路径
	var repositoryPath string

	// 优先从上下文中获取
	if repo, exists := c.Get("repository"); exists {
		repositoryPath = repo.(string)
	} else {
		repositoryPath = c.Param("repository")
	}

	// 打印调试信息
	log.Printf("处理标签列表请求: repository=%s, URL=%s", repositoryPath, c.Request.URL.Path)

	if repositoryPath == "" {
		c.String(http.StatusBadRequest, "Repository not specified")
		return
	}

	tags, err := h.storage.ListTags(repositoryPath)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"name": repositoryPath,
		"tags": tags,
	})
}

// handleManifests 处理清单
func (h *Handler) handleManifests(c *gin.Context) {
	// 获取完整的仓库路径
	var repositoryPath string
	var reference string

	// 优先从上下文中获取
	if repo, exists := c.Get("repository"); exists {
		repositoryPath = repo.(string)
	} else {
		repositoryPath = c.Param("repository")
	}

	if ref, exists := c.Get("reference"); exists {
		reference = ref.(string)
	} else {
		reference = c.Param("reference")
	}

	// 打印调试信息，帮助诊断问题
	log.Printf("处理manifest请求: repository=%s, reference=%s, URL=%s", repositoryPath, reference, c.Request.URL.Path)

	if repositoryPath == "" || reference == "" {
		c.String(http.StatusBadRequest, "Repository or reference not specified")
		return
	}

	switch c.Request.Method {
	case http.MethodHead:
		h.handleHeadManifest(c, repositoryPath, reference)
	case http.MethodGet:
		h.handleGetManifest(c, repositoryPath, reference)
	case http.MethodPut:
		h.handlePutManifest(c, repositoryPath, reference)
	case http.MethodDelete:
		h.handleDeleteManifest(c, repositoryPath, reference)
	default:
		c.String(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleHeadManifest 处理HEAD请求，检查manifest是否存在
func (h *Handler) handleHeadManifest(c *gin.Context, repository, reference string) {
	// 检查 manifest 是否存在
	manifest, digest, err := h.storage.GetManifest(repository, reference)
	if err != nil {
		// 设置响应头
		c.Header("Content-Type", MediaTypeManifestV2)
		c.Header("Docker-Content-Digest", "")
		c.String(http.StatusNotFound, "manifest unknown")
		return
	}

	// 检测清单类型
	mediaType := detectManifestMediaType(manifest)

	// 设置响应头
	c.Header("Content-Type", mediaType)
	c.Header("Docker-Content-Digest", digest)
	c.Header("Content-Length", fmt.Sprintf("%d", len(manifest)))
	c.Status(http.StatusOK)
}

// handleGetManifest 处理GET请求，获取manifest内容
func (h *Handler) handleGetManifest(c *gin.Context, repository, reference string) {
	var manifest []byte
	var digest string
	var err error

	// 检查是否是 digest 请求
	if strings.HasPrefix(reference, "sha256:") {
		// 如果是 digest 请求，直接返回对应的 manifest
		manifest, digest, err = h.storage.GetManifestByDigest(repository, reference)
	} else {
		// 如果是 tag 请求，通过 tag 获取 manifest
		manifest, digest, err = h.storage.GetManifest(repository, reference)
	}

	if err != nil {
		// 设置响应头
		c.Header("Content-Type", MediaTypeManifestV2)
		c.Header("Docker-Content-Digest", "")
		c.String(http.StatusNotFound, "manifest unknown")
		return
	}

	// 检测清单类型
	mediaType := detectManifestMediaType(manifest)
	c.Header("Content-Type", mediaType)
	c.Header("Docker-Content-Digest", digest)
	c.Data(http.StatusOK, mediaType, manifest)
}

// handlePutManifest 处理PUT请求，上传manifest
func (h *Handler) handlePutManifest(c *gin.Context, repository, reference string) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	// 计算请求体的 digest
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(body))

	// 检测清单类型
	mediaType := detectManifestMediaType(body)

	// 验证 manifest 格式
	if err := h.validateManifest(body, mediaType); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	// 确保 manifest 目录存在
	manifestDir := filepath.Join(h.storage.(*storage.FileStorage).RootDir(), "repositories", repository, "_manifests")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to create manifest directory: %v", err))
		return
	}

	if err := h.storage.PutManifest(repository, reference, digest, body); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Docker-Content-Digest", digest)
	c.Status(http.StatusCreated)
}

// handleDeleteManifest 处理DELETE请求，删除manifest
func (h *Handler) handleDeleteManifest(c *gin.Context, repository, reference string) {
	if err := h.storage.DeleteManifest(repository, reference); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusAccepted)
}

// handleBlobs 处理 blob
func (h *Handler) handleBlobs(c *gin.Context) {
	// 获取完整的仓库路径
	var repositoryPath string
	var digest string

	// 优先从上下文中获取
	if repo, exists := c.Get("repository"); exists {
		repositoryPath = repo.(string)
	} else {
		repositoryPath = c.Param("repository")
	}

	if dig, exists := c.Get("digest"); exists {
		digest = dig.(string)
	} else {
		digest = c.Param("digest")
	}

	// 打印调试信息
	log.Printf("处理blob请求: repository=%s, digest=%s, URL=%s", repositoryPath, digest, c.Request.URL.Path)

	if repositoryPath == "" || digest == "" {
		c.String(http.StatusBadRequest, "Repository or digest not specified")
		return
	}

	switch c.Request.Method {
	case http.MethodHead:
		h.handleHeadBlob(c, repositoryPath, digest)
	case http.MethodGet:
		h.handleGetBlob(c, repositoryPath, digest)
	case http.MethodDelete:
		h.handleDeleteBlob(c, repositoryPath, digest)
	default:
		c.String(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleHeadBlob 处理HEAD请求，检查blob是否存在
func (h *Handler) handleHeadBlob(c *gin.Context, repository, digest string) {
	// 检查 blob 是否存在
	size, err := h.storage.GetBlobSize(repository, digest)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Docker-Content-Digest", digest)
	c.Header("Content-Length", fmt.Sprintf("%d", size))
	c.Status(http.StatusOK)
}

// handleGetBlob 处理GET请求，获取blob内容
func (h *Handler) handleGetBlob(c *gin.Context, repository, digest string) {
	reader, size, err := h.storage.GetBlob(repository, digest)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}
	defer reader.Close()

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Docker-Content-Digest", digest)
	c.Header("Content-Length", fmt.Sprintf("%d", size))

	// 使用Gin的Reader函数将blob流式传输到客户端
	c.DataFromReader(http.StatusOK, size, "application/octet-stream", reader, nil)
}

// handleDeleteBlob 处理DELETE请求，删除blob
func (h *Handler) handleDeleteBlob(c *gin.Context, repository, digest string) {
	if err := h.storage.DeleteBlob(repository, digest); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusAccepted)
}

// handleInitiateUpload 处理上传初始化
func (h *Handler) handleInitiateUpload(c *gin.Context) {
	// 获取完整的仓库路径
	var repositoryPath string

	// 优先从上下文中获取
	if repo, exists := c.Get("repository"); exists {
		repositoryPath = repo.(string)
	} else {
		repositoryPath = c.Param("repository")
	}

	// 打印调试信息
	log.Printf("处理上传初始化请求: repository=%s, URL=%s", repositoryPath, c.Request.URL.Path)

	if repositoryPath == "" {
		c.String(http.StatusBadRequest, "Repository not specified")
		return
	}

	// 生成上传 ID
	uploadID := generateUploadID()

	// 创建上传路径
	if err := h.storage.InitiateUpload(repositoryPath, uploadID); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// 设置响应头
	c.Header("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", repositoryPath, uploadID))
	c.Header("Range", "0-0")
	c.Header("Docker-Upload-UUID", uploadID)
	c.Status(http.StatusAccepted)
}

// handleUpload 处理上传
func (h *Handler) handleUpload(c *gin.Context) {
	// 获取完整的仓库路径
	var repositoryPath string
	var uploadID string

	// 优先从上下文中获取
	if repo, exists := c.Get("repository"); exists {
		repositoryPath = repo.(string)
	} else {
		repositoryPath = c.Param("repository")
	}

	if uuid, exists := c.Get("uuid"); exists {
		uploadID = uuid.(string)
	} else {
		uploadID = c.Param("uuid")
	}

	// 打印调试信息
	log.Printf("处理上传请求: repository=%s, uploadID=%s, URL=%s", repositoryPath, uploadID, c.Request.URL.Path)

	if repositoryPath == "" || uploadID == "" {
		c.String(http.StatusBadRequest, "Repository or upload ID not specified")
		return
	}

	switch c.Request.Method {
	case http.MethodPatch:
		h.handlePatchUpload(c, repositoryPath, uploadID)
	case http.MethodPut:
		h.handlePutUpload(c, repositoryPath, uploadID)
	default:
		c.String(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handlePatchUpload 处理PATCH请求，追加上传数据
func (h *Handler) handlePatchUpload(c *gin.Context, repository, uploadID string) {
	// 追加数据
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	offset, err := h.storage.AppendToUpload(repository, uploadID, body)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", repository, uploadID))
	c.Header("Range", fmt.Sprintf("0-%d", offset-1))
	c.Header("Docker-Upload-UUID", uploadID)
	c.Status(http.StatusAccepted)
}

// handlePutUpload 处理PUT请求，完成上传
func (h *Handler) handlePutUpload(c *gin.Context, repository, uploadID string) {
	// 完成上传
	digest := c.Query("digest")
	if digest == "" {
		c.String(http.StatusBadRequest, "Digest parameter required")
		return
	}

	// 处理可能的剩余数据
	var body []byte
	if c.Request.ContentLength > 0 {
		body, _ = io.ReadAll(c.Request.Body)
	}

	if err := h.storage.CompleteUpload(repository, uploadID, digest, body); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Docker-Content-Digest", digest)
	c.Header("Location", fmt.Sprintf("/v2/%s/blobs/%s", repository, digest))
	c.Status(http.StatusCreated)
}
