package storage

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// MemoryStorage 实现基于内存的存储
type MemoryStorage struct {
	repositories map[string]*Repository
	uploads      map[string]map[string][]byte
	mutex        sync.RWMutex
}

// Repository 表示内存中的仓库
type Repository struct {
	Name      string
	Tags      map[string]string // tag -> digest
	Manifests map[string][]byte // digest -> manifest
	Blobs     map[string][]byte // digest -> blob
}

// NewMemoryStorage 创建新的内存存储
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		repositories: make(map[string]*Repository),
		uploads:      make(map[string]map[string][]byte),
	}
}

// ListRepositories 列出所有仓库
func (s *MemoryStorage) ListRepositories() ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	repos := make([]string, 0, len(s.repositories))
	for name := range s.repositories {
		repos = append(repos, name)
	}
	return repos, nil
}

// ListTags 列出仓库的所有标签
func (s *MemoryStorage) ListTags(repository string) ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	repo, ok := s.repositories[repository]
	if !ok {
		return []string{}, nil
	}

	tags := make([]string, 0, len(repo.Tags))
	for tag := range repo.Tags {
		tags = append(tags, tag)
	}
	return tags, nil
}

// GetManifest 获取清单
func (s *MemoryStorage) GetManifest(repository, reference string) ([]byte, string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 首先检查是否是 digest
	if strings.HasPrefix(reference, "sha256:") {
		return s.GetManifestByDigest(repository, reference)
	}

	repo, ok := s.repositories[repository]
	if !ok {
		return nil, "", fmt.Errorf("repository not found: %s", repository)
	}

	digest, ok := repo.Tags[reference]
	if !ok {
		return nil, "", fmt.Errorf("tag not found: %s", reference)
	}

	return s.GetManifestByDigest(repository, digest)
}

// GetManifestByDigest 通过摘要获取清单
func (s *MemoryStorage) GetManifestByDigest(repository, digest string) ([]byte, string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	repo, ok := s.repositories[repository]
	if !ok {
		return nil, "", fmt.Errorf("repository not found: %s", repository)
	}

	manifest, ok := repo.Manifests[digest]
	if !ok {
		return nil, "", fmt.Errorf("manifest not found: %s", digest)
	}

	return manifest, digest, nil
}

// PutManifest 存储清单
func (s *MemoryStorage) PutManifest(repository, reference, digest string, manifest []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 确保仓库存在
	repo, ok := s.repositories[repository]
	if !ok {
		repo = &Repository{
			Name:      repository,
			Tags:      make(map[string]string),
			Manifests: make(map[string][]byte),
			Blobs:     make(map[string][]byte),
		}
		s.repositories[repository] = repo
	}

	// 存储清单
	repo.Manifests[digest] = manifest

	// 如果提供了标签引用，更新标签
	if reference != "" && !strings.HasPrefix(reference, "sha256:") {
		repo.Tags[reference] = digest
	}

	return nil
}

// DeleteManifest 删除清单
func (s *MemoryStorage) DeleteManifest(repository, reference string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	repo, ok := s.repositories[repository]
	if !ok {
		return fmt.Errorf("repository not found: %s", repository)
	}

	// 如果是摘要，直接删除清单
	if strings.HasPrefix(reference, "sha256:") {
		delete(repo.Manifests, reference)
		return nil
	}

	// 如果是标签，找到对应的摘要，然后删除标签和清单
	digest, ok := repo.Tags[reference]
	if !ok {
		return fmt.Errorf("tag not found: %s", reference)
	}

	delete(repo.Tags, reference)
	delete(repo.Manifests, digest)

	return nil
}

// GetBlobSize 获取 blob 大小
func (s *MemoryStorage) GetBlobSize(repository, digest string) (int64, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	repo, ok := s.repositories[repository]
	if !ok {
		return 0, fmt.Errorf("repository not found: %s", repository)
	}

	blob, ok := repo.Blobs[digest]
	if !ok {
		return 0, fmt.Errorf("blob not found: %s", digest)
	}

	return int64(len(blob)), nil
}

// GetBlob 获取 blob
func (s *MemoryStorage) GetBlob(repository, digest string) (io.ReadCloser, int64, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	repo, ok := s.repositories[repository]
	if !ok {
		return nil, 0, fmt.Errorf("repository not found: %s", repository)
	}

	blob, ok := repo.Blobs[digest]
	if !ok {
		return nil, 0, fmt.Errorf("blob not found: %s", digest)
	}

	return io.NopCloser(bytes.NewReader(blob)), int64(len(blob)), nil
}

// DeleteBlob 删除 blob
func (s *MemoryStorage) DeleteBlob(repository, digest string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	repo, ok := s.repositories[repository]
	if !ok {
		return fmt.Errorf("repository not found: %s", repository)
	}

	delete(repo.Blobs, digest)
	return nil
}

// InitiateUpload 初始化上传
func (s *MemoryStorage) InitiateUpload(repository, uploadID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 确保仓库存在
	if _, ok := s.repositories[repository]; !ok {
		s.repositories[repository] = &Repository{
			Name:      repository,
			Tags:      make(map[string]string),
			Manifests: make(map[string][]byte),
			Blobs:     make(map[string][]byte),
		}
	}

	// 确保上传映射存在
	if _, ok := s.uploads[repository]; !ok {
		s.uploads[repository] = make(map[string][]byte)
	}

	// 初始化空上传
	s.uploads[repository][uploadID] = []byte{}
	return nil
}

// AppendToUpload 追加数据到上传
func (s *MemoryStorage) AppendToUpload(repository, uploadID string, data []byte) (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	repoUploads, ok := s.uploads[repository]
	if !ok {
		return 0, fmt.Errorf("no uploads for repository: %s", repository)
	}

	current, ok := repoUploads[uploadID]
	if !ok {
		return 0, fmt.Errorf("upload not found: %s", uploadID)
	}

	// 追加数据
	repoUploads[uploadID] = append(current, data...)
	return int64(len(repoUploads[uploadID])), nil
}

// CompleteUpload 完成上传
func (s *MemoryStorage) CompleteUpload(repository, uploadID, digest string, data []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	repoUploads, ok := s.uploads[repository]
	if !ok {
		return fmt.Errorf("no uploads for repository: %s", repository)
	}

	current, ok := repoUploads[uploadID]
	if !ok {
		return fmt.Errorf("upload not found: %s", uploadID)
	}

	// 处理最后的数据片段
	if len(data) > 0 {
		current = append(current, data...)
	}

	// 确保仓库存在
	repo, ok := s.repositories[repository]
	if !ok {
		repo = &Repository{
			Name:      repository,
			Tags:      make(map[string]string),
			Manifests: make(map[string][]byte),
			Blobs:     make(map[string][]byte),
		}
		s.repositories[repository] = repo
	}

	// 存储 blob
	repo.Blobs[digest] = current

	// 清理上传
	delete(repoUploads, uploadID)
	return nil
}

// generateUploadID 生成上传 ID (辅助函数)
func generateUploadID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
