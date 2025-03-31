package storage

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileStorage 实现基于文件系统的存储
type FileStorage struct {
	rootDir string
	mutex   sync.RWMutex
}

// NewFileStorage 创建新的文件存储
func NewFileStorage(rootDir string) (*FileStorage, error) {
	// 确保根目录存在
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %v", err)
	}

	// 创建必要的子目录
	for _, dir := range []string{
		filepath.Join(rootDir, "repositories"),
		filepath.Join(rootDir, "uploads"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return &FileStorage{
		rootDir: rootDir,
	}, nil
}

// RootDir 返回存储根目录
func (s *FileStorage) RootDir() string {
	return s.rootDir
}

// ListRepositories 列出所有仓库
func (s *FileStorage) ListRepositories() ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	repositoriesDir := filepath.Join(s.rootDir, "repositories")
	entries, err := os.ReadDir(repositoriesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read repositories directory: %v", err)
	}

	var repositories []string
	for _, entry := range entries {
		if entry.IsDir() {
			repositories = append(repositories, entry.Name())
		}
	}

	return repositories, nil
}

// ListTags 列出仓库的所有标签
func (s *FileStorage) ListTags(repository string) ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tagsDir := filepath.Join(s.rootDir, "repositories", repository, "tags")
	if _, err := os.Stat(tagsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(tagsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tags directory: %v", err)
	}

	var tags []string
	for _, entry := range entries {
		if !entry.IsDir() {
			tags = append(tags, entry.Name())
		}
	}

	return tags, nil
}

// GetManifest 获取清单
func (s *FileStorage) GetManifest(repository, reference string) ([]byte, string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 首先检查是否是 digest
	if strings.HasPrefix(reference, "sha256:") {
		return s.GetManifestByDigest(repository, reference)
	}

	// 如果是 tag，首先找到对应的 digest
	tagFile := filepath.Join(s.rootDir, "repositories", repository, "tags", reference)
	data, err := os.ReadFile(tagFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read tag file: %v", err)
	}

	digest := string(data)
	return s.GetManifestByDigest(repository, digest)
}

// GetManifestByDigest 通过摘要获取清单
func (s *FileStorage) GetManifestByDigest(repository, digest string) ([]byte, string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	manifestFile := filepath.Join(s.rootDir, "repositories", repository, "_manifests", digest)
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read manifest file: %v", err)
	}

	return data, digest, nil
}

// PutManifest 存储清单
func (s *FileStorage) PutManifest(repository, reference, digest string, manifest []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 确保仓库目录存在
	repoDir := filepath.Join(s.rootDir, "repositories", repository)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %v", err)
	}

	// 确保清单目录存在
	manifestsDir := filepath.Join(repoDir, "_manifests")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return fmt.Errorf("failed to create manifests directory: %v", err)
	}

	// 写入清单文件
	manifestFile := filepath.Join(manifestsDir, digest)
	if err := os.WriteFile(manifestFile, manifest, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %v", err)
	}

	// 如果提供了标签引用，更新标签
	if reference != "" && !strings.HasPrefix(reference, "sha256:") {
		tagsDir := filepath.Join(repoDir, "tags")
		if err := os.MkdirAll(tagsDir, 0755); err != nil {
			return fmt.Errorf("failed to create tags directory: %v", err)
		}

		tagFile := filepath.Join(tagsDir, reference)
		if err := os.WriteFile(tagFile, []byte(digest), 0644); err != nil {
			return fmt.Errorf("failed to write tag file: %v", err)
		}
	}

	return nil
}

// DeleteManifest 删除清单
func (s *FileStorage) DeleteManifest(repository, reference string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 如果是摘要，直接删除清单
	if strings.HasPrefix(reference, "sha256:") {
		manifestFile := filepath.Join(s.rootDir, "repositories", repository, "_manifests", reference)
		if err := os.Remove(manifestFile); err != nil {
			return fmt.Errorf("failed to remove manifest file: %v", err)
		}
		return nil
	}

	// 如果是标签，找到对应的摘要，然后删除标签和清单
	tagFile := filepath.Join(s.rootDir, "repositories", repository, "tags", reference)
	data, err := os.ReadFile(tagFile)
	if err != nil {
		return fmt.Errorf("failed to read tag file: %v", err)
	}

	digest := string(data)

	// 删除标签
	if err := os.Remove(tagFile); err != nil {
		return fmt.Errorf("failed to remove tag file: %v", err)
	}

	// 删除清单
	manifestFile := filepath.Join(s.rootDir, "repositories", repository, "_manifests", digest)
	if err := os.Remove(manifestFile); err != nil {
		return fmt.Errorf("failed to remove manifest file: %v", err)
	}

	return nil
}

// GetBlobSize 获取 blob 大小
func (s *FileStorage) GetBlobSize(repository, digest string) (int64, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	blobFile := filepath.Join(s.rootDir, "repositories", repository, "_blobs", digest)
	info, err := os.Stat(blobFile)
	if err != nil {
		return 0, fmt.Errorf("failed to stat blob file: %v", err)
	}

	return info.Size(), nil
}

// GetBlob 获取 blob
func (s *FileStorage) GetBlob(repository, digest string) (io.ReadCloser, int64, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	blobFile := filepath.Join(s.rootDir, "repositories", repository, "_blobs", digest)
	file, err := os.Open(blobFile)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open blob file: %v", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("failed to stat blob file: %v", err)
	}

	return file, info.Size(), nil
}

// DeleteBlob 删除 blob
func (s *FileStorage) DeleteBlob(repository, digest string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	blobFile := filepath.Join(s.rootDir, "repositories", repository, "_blobs", digest)
	if err := os.Remove(blobFile); err != nil {
		return fmt.Errorf("failed to remove blob file: %v", err)
	}

	return nil
}

// InitiateUpload 初始化上传
func (s *FileStorage) InitiateUpload(repository, uploadID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 确保仓库目录存在
	repoDir := filepath.Join(s.rootDir, "repositories", repository)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %v", err)
	}

	// 确保上传目录存在
	uploadsDir := filepath.Join(s.rootDir, "uploads", repository)
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create uploads directory: %v", err)
	}

	// 创建上传文件
	uploadFile := filepath.Join(uploadsDir, uploadID)
	file, err := os.Create(uploadFile)
	if err != nil {
		return fmt.Errorf("failed to create upload file: %v", err)
	}
	defer file.Close()

	return nil
}

// AppendToUpload 追加数据到上传
func (s *FileStorage) AppendToUpload(repository, uploadID string, data []byte) (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	uploadFile := filepath.Join(s.rootDir, "uploads", repository, uploadID)
	file, err := os.OpenFile(uploadFile, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to open upload file: %v", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return 0, fmt.Errorf("failed to write to upload file: %v", err)
	}

	// 获取文件大小
	info, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to stat upload file: %v", err)
	}

	return info.Size(), nil
}

// CompleteUpload 完成上传
func (s *FileStorage) CompleteUpload(repository, uploadID, digest string, data []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 确保仓库的blob目录存在
	blobsDir := filepath.Join(s.rootDir, "repositories", repository, "_blobs")
	if err := os.MkdirAll(blobsDir, 0755); err != nil {
		return fmt.Errorf("failed to create blobs directory: %v", err)
	}

	// 处理最后的数据片段
	uploadFile := filepath.Join(s.rootDir, "uploads", repository, uploadID)
	if len(data) > 0 {
		file, err := os.OpenFile(uploadFile, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open upload file: %v", err)
		}
		if _, err := file.Write(data); err != nil {
			file.Close()
			return fmt.Errorf("failed to write to upload file: %v", err)
		}
		file.Close()
	}

	// 移动上传文件到blob文件
	blobFile := filepath.Join(blobsDir, digest)
	if err := os.Rename(uploadFile, blobFile); err != nil {
		// 如果无法重命名（可能跨设备），则复制
		uploadData, err := ioutil.ReadFile(uploadFile)
		if err != nil {
			return fmt.Errorf("failed to read upload file: %v", err)
		}

		if err := ioutil.WriteFile(blobFile, uploadData, 0644); err != nil {
			return fmt.Errorf("failed to write blob file: %v", err)
		}

		// 删除上传文件
		if err := os.Remove(uploadFile); err != nil {
			return fmt.Errorf("failed to remove upload file: %v", err)
		}
	}

	return nil
}
