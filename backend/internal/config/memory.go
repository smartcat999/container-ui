package config

import (
	"errors"
	"sync"
)

// ConfigStore 定义配置存储接口
type ConfigStore interface {
	// Get 获取特定主机名的配置
	Get(hostName string) (Config, bool, error)

	// List 列出所有配置
	List() ([]Config, error)

	// Add 添加或更新配置
	Add(config Config) error

	// Remove 删除配置
	Remove(hostName string) (bool, error)

	// Close 关闭存储
	Close() error
}

// MemoryConfigStore 内存配置存储实现
type MemoryConfigStore struct {
	configs map[string]Config
	mu      sync.RWMutex
}

// NewMemoryConfigStore 创建新的内存配置存储
func NewMemoryConfigStore() *MemoryConfigStore {
	return &MemoryConfigStore{
		configs: make(map[string]Config),
	}
}

// Get 获取特定主机名的配置
func (s *MemoryConfigStore) Get(hostName string) (Config, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, ok := s.configs[hostName]
	return config, ok, nil
}

// List 列出所有配置
func (s *MemoryConfigStore) List() ([]Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var configs []Config
	for _, config := range s.configs {
		// 创建副本，不包含敏感信息
		safeConfig := Config{
			HostName:  config.HostName,
			RemoteURL: config.RemoteURL,
		}
		configs = append(configs, safeConfig)
	}

	return configs, nil
}

// Add 添加或更新配置
func (s *MemoryConfigStore) Add(config Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.configs[config.HostName] = config
	return nil
}

// Remove 删除配置
func (s *MemoryConfigStore) Remove(hostName string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.configs[hostName]; exists {
		delete(s.configs, hostName)
		return true, nil
	}

	return false, nil
}

// Close 关闭存储
func (s *MemoryConfigStore) Close() error {
	return nil
}

// CreateConfigStore 创建配置存储
func CreateConfigStore(configType, configPath string) (ConfigStore, error) {
	switch configType {
	case "memory":
		return NewMemoryConfigStore(), nil
	case "file":
		if configPath == "" {
			return nil, errors.New("file path is required for file config store")
		}
		return NewFileConfigStore(configPath)
	default:
		return nil, errors.New("unsupported config store type")
	}
}
