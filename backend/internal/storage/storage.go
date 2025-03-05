package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Storage 定义存储接口
type Storage interface {
	// 仓库操作
	CreateRepository(name string) error
	DeleteRepository(name string) error
	ListRepositories() ([]string, error)

	// 标签操作
	PutTag(repository, tag, digest string) error
	GetTag(repository, tag string) (string, error)
	DeleteTag(repository, tag string) error
	ListTags(repository string) ([]string, error)

	// 层操作
	PutLayer(repository, digest string, reader interface{}) error
	GetLayer(repository, digest string) (io.ReadCloser, error)
	DeleteLayer(repository, digest string) error
	LayerExists(repository, digest string) (bool, error)

	// 清单操作
	PutManifest(repository, tag, digest string, manifest []byte) error
	GetManifest(repository, tag string) ([]byte, string, error)
	GetManifestByDigest(repository, digest string) ([]byte, string, error)
	DeleteManifest(repository, tag string) error
}

// FileStorage 基于文件系统的存储实现
type FileStorage struct {
	rootDir string
	mu      sync.RWMutex
}

// NewFileStorage 创建新的文件存储
func NewFileStorage(rootDir string) (*FileStorage, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, err
	}
	return &FileStorage{
		rootDir: rootDir,
	}, nil
}

// 仓库路径
func (s *FileStorage) repoPath(repository string) string {
	return filepath.Join(s.rootDir, "repositories", repository)
}

// 标签路径
func (s *FileStorage) tagPath(repository, tag string) string {
	return filepath.Join(s.repoPath(repository), "_tags", tag)
}

// 层路径
func (s *FileStorage) layerPath(repository, digest string) string {
	return filepath.Join(s.repoPath(repository), "_layers", digest)
}

// 清单路径
func (s *FileStorage) manifestPath(repository, tag string) string {
	return filepath.Join(s.repoPath(repository), "_manifests", tag)
}

// CreateRepository 创建仓库
func (s *FileStorage) CreateRepository(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repoPath := s.repoPath(name)
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return err
	}

	// 创建必要的子目录
	dirs := []string{
		filepath.Join(repoPath, "_tags"),
		filepath.Join(repoPath, "_layers"),
		filepath.Join(repoPath, "_manifests"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// DeleteRepository 删除仓库
func (s *FileStorage) DeleteRepository(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return os.RemoveAll(s.repoPath(name))
}

// ListRepositories 列出仓库
func (s *FileStorage) ListRepositories() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	reposDir := filepath.Join(s.rootDir, "repositories")
	entries, err := os.ReadDir(reposDir)
	if err != nil {
		return nil, err
	}

	var repos []string
	for _, entry := range entries {
		if entry.IsDir() {
			repos = append(repos, entry.Name())
		}
	}

	return repos, nil
}

// PutTag 添加标签
func (s *FileStorage) PutTag(repository, tag, digest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tagPath := s.tagPath(repository, tag)
	return os.WriteFile(tagPath, []byte(digest), 0644)
}

// GetTag 获取标签
func (s *FileStorage) GetTag(repository, tag string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tagPath := s.tagPath(repository, tag)
	data, err := os.ReadFile(tagPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// DeleteTag 删除标签
func (s *FileStorage) DeleteTag(repository, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return os.Remove(s.tagPath(repository, tag))
}

// ListTags 列出标签
func (s *FileStorage) ListTags(repository string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tagsDir := filepath.Join(s.repoPath(repository), "_tags")
	entries, err := os.ReadDir(tagsDir)
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, entry := range entries {
		if !entry.IsDir() {
			tags = append(tags, entry.Name())
		}
	}

	return tags, nil
}

// RootDir 获取存储根目录
func (s *FileStorage) RootDir() string {
	return s.rootDir
}

// PutLayer 存储层
func (s *FileStorage) PutLayer(repository, digest string, reader interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	layerPath := s.layerPath(repository, digest)
	file, err := os.Create(layerPath)
	if err != nil {
		return err
	}
	defer file.Close()

	switch r := reader.(type) {
	case io.Reader:
		_, err = io.Copy(file, r)
	case string:
		// 如果是文件路径，打开文件并复制
		src, err := os.Open(r)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(file, src)
	default:
		return fmt.Errorf("unsupported reader type: %T", reader)
	}

	return err
}

// GetLayer 获取层
func (s *FileStorage) GetLayer(repository, digest string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	layerPath := s.layerPath(repository, digest)
	return os.Open(layerPath)
}

// DeleteLayer 删除层
func (s *FileStorage) DeleteLayer(repository, digest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return os.Remove(s.layerPath(repository, digest))
}

// LayerExists 检查层是否存在
func (s *FileStorage) LayerExists(repository, digest string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	layerPath := s.layerPath(repository, digest)
	_, err := os.Stat(layerPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// PutManifest 存储清单
func (s *FileStorage) PutManifest(repository, tag, digest string, manifest []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保目录存在
	manifestDir := filepath.Dir(s.manifestPath(repository, tag))
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return err
	}

	// 写入清单文件
	manifestPath := s.manifestPath(repository, tag)
	if err := os.WriteFile(manifestPath, manifest, 0644); err != nil {
		return err
	}

	// 确保标签目录存在
	tagDir := filepath.Dir(s.tagPath(repository, tag))
	if err := os.MkdirAll(tagDir, 0755); err != nil {
		return err
	}

	// 写入标签文件
	tagPath := s.tagPath(repository, tag)
	return os.WriteFile(tagPath, []byte(digest), 0644)
}

// GetManifest 获取清单
func (s *FileStorage) GetManifest(repository, tag string) ([]byte, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	manifestPath := s.manifestPath(repository, tag)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", err
	}

	digest, err := s.GetTag(repository, tag)
	if err != nil {
		return nil, "", err
	}

	return data, digest, nil
}

// GetManifestByDigest 通过摘要获取清单
func (s *FileStorage) GetManifestByDigest(repository, digest string) ([]byte, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 遍历 _manifests 目录下的所有文件
	manifestsDir := filepath.Join(s.repoPath(repository), "_manifests")
	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		return nil, "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(manifestsDir, entry.Name())
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		// 计算文件的摘要
		fileDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
		if fileDigest == digest {
			return data, digest, nil
		}
	}

	return nil, "", fmt.Errorf("manifest not found with digest: %s", digest)
}

// DeleteManifest 删除清单
func (s *FileStorage) DeleteManifest(repository, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(s.manifestPath(repository, tag)); err != nil {
		return err
	}

	return s.DeleteTag(repository, tag)
}

// CalculateDigest 计算内容的摘要
func CalculateDigest(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
