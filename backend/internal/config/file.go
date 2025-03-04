package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

// FileConfigStore 文件配置存储实现
type FileConfigStore struct {
	*MemoryConfigStore
	filePath string
}

// NewFileConfigStore 创建新的文件配置存储
func NewFileConfigStore(filePath string) (*FileConfigStore, error) {
	store := &FileConfigStore{
		MemoryConfigStore: NewMemoryConfigStore(),
		filePath:          filePath,
	}

	// 如果文件存在，加载配置
	if _, err := os.Stat(filePath); err == nil {
		if err := store.loadFromFile(); err != nil {
			return nil, err
		}
	}

	return store, nil
}

// loadFromFile 从文件加载配置
func (s *FileConfigStore) loadFromFile() error {
	data, err := ioutil.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var configs []Config
	if err := json.Unmarshal(data, &configs); err != nil {
		return err
	}

	for _, config := range configs {
		if err := s.MemoryConfigStore.Add(config); err != nil {
			return err
		}
	}

	return nil
}

// saveToFile 将配置保存到文件
func (s *FileConfigStore) saveToFile() error {
	configs, err := s.MemoryConfigStore.List()
	if err != nil {
		return err
	}

	// 重新获取完整配置（包括敏感信息）
	var fullConfigs []Config
	for _, config := range configs {
		fullConfig, exists, err := s.MemoryConfigStore.Get(config.HostName)
		if err != nil {
			return err
		}
		if exists {
			fullConfigs = append(fullConfigs, fullConfig)
		}
	}

	data, err := json.MarshalIndent(fullConfigs, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(s.filePath, data, 0644)
}

// Add 添加或更新配置并保存到文件
func (s *FileConfigStore) Add(config Config) error {
	if err := s.MemoryConfigStore.Add(config); err != nil {
		return err
	}

	return s.saveToFile()
}

// Remove 删除配置并保存到文件
func (s *FileConfigStore) Remove(hostName string) (bool, error) {
	removed, err := s.MemoryConfigStore.Remove(hostName)
	if err != nil {
		return false, err
	}

	if removed {
		if err := s.saveToFile(); err != nil {
			return true, err
		}
	}

	return removed, nil
}
