package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/volume"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type DockerService struct {
	client *client.Client
}

type ContainerInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	State   string `json:"state"`
	Created int64  `json:"created"`
	Ports   []Port `json:"ports"`
}

type Port struct {
	IP          string `json:"ip"`
	PrivatePort uint16 `json:"privatePort"`
	PublicPort  uint16 `json:"publicPort"`
	Type        string `json:"type"`
}

type ImageInfo struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       int64  `json:"size"`
	Created    int64  `json:"created"`
}

type NetworkInfo struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Driver  string       `json:"driver"`
	Scope   string       `json:"scope"`
	IPAM    network.IPAM `json:"ipam"`
	Created time.Time    `json:"created"`
}

type VolumeInfo struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	CreatedAt  string            `json:"created"`
	Labels     map[string]string `json:"labels"`
	Scope      string            `json:"scope"`
	Options    map[string]string `json:"options"`
}

type ContextConfig struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	SSHKeyFile string `json:"sshKeyFile,omitempty"`
}

// ContainerConfig 容器配置
type ContainerConfig struct {
	ImageID       string
	Name          string
	Command       string
	Args          []string
	Ports         []PortMapping
	Env           []EnvVar
	Volumes       []VolumeMapping
	RestartPolicy string
	NetworkMode   string
}

// PortMapping 端口映射
type PortMapping struct {
	Host      uint16
	Container uint16
}

// EnvVar 环境变量
type EnvVar struct {
	Key   string
	Value string
}

// VolumeMapping 数据卷映射
type VolumeMapping struct {
	Host      string
	Container string
	Mode      string
}

func NewDockerService() (*DockerService, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &DockerService{client: cli}, nil
}

func (s *DockerService) ListContainers() ([]ContainerInfo, error) {
	containers, err := s.client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var containerInfos []ContainerInfo
	for _, container := range containers {
		// 处理容器名称，移除开头的 "/"
		name := strings.TrimPrefix(container.Names[0], "/")

		// 转换端口信息
		var ports []Port
		for _, p := range container.Ports {
			ports = append(ports, Port{
				IP:          p.IP,
				PrivatePort: p.PrivatePort,
				PublicPort:  p.PublicPort,
				Type:        p.Type,
			})
		}

		containerInfos = append(containerInfos, ContainerInfo{
			ID:      container.ID[:12], // 只显示ID的前12位
			Name:    name,
			Image:   container.Image,
			Status:  container.Status,
			State:   container.State,
			Created: container.Created,
			Ports:   ports,
		})
	}

	return containerInfos, nil
}

func (s *DockerService) StartContainer(id string) error {
	return s.client.ContainerStart(context.Background(), id, types.ContainerStartOptions{})
}

func (s *DockerService) StopContainer(id string) error {
	return s.client.ContainerStop(context.Background(), id, container.StopOptions{})
}

func (s *DockerService) GetContainerDetail(id string) (types.ContainerJSON, error) {
	return s.client.ContainerInspect(context.Background(), id)
}

func (s *DockerService) ListImages() ([]ImageInfo, error) {
	images, err := s.client.ImageList(context.Background(), types.ImageListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var imageInfos []ImageInfo
	for _, image := range images {
		// 处理 RepoTags，可能为空
		repository := "<none>"
		tag := "<none>"
		if len(image.RepoTags) > 0 {
			parts := strings.Split(image.RepoTags[0], ":")
			if len(parts) == 2 {
				repository = parts[0]
				tag = parts[1]
			}
		}

		imageInfos = append(imageInfos, ImageInfo{
			ID:         image.ID[7:19], // 移除 "sha256:" 前缀并截取
			Repository: repository,
			Tag:        tag,
			Size:       image.Size,
			Created:    image.Created,
		})
	}

	return imageInfos, nil
}

func (s *DockerService) DeleteImage(id string) error {
	_, err := s.client.ImageRemove(context.Background(), id, types.ImageRemoveOptions{Force: false})
	return err
}

func (s *DockerService) CreateContainer(config ContainerConfig) error {
	// 准备端口绑定
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	if len(config.Ports) > 0 {
		for _, p := range config.Ports {
			containerPort := nat.Port(fmt.Sprintf("%d/tcp", p.Container))
			hostBinding := nat.PortBinding{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", p.Host),
			}
			portBindings[containerPort] = []nat.PortBinding{hostBinding}
			exposedPorts[containerPort] = struct{}{}
		}
	}

	// 准备环境变量
	var env []string
	if len(config.Env) > 0 {
		env = make([]string, len(config.Env))
		for i, e := range config.Env {
			env[i] = fmt.Sprintf("%s=%s", e.Key, e.Value)
		}
	}

	// 准备数据卷
	var binds []string
	if len(config.Volumes) > 0 {
		binds = make([]string, len(config.Volumes))
		for i, v := range config.Volumes {
			binds[i] = fmt.Sprintf("%s:%s:%s", v.Host, v.Container, v.Mode)
		}
	}

	// 准备重启策略
	var restartPolicy container.RestartPolicy
	switch config.RestartPolicy {
	case "always":
		restartPolicy = container.RestartPolicy{Name: "always"}
	case "unless-stopped":
		restartPolicy = container.RestartPolicy{Name: "unless-stopped"}
	case "on-failure":
		restartPolicy = container.RestartPolicy{Name: "on-failure"}
	default:
		restartPolicy = container.RestartPolicy{Name: "no"}
	}

	// 准备命令和参数
	var cmd []string
	if config.Command != "" {
		cmd = append(cmd, config.Command)
		if len(config.Args) > 0 {
			cmd = append(cmd, config.Args...)
		}
	}

	// 创建容器配置
	containerConfig := &container.Config{
		Image: config.ImageID,
	}

	// 只有在有命令时才设置
	if len(cmd) > 0 {
		containerConfig.Cmd = cmd
	}

	// 只有在有端口时才设置
	if len(exposedPorts) > 0 {
		containerConfig.ExposedPorts = exposedPorts
	}

	// 只有在有环境变量时才设置
	if len(env) > 0 {
		containerConfig.Env = env
	}

	// 主机配置
	hostConfig := &container.HostConfig{
		RestartPolicy: restartPolicy,
	}

	// 只有在有端口映射时才设置
	if len(portBindings) > 0 {
		hostConfig.PortBindings = portBindings
	}

	// 只有在有数据卷时才设置
	if len(binds) > 0 {
		hostConfig.Binds = binds
	}

	// 只有在指定网络模式时才设置
	if config.NetworkMode != "" {
		hostConfig.NetworkMode = container.NetworkMode(config.NetworkMode)
	}

	// 创建容器
	resp, err := s.client.ContainerCreate(
		context.Background(),
		containerConfig,
		hostConfig,
		nil,         // 网络配置，使用默认值
		nil,         // 平台配置，使用默认值
		config.Name, // 如果名称为空，Docker 会自动生成
	)
	if err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	// 启动容器
	if err := s.client.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	return nil
}

func (s *DockerService) GetImageDetail(id string) (types.ImageInspect, error) {
	inspect, _, err := s.client.ImageInspectWithRaw(context.Background(), id)
	if err != nil {
		return types.ImageInspect{}, err
	}
	return inspect, nil
}

func (s *DockerService) ListNetworks() ([]NetworkInfo, error) {
	networks, err := s.client.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	var networkInfos []NetworkInfo
	for _, network := range networks {
		networkInfos = append(networkInfos, NetworkInfo{
			ID:      network.ID,
			Name:    network.Name,
			Driver:  network.Driver,
			Scope:   network.Scope,
			IPAM:    network.IPAM,
			Created: network.Created,
		})
	}

	return networkInfos, nil
}

func (s *DockerService) GetNetworkDetail(id string) (types.NetworkResource, error) {
	return s.client.NetworkInspect(context.Background(), id, types.NetworkInspectOptions{})
}

func (s *DockerService) DeleteNetwork(id string) error {
	return s.client.NetworkRemove(context.Background(), id)
}

func (s *DockerService) ListVolumes() ([]VolumeInfo, error) {
	volumes, err := s.client.VolumeList(context.Background(), volume.ListOptions{})
	if err != nil {
		return nil, err
	}

	var volumeInfos []VolumeInfo
	for _, volume := range volumes.Volumes {
		volumeInfos = append(volumeInfos, VolumeInfo{
			Name:       volume.Name,
			Driver:     volume.Driver,
			Mountpoint: volume.Mountpoint,
			CreatedAt:  volume.CreatedAt,
			Labels:     volume.Labels,
			Scope:      volume.Scope,
			Options:    volume.Options,
		})
	}

	return volumeInfos, nil
}

func (s *DockerService) GetVolumeDetail(name string) (volume.Volume, error) {
	return s.client.VolumeInspect(context.Background(), name)
}

func (s *DockerService) DeleteVolume(name string) error {
	return s.client.VolumeRemove(context.Background(), name, true)
}

func (s *DockerService) GetContainerLogs(id string) (string, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       "1000", // 获取最后1000行日志
	}

	logs, err := s.client.ContainerLogs(context.Background(), id, options)
	if err != nil {
		return "", err
	}
	defer logs.Close()

	// 读取日志内容
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(logs)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *DockerService) ListContexts() ([]string, error) {
	// 从配置文件读取所有 context
	contexts := []string{"default"}
	configFile := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")

	data, err := os.ReadFile(configFile)
	if err == nil {
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err == nil {
			if contextsMap, ok := config["contexts"].(map[string]interface{}); ok {
				for name := range contextsMap {
					if name != "default" {
						contexts = append(contexts, name)
					}
				}
			}
		}
	}

	return contexts, nil
}

func (s *DockerService) GetCurrentContext() (string, error) {
	// 从环境变量或配置文件获取当前 context
	if context := os.Getenv("DOCKER_CONTEXT"); context != "" {
		return context, nil
	}

	configFile := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")
	data, err := os.ReadFile(configFile)
	if err == nil {
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err == nil {
			if currentContext, ok := config["current-context"].(string); ok {
				return currentContext, nil
			}
		}
	}

	return "default", nil
}

func (s *DockerService) SwitchContext(name string) error {
	// 切换 Docker context
	if name == "default" {
		os.Unsetenv("DOCKER_HOST")
		os.Unsetenv("DOCKER_CONTEXT")
	} else {
		configFile := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")
		data, err := os.ReadFile(configFile)
		if err != nil {
			return err
		}

		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}

		contextsMap, ok := config["contexts"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("context %s not found", name)
		}

		contextConfig, ok := contextsMap[name].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid context configuration for %s", name)
		}

		if endpoint, ok := contextConfig["Endpoints"].(map[string]interface{}); ok {
			if docker, ok := endpoint["docker"].(map[string]interface{}); ok {
				if host, ok := docker["Host"].(string); ok {
					// 验证 host 格式
					if !strings.HasPrefix(host, "tcp://") && !strings.HasPrefix(host, "unix://") {
						return fmt.Errorf("invalid host format, must start with tcp:// or unix://")
					}

					// 如果是 unix socket，验证文件是否存在
					if strings.HasPrefix(host, "unix://") {
						socketPath := strings.TrimPrefix(host, "unix://")
						if _, err := os.Stat(socketPath); err != nil {
							return fmt.Errorf("socket file not found: %s", socketPath)
						}
					}

					os.Setenv("DOCKER_HOST", host)
					os.Setenv("DOCKER_CONTEXT", name)
				}
			}
		}
	}

	// 重新创建 Docker 客户端
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %v", err)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to docker daemon: %v", err)
	}

	s.client = cli
	return nil
}

func (s *DockerService) CreateContext(config ContextConfig) error {
	// 验证 host 格式
	if !strings.HasPrefix(config.Host, "tcp://") && !strings.HasPrefix(config.Host, "unix://") {
		return fmt.Errorf("invalid host format, must start with tcp:// or unix://")
	}

	// 如果是 unix socket，验证文件是否存在
	if strings.HasPrefix(config.Host, "unix://") {
		socketPath := strings.TrimPrefix(config.Host, "unix://")
		if _, err := os.Stat(socketPath); err != nil {
			return fmt.Errorf("socket file not found: %s", socketPath)
		}
	}

	configFile := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")
	var dockerConfig map[string]interface{}

	// 读取现有配置
	data, err := os.ReadFile(configFile)
	if err == nil {
		if err := json.Unmarshal(data, &dockerConfig); err != nil {
			return err
		}
	} else {
		dockerConfig = make(map[string]interface{})
	}

	// 确保 contexts 字段存在
	if _, ok := dockerConfig["contexts"]; !ok {
		dockerConfig["contexts"] = make(map[string]interface{})
	}

	contexts := dockerConfig["contexts"].(map[string]interface{})

	// 添加新的 context
	contextConfig := map[string]interface{}{
		"Endpoints": map[string]interface{}{
			"docker": map[string]interface{}{
				"Host": config.Host,
			},
		},
	}

	contexts[config.Name] = contextConfig

	// 保存配置
	data, err = json.MarshalIndent(dockerConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func (s *DockerService) GetDefaultContextConfig() (string, error) {
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}
	return host, nil
}

func (s *DockerService) UpdateDefaultContext(host string) error {
	// 验证 host 格式
	if !strings.HasPrefix(host, "unix://") && !strings.HasPrefix(host, "tcp://") {
		return fmt.Errorf("invalid host format, must start with unix:// or tcp://")
	}

	// 更新环境变量
	os.Setenv("DOCKER_HOST", host)

	// 重新创建 Docker 客户端
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	s.client = client

	return nil
}

func (s *DockerService) DeleteContext(name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete default context")
	}

	configFile := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")

	// 读取现有配置
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// 获取 contexts 配置
	contexts, ok := config["contexts"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid contexts configuration")
	}

	// 检查 context 是否存在
	if _, exists := contexts[name]; !exists {
		return fmt.Errorf("context %s not found", name)
	}

	// 删除 context
	delete(contexts, name)

	// 如果删除的是当前 context，切换到默认 context
	if currentContext, ok := config["current-context"].(string); ok && currentContext == name {
		config["current-context"] = "default"
		os.Setenv("DOCKER_CONTEXT", "default")
		os.Unsetenv("DOCKER_HOST")
	}

	// 保存配置
	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func (s *DockerService) GetContextConfig(name string) (string, error) {
	if name == "default" {
		return s.GetDefaultContextConfig()
	}

	configFile := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return "", err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}

	contexts, ok := config["contexts"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid contexts configuration")
	}

	contextConfig, ok := contexts[name].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("context %s not found", name)
	}

	endpoints, ok := contextConfig["Endpoints"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid context configuration")
	}

	docker, ok := endpoints["docker"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid docker configuration")
	}

	host, ok := docker["Host"].(string)
	if !ok {
		return "", fmt.Errorf("host not found in context configuration")
	}

	return host, nil
}

func (s *DockerService) UpdateContextConfig(name string, host string) error {
	// 验证 host 格式
	if !strings.HasPrefix(host, "tcp://") && !strings.HasPrefix(host, "unix://") {
		return fmt.Errorf("invalid host format, must start with tcp:// or unix://")
	}

	// 如果是 unix socket，验证文件是否存在
	if strings.HasPrefix(host, "unix://") {
		socketPath := strings.TrimPrefix(host, "unix://")
		if _, err := os.Stat(socketPath); err != nil {
			return fmt.Errorf("socket file not found: %s", socketPath)
		}
	}

	if name == "default" {
		return s.UpdateDefaultContext(host)
	}

	configFile := filepath.Join(os.Getenv("HOME"), ".docker", "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	contexts, ok := config["contexts"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid contexts configuration")
	}

	contextConfig, ok := contexts[name].(map[string]interface{})
	if !ok {
		return fmt.Errorf("context %s not found", name)
	}

	endpoints, ok := contextConfig["Endpoints"].(map[string]interface{})
	if !ok {
		endpoints = make(map[string]interface{})
		contextConfig["Endpoints"] = endpoints
	}

	docker, ok := endpoints["docker"].(map[string]interface{})
	if !ok {
		docker = make(map[string]interface{})
		endpoints["docker"] = docker
	}

	docker["Host"] = host

	// 如果是当前 context，更新环境变量
	if currentContext, ok := config["current-context"].(string); ok && currentContext == name {
		os.Setenv("DOCKER_HOST", host)

		// 重新创建 Docker 客户端
		client, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return err
		}
		s.client = client
	}

	// 保存配置
	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func (s *DockerService) DeleteContainer(id string, force bool) error {
	options := types.ContainerRemoveOptions{
		Force:         force, // 如果容器正在运行，是否强制删除
		RemoveVolumes: false, // 默认不删除关联的匿名卷
	}
	return s.client.ContainerRemove(context.Background(), id, options)
}

// CreateExec 创建执行实例
func (s *DockerService) CreateExec(containerID string, config types.ExecConfig) (types.IDResponse, error) {
	return s.client.ContainerExecCreate(context.Background(), containerID, config)
}

// AttachExec 附加到执行实例
func (s *DockerService) AttachExec(execID string, tty bool) (io.ReadWriteCloser, error) {
	resp, err := s.client.ContainerExecAttach(context.Background(), execID, types.ExecStartCheck{
		Tty:    tty,
		Detach: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to attach exec: %v", err)
	}
	return resp.Conn, nil
}

// StartExec 启动执行实例
func (s *DockerService) StartExec(execID string, config types.ExecStartCheck) error {
	err := s.client.ContainerExecStart(context.Background(), execID, config)
	if err != nil {
		return fmt.Errorf("failed to start exec: %v", err)
	}
	return nil
}

// ResizeExec 调整终端大小
func (s *DockerService) ResizeExec(execID string, height, width int) error {
	return s.client.ContainerExecResize(context.Background(), execID, types.ResizeOptions{
		Height: uint(height),
		Width:  uint(width),
	})
}
