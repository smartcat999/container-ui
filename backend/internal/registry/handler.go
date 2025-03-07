package registry

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// HandleV2 处理 V2 API 请求
func (h *Handler) HandleV2(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v2/")
	// 移除末尾的斜杠，避免产生空字符串
	path = strings.TrimRight(path, "/")
	parts := strings.Split(path, "/")

	switch {
	case path == "":
		// 检查 API 版本
		h.handleVersionCheck(w, r)
	case parts[0] == "_catalog":
		// 获取仓库列表
		h.handleCatalog(w, r)
	case len(parts) >= 2:
		// 查找特殊路径标记
		var repository string
		var remainingParts []string
		for i, part := range parts {
			if part == "blobs" || part == "manifests" || part == "tags" {
				repository = strings.Join(parts[:i], "/")
				remainingParts = parts[i:]
				break
			}
		}

		if repository == "" {
			http.Error(w, "Invalid repository path", http.StatusBadRequest)
			return
		}

		// 处理剩余路径
		switch {
		case len(remainingParts) == 1 && remainingParts[0] == "tags":
			// 获取标签列表
			h.handleListTags(w, r, repository)
		case len(remainingParts) == 2 && remainingParts[0] == "manifests":
			// 处理清单
			h.handleManifests(w, r, repository, remainingParts[1])
		case remainingParts[0] == "blobs":
			// 处理 blobs 相关请求
			if len(remainingParts) == 2 && remainingParts[1] == "uploads" {
				// 处理上传初始化（POST）
				h.handleInitiateUpload(w, r, repository)
			} else if len(remainingParts) == 3 && remainingParts[1] == "uploads" {
				// 处理上传（PATCH/PUT）
				uploadID := remainingParts[2]
				h.handleUpload(w, r, repository, uploadID)
			} else if len(remainingParts) == 2 {
				// 处理层（HEAD/GET）
				digest := remainingParts[1]
				h.handleBlobs(w, r, repository, digest)
			} else {
				http.Error(w, "Invalid blob path", http.StatusBadRequest)
			}
		}
	}
}

// handleVersionCheck 处理版本检查
func (h *Handler) handleVersionCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusOK)
}

// handleCatalog 处理仓库列表
func (h *Handler) handleCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repositories, err := h.storage.ListRepositories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"repositories": repositories,
	})
}

// handleListTags 处理标签列表
func (h *Handler) handleListTags(w http.ResponseWriter, r *http.Request, repository string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tags, err := h.storage.ListTags(repository)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name": repository,
		"tags": tags,
	})
}

// handleManifests 处理清单
func (h *Handler) handleManifests(w http.ResponseWriter, r *http.Request, repository, reference string) {
	switch r.Method {
	case http.MethodHead:
		// 检查 manifest 是否存在
		manifest, digest, err := h.storage.GetManifest(repository, reference)
		if err != nil {
			// 设置响应头
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
			w.Header().Set("Docker-Content-Digest", "")
			http.Error(w, "manifest unknown", http.StatusNotFound)
			return
		}

		// 设置响应头
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.Header().Set("Docker-Content-Digest", digest)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(manifest)))
		w.WriteHeader(http.StatusOK)

	case http.MethodGet:
		// 检查是否是 digest 请求
		if strings.HasPrefix(reference, "sha256:") {
			// 如果是 digest 请求，直接返回对应的 manifest
			manifest, digest, err := h.storage.GetManifestByDigest(repository, reference)
			if err != nil {
				// 设置响应头
				w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
				w.Header().Set("Docker-Content-Digest", "")
				http.Error(w, "manifest unknown", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
			w.Header().Set("Docker-Content-Digest", digest)
			w.Write(manifest)
		} else {
			// 如果是 tag 请求，通过 tag 获取 manifest
			manifest, digest, err := h.storage.GetManifest(repository, reference)
			if err != nil {
				// 设置响应头
				w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
				w.Header().Set("Docker-Content-Digest", "")
				http.Error(w, "manifest unknown", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
			w.Header().Set("Docker-Content-Digest", digest)
			w.Write(manifest)
		}

	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 计算请求体的 digest
		digest := fmt.Sprintf("sha256:%x", sha256.Sum256(body))

		// 验证 manifest 格式
		var manifest Manifest
		if err := json.Unmarshal(body, &manifest); err != nil {
			http.Error(w, "Invalid manifest format", http.StatusBadRequest)
			return
		}

		if manifest.SchemaVersion != 2 {
			http.Error(w, "Unsupported manifest schema version", http.StatusBadRequest)
			return
		}

		// 确保 manifest 目录存在
		manifestDir := filepath.Join(h.storage.(*storage.FileStorage).RootDir(), "repositories", repository, "_manifests")
		if err := os.MkdirAll(manifestDir, 0755); err != nil {
			http.Error(w, fmt.Sprintf("Failed to create manifest directory: %v", err), http.StatusInternalServerError)
			return
		}

		if err := h.storage.PutManifest(repository, reference, digest, body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Docker-Content-Digest", digest)
		w.WriteHeader(http.StatusCreated)

	case http.MethodDelete:
		if err := h.storage.DeleteManifest(repository, reference); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBlobs 处理层
func (h *Handler) handleBlobs(w http.ResponseWriter, r *http.Request, repository, digest string) {
	switch r.Method {
	case http.MethodHead:
		// 检查层是否存在
		exists, err := h.storage.LayerExists(repository, digest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if exists {
			// 如果层存在，返回 200 OK 和正确的 Content-Length
			layerPath := filepath.Join(h.storage.(*storage.FileStorage).RootDir(), "repositories", repository, "_layers", digest)
			if fileInfo, err := os.Stat(layerPath); err == nil {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
			}
			w.Header().Set("Docker-Content-Digest", digest)
			w.WriteHeader(http.StatusOK)
		} else {
			// 如果层不存在，返回 404 Not Found
			w.WriteHeader(http.StatusNotFound)
		}

	case http.MethodGet:
		// 获取层内容
		reader, err := h.storage.GetLayer(repository, digest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		// 设置正确的响应头
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Docker-Content-Digest", digest)

		// 获取文件大小并设置 Content-Length
		if fileInfo, err := os.Stat(filepath.Join(h.storage.(*storage.FileStorage).RootDir(), "repositories", repository, "_layers", digest)); err == nil {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
		}

		io.Copy(w, reader)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleInitiateUpload 处理上传初始化
func (h *Handler) handleInitiateUpload(w http.ResponseWriter, r *http.Request, repository string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 生成上传ID
	uploadID := generateUploadID()

	// 创建上传目录
	uploadPath := filepath.Join(h.storage.(*storage.FileStorage).RootDir(), "uploads", uploadID)
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回上传URL
	w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", repository, uploadID))
	w.WriteHeader(http.StatusAccepted)
}

// handleUpload 处理上传
func (h *Handler) handleUpload(w http.ResponseWriter, r *http.Request, repository, uploadID string) {
	switch r.Method {
	case http.MethodPatch:
		// 上传数据块
		uploadPath := filepath.Join(h.storage.(*storage.FileStorage).RootDir(), "uploads", uploadID, "data")
		file, err := os.OpenFile(uploadPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		if _, err := io.Copy(file, r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 获取上传进度
		if fileInfo, err := os.Stat(uploadPath); err == nil {
			// panic response EOF error
			//w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
			// Range field must
			w.Header().Set("Range", fmt.Sprintf("0-%d", fileInfo.Size()-1))
		}

		// 设置 Location 头，指向当前上传会话的 URL
		w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", repository, uploadID))
		w.Header().Set("Docker-Upload-UUID", uploadID)
		w.WriteHeader(http.StatusAccepted)

	case http.MethodPut:
		// 完成上传
		digest := r.URL.Query().Get("digest")
		if digest == "" {
			http.Error(w, "digest parameter is required", http.StatusBadRequest)
			return
		}

		// 移动上传的文件到最终位置
		uploadPath := filepath.Join(h.storage.(*storage.FileStorage).RootDir(), "uploads", uploadID, "data")
		targetDir := filepath.Join(h.storage.(*storage.FileStorage).RootDir(), "repositories", repository, "_layers")

		// 确保目标目录存在
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := h.storage.PutLayer(repository, digest, uploadPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 清理上传目录
		os.RemoveAll(filepath.Dir(uploadPath))

		// 设置正确的响应头
		w.Header().Set("Docker-Content-Digest", digest)
		w.Header().Set("Content-Length", "0")
		w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/%s", repository, digest))
		w.WriteHeader(http.StatusCreated)

	case http.MethodHead:
		// 检查上传状态
		uploadPath := filepath.Join(h.storage.(*storage.FileStorage).RootDir(), "uploads", uploadID, "data")
		if fileInfo, err := os.Stat(uploadPath); err == nil {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
			w.Header().Set("Range", fmt.Sprintf("0-%d", fileInfo.Size()-1))
			w.WriteHeader(http.StatusNoContent)
		} else if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodDelete:
		// 删除上传目录
		uploadPath := filepath.Join(h.storage.(*storage.FileStorage).RootDir(), "uploads", uploadID)
		if err := os.RemoveAll(uploadPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// generateUploadID 生成上传ID
func generateUploadID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
