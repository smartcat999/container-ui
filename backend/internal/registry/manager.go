package registry

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/smartcat999/container-ui/internal/config"
	proxytransprt "github.com/smartcat999/container-ui/internal/proxy"
)

// Manager 管理镜像仓库配置
type Manager struct {
	store config.ConfigStore
	// 添加代理处理器缓存，避免重复创建
	proxyHandlers sync.Map
}

// NewManager 创建一个新的仓库管理器
func NewManager(store config.ConfigStore) *Manager {
	rm := &Manager{
		store: store,
	}

	// 加载默认配置
	rm.loadDefaultConfigs()

	return rm
}

// loadDefaultConfigs 加载默认的仓库配置
func (rm *Manager) loadDefaultConfigs() {
	defaultConfigs := []config.Config{
		//{HostName: "localhost", RemoteURL: "https://localhost:7443"},
		//{HostName: "docker.io", RemoteURL: "https://localhost:7443"},
		//{HostName: "registry-1.docker.io", RemoteURL: "https://localhost:7443"},
		//{HostName: "auth.docker.io", RemoteURL: "https://localhost:7443"},
		{HostName: "docker.io", RemoteURL: "https://registry-1.docker.io"},
		{HostName: "registry-1.docker.io", RemoteURL: "https://registry-1.docker.io"},
		{HostName: "auth.docker.io", RemoteURL: "https://auth.docker.io"},
		{HostName: "gcr.io", RemoteURL: "https://gcr.io"},
		{HostName: "k8s.gcr.io", RemoteURL: "https://k8s.gcr.io"},
		{HostName: "quay.io", RemoteURL: "https://quay.io"},
		{HostName: "ghcr.io", RemoteURL: "https://ghcr.io"},
		{HostName: "registry.k8s.io", RemoteURL: "https://registry.k8s.io"},
		{HostName: "mcr.microsoft.com", RemoteURL: "https://mcr.microsoft.com"},
		{HostName: "registry.cn-beijing.aliyuncs.com", RemoteURL: "https://registry.cn-beijing.aliyuncs.com"},
	}

	for _, config := range defaultConfigs {
		if err := rm.AddConfig(config); err != nil {
			log.Printf("Warning: Failed to add default config for %s: %v", config.HostName, err)
		}
	}
}

// GetConfig 获取指定主机名的配置
func (rm *Manager) GetConfig(hostName string) (config.Config, bool) {
	cfg, exists, err := rm.store.Get(hostName)
	if err != nil {
		log.Printf("Error getting config for %s: %v", hostName, err)
		return config.Config{}, false
	}
	return cfg, exists
}

// GetDefaultConfig 获取默认配置
func (rm *Manager) GetDefaultConfig() config.Config {
	// 默认使用docker.io
	cfg, exists, err := rm.store.Get("docker.io")
	if err == nil && exists {
		return cfg
	}

	// 如果没有docker.io配置，获取第一个配置
	configs, err := rm.store.List()
	if err == nil && len(configs) > 0 {
		// 获取完整配置
		cfg, exists, err := rm.store.Get(configs[0].HostName)
		if err == nil && exists {
			return cfg
		}
	}

	// 如果没有任何配置，返回默认的docker.io配置
	return config.Config{
		HostName:  "docker.io",
		RemoteURL: "https://registry-1.docker.io",
	}
}

// AddConfig 添加或更新配置
func (rm *Manager) AddConfig(config config.Config) error {
	if err := rm.store.Add(config); err != nil {
		return err
	}

	// 清除缓存的代理处理器
	rm.proxyHandlers.Delete(config.HostName)

	log.Printf("Registry config added/updated: %s -> %s", config.HostName, config.RemoteURL)
	return nil
}

// RemoveConfig 删除配置
func (rm *Manager) RemoveConfig(hostName string) (bool, error) {
	removed, err := rm.store.Remove(hostName)
	if err != nil {
		return false, err
	}

	if removed {
		// 清除缓存的代理处理器
		rm.proxyHandlers.Delete(hostName)
		log.Printf("Registry config removed: %s", hostName)
	}

	return removed, nil
}

// ListConfigs 列出所有配置
func (rm *Manager) ListConfigs() ([]config.Config, error) {
	return rm.store.List()
}

// Close 关闭管理器
func (rm *Manager) Close() error {
	return rm.store.Close()
}

// GetProxyHandler 获取或创建代理处理器
func (rm *Manager) GetProxyHandler(config config.Config) (http.Handler, error) {
	// 尝试从缓存获取
	if handler, ok := rm.proxyHandlers.Load(config.HostName); ok {
		return handler.(http.Handler), nil
	}

	// 创建新的代理处理器
	handler, err := NewRegistryProxyHandler(config)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	rm.proxyHandlers.Store(config.HostName, handler)
	return handler, nil
}

// NewRegistryProxyHandler 创建新的镜像仓库代理处理器
func NewRegistryProxyHandler(config config.Config) (http.Handler, error) {
	remoteURL, err := url.Parse(config.RemoteURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(remoteURL)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Minute,
			KeepAlive: 30 * time.Minute,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       60 * time.Minute,
		TLSHandshakeTimeout:   5 * time.Minute,
		ResponseHeaderTimeout: 30 * time.Minute,
		ExpectContinueTimeout: 5 * time.Minute,
		MaxIdleConnsPerHost:   20,
		DisableCompression:    false,
	}
	proxy.Transport = proxytransprt.NewRedirectFollowingTransport(transport, 5)

	// 自定义Director函数，添加认证信息
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// 设置Host头
		req.Host = remoteURL.Host

		// 如果配置了认证信息，添加到请求中
		if config.Username != "" && config.Password != "" {
			// 保留客户端原始认证信息
			if _, _, ok := req.BasicAuth(); !ok {
				req.SetBasicAuth(config.Username, config.Password)
			}
		}

		// 添加调试日志
		log.Printf("Proxying request: %s %s -> %s %s %s",
			req.Method, req.URL.Path, remoteURL.String(), req.Header.Get("Content-Type"), req.Header.Get("Content-Length"))
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Registry proxy error: "+err.Error(), http.StatusBadGateway)
	}

	// 自定义ModifyResponse函数，处理响应
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 添加调试日志
		log.Printf("Received response: %d for %s %s %s %s %s", resp.StatusCode, resp.Request.Method, resp.Request.URL.Path,
			resp.Header.Get("Content-Type"), resp.Header.Get("Range"), resp.Header.Get("Content-Length"))

		// 对于大型响应，使用自定义的响应复制器
		if resp.ContentLength > 0 && resp.StatusCode >= http.StatusCreated && http.StatusIMUsed >= resp.StatusCode {
			log.Printf("处理响应: %.2f MB", float64(resp.ContentLength)/(1024*1024))
			// 创建一个新的响应体读取器
			originalBody := resp.Body
			resp.Body = &bufferedReadCloser{
				reader: originalBody,
				closer: originalBody,
				size:   resp.ContentLength,
			}
		}

		// 保持原始的 Content-Length 和 Range 头
		if resp.Header.Get("Content-Length") != "" {
			log.Printf("Original Content-Length: %s", resp.Header.Get("Content-Length"))
		}
		if resp.Header.Get("Range") != "" {
			log.Printf("Original Range: %s", resp.Header.Get("Range"))
		}

		return nil
	}

	// 自定义 FlushInterval 设置
	proxy.FlushInterval = 100 * time.Millisecond

	return proxy, nil
}

// bufferedReadCloser 带缓冲的读取器，用于处理大型响应
type bufferedReadCloser struct {
	reader io.Reader
	closer io.Closer
	size   int64
}

func (b *bufferedReadCloser) Read(p []byte) (n int, err error) {
	// 使用更大的缓冲区
	buf := make([]byte, 32*1024) // 32KB 缓冲区
	n, err = b.reader.Read(buf)
	if err != nil {
		if err == io.EOF {
			log.Printf("读取完成，总大小: %.2f MB", float64(b.size)/(1024*1024))
		} else {
			log.Printf("读取错误: %v", err)
		}
		return 0, err
	}
	// 复制数据到目标缓冲区
	copy(p, buf[:n])
	return n, nil
}

func (b *bufferedReadCloser) Close() error {
	return b.closer.Close()
}
