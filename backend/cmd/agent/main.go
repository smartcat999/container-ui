package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/smartcat999/container-ui/internal/config"
	"github.com/smartcat999/container-ui/internal/registry"
	"github.com/smartcat999/container-ui/internal/server"
	"github.com/smartcat999/container-ui/internal/utils"
)

// RegistryConfig 表示单个镜像仓库的配置
type RegistryConfig struct {
	HostName  string `json:"hostName"`
	RemoteURL string `json:"remoteUrl"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
}

// ConfigStore 定义配置存储接口
type ConfigStore interface {
	// Get 获取特定主机名的配置
	Get(hostName string) (RegistryConfig, bool, error)

	// List 列出所有配置
	List() ([]RegistryConfig, error)

	// Add 添加或更新配置
	Add(config RegistryConfig) error

	// Remove 删除配置
	Remove(hostName string) (bool, error)

	// Close 关闭存储
	Close() error
}

// MemoryConfigStore 内存配置存储实现
type MemoryConfigStore struct {
	configs map[string]RegistryConfig
	mu      sync.RWMutex
}

// NewMemoryConfigStore 创建新的内存配置存储
func NewMemoryConfigStore() *MemoryConfigStore {
	return &MemoryConfigStore{
		configs: make(map[string]RegistryConfig),
	}
}

// Get 获取特定主机名的配置
func (s *MemoryConfigStore) Get(hostName string) (RegistryConfig, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, ok := s.configs[hostName]
	return config, ok, nil
}

// List 列出所有配置
func (s *MemoryConfigStore) List() ([]RegistryConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var configs []RegistryConfig
	for _, config := range s.configs {
		// 创建副本，不包含敏感信息
		safeConfig := RegistryConfig{
			HostName:  config.HostName,
			RemoteURL: config.RemoteURL,
		}
		configs = append(configs, safeConfig)
	}

	return configs, nil
}

// Add 添加或更新配置
func (s *MemoryConfigStore) Add(config RegistryConfig) error {
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

	var configs []RegistryConfig
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
	var fullConfigs []RegistryConfig
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

	return ioutil.WriteFile(s.filePath, data, 0644)
}

// Add 添加或更新配置并保存到文件
func (s *FileConfigStore) Add(config RegistryConfig) error {
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

// RegistryManager 管理镜像仓库配置
type RegistryManager struct {
	store ConfigStore
	// 添加代理处理器缓存，避免重复创建
	proxyHandlers sync.Map
}

// NewRegistryManager 创建一个新的仓库管理器
func NewRegistryManager(store ConfigStore) *RegistryManager {
	rm := &RegistryManager{
		store: store,
	}

	// 加载默认配置
	rm.loadDefaultConfigs()

	// 从环境变量加载配置
	rm.loadFromEnv()

	return rm
}

// loadDefaultConfigs 加载默认的仓库配置
func (rm *RegistryManager) loadDefaultConfigs() {
	defaultConfigs := []RegistryConfig{
		{HostName: "docker.io", RemoteURL: "https://registry-1.docker.io"},
		{HostName: "registry-1.docker.io", RemoteURL: "https://registry-1.docker.io"},
		{HostName: "auth.docker.io", RemoteURL: "https://auth.docker.io"},
		{HostName: "gcr.io", RemoteURL: "https://gcr.io"},
		{HostName: "k8s.gcr.io", RemoteURL: "https://k8s.gcr.io"},
		{HostName: "quay.io", RemoteURL: "https://quay.io"},
		{HostName: "ghcr.io", RemoteURL: "https://ghcr.io"},
		{HostName: "registry.k8s.io", RemoteURL: "https://registry.k8s.io"},
		{HostName: "mcr.microsoft.com", RemoteURL: "https://mcr.microsoft.com"},
	}

	for _, config := range defaultConfigs {
		if err := rm.AddConfig(config); err != nil {
			log.Printf("Warning: Failed to add default config for %s: %v", config.HostName, err)
		}
	}
}

// loadFromEnv 从环境变量加载配置
func (rm *RegistryManager) loadFromEnv() {
	// 从环境变量获取映射配置
	// 格式: REGISTRY_MAPPINGS=host1=url1,host2=url2
	mappingsStr := utils.GetEnvOrDefault("REGISTRY_MAPPINGS", "")

	if mappingsStr != "" {
		pairs := strings.Split(mappingsStr, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				host := strings.TrimSpace(parts[0])
				url := strings.TrimSpace(parts[1])
				if host != "" && url != "" {
					if err := rm.AddConfig(RegistryConfig{
						HostName:  host,
						RemoteURL: url,
					}); err != nil {
						log.Printf("Warning: Failed to add config from env for %s: %v", host, err)
					}
				}
			}
		}
	}
}

// GetConfig 获取指定主机名的配置
func (rm *RegistryManager) GetConfig(hostName string) (RegistryConfig, bool) {
	config, exists, err := rm.store.Get(hostName)
	if err != nil {
		log.Printf("Error getting config for %s: %v", hostName, err)
		return RegistryConfig{}, false
	}
	return config, exists
}

// GetDefaultConfig 获取默认配置
func (rm *RegistryManager) GetDefaultConfig() RegistryConfig {
	// 默认使用docker.io
	config, exists, err := rm.store.Get("docker.io")
	if err == nil && exists {
		return config
	}

	// 如果没有docker.io配置，获取第一个配置
	configs, err := rm.store.List()
	if err == nil && len(configs) > 0 {
		// 获取完整配置
		config, exists, err := rm.store.Get(configs[0].HostName)
		if err == nil && exists {
			return config
		}
	}

	// 如果没有任何配置，返回默认的docker.io配置
	return RegistryConfig{
		HostName:  "docker.io",
		RemoteURL: "https://registry-1.docker.io",
	}
}

// AddConfig 添加或更新配置
func (rm *RegistryManager) AddConfig(config RegistryConfig) error {
	if err := rm.store.Add(config); err != nil {
		return err
	}

	// 清除缓存的代理处理器
	rm.proxyHandlers.Delete(config.HostName)

	log.Printf("Registry config added/updated: %s -> %s", config.HostName, config.RemoteURL)
	return nil
}

// RemoveConfig 删除配置
func (rm *RegistryManager) RemoveConfig(hostName string) (bool, error) {
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
func (rm *RegistryManager) ListConfigs() ([]RegistryConfig, error) {
	return rm.store.List()
}

// Close 关闭管理器
func (rm *RegistryManager) Close() error {
	return rm.store.Close()
}

// GetProxyHandler 获取或创建代理处理器
func (rm *RegistryManager) GetProxyHandler(config RegistryConfig) (http.Handler, error) {
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
func NewRegistryProxyHandler(config RegistryConfig) (http.Handler, error) {
	remoteURL, err := url.Parse(config.RemoteURL)
	if err != nil {
		return nil, err
	}

	// 创建自定义传输层，自动跟随重定向
	baseTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		// 显著增加超时设置
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Minute,  // 5分钟连接超时
			KeepAlive: 30 * time.Minute, // 30分钟保活时间
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       60 * time.Minute, // 1小时空闲连接超时
		TLSHandshakeTimeout:   5 * time.Minute,  // 5分钟TLS握手超时
		ResponseHeaderTimeout: 30 * time.Minute, // 30分钟响应头超时
		ExpectContinueTimeout: 5 * time.Minute,  // 5分钟100-continue超时
		MaxIdleConnsPerHost:   20,               // 增加每个主机的最大空闲连接数
		DisableCompression:    false,            // 启用压缩可以减少传输数据量
	}

	redirectTransport := &redirectFollowingTransport{
		Transport:    baseTransport,
		maxRedirects: 5, // 最多跟随5次重定向
	}

	proxy := httputil.NewSingleHostReverseProxy(remoteURL)
	proxy.Transport = redirectTransport

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
		log.Printf("Proxying request: %s %s -> %s", req.Method, req.URL.Path, remoteURL.String())
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Registry proxy error: "+err.Error(), http.StatusBadGateway)
	}

	// 自定义ModifyResponse函数，处理响应
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 添加调试日志
		log.Printf("Received response: %d for %s %s", resp.StatusCode, resp.Request.Method, resp.Request.URL.Path)

		// 对于大型响应，增加缓冲区大小
		if resp.ContentLength > 10*1024*1024 { // 大于10MB的响应
			log.Printf("处理大型响应: %.2f MB", float64(resp.ContentLength)/(1024*1024))
			// 使用更大的缓冲区读取响应体
			resp.Body = &bufferedReadCloser{
				ReadCloser: resp.Body,
				bufferSize: 1024 * 1024, // 1MB 缓冲区
			}
		}

		return nil
	}

	return proxy, nil
}

// 自定义带缓冲的 ReadCloser
type bufferedReadCloser struct {
	ReadCloser io.ReadCloser
	bufferSize int
	buffer     *bufio.Reader
}

func (b *bufferedReadCloser) Read(p []byte) (int, error) {
	if b.buffer == nil {
		b.buffer = bufio.NewReaderSize(b.ReadCloser, b.bufferSize)
	}
	return b.buffer.Read(p)
}

func (b *bufferedReadCloser) Close() error {
	return b.ReadCloser.Close()
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

// generateCertificates 生成CA证书和服务器证书
func generateCertificates() (caCert, caKey, serverCert, serverKey []byte, err error) {
	// 1. 生成CA私钥
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("生成CA私钥失败: %v", err)
	}

	// 2. 创建CA证书模板
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Registry Proxy CA"},
			CommonName:   "Registry Proxy Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10年有效期
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// 3. 创建CA证书
	caBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("创建CA证书失败: %v", err)
	}

	// 4. 生成服务器私钥
	serverPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("生成服务器私钥失败: %v", err)
	}

	// 5. 创建服务器证书模板
	// 获取主机名
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	serverTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Registry Proxy Server"},
			CommonName:   hostname,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // 1年有效期
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{hostname, "localhost", "registry-1.docker.io", "docker.io", "auth.docker.io"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("172.31.19.16")},
	}

	// 6. 使用CA证书签名服务器证书
	serverBytes, err := x509.CreateCertificate(rand.Reader, &serverTemplate, &caTemplate, &serverPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("创建服务器证书失败: %v", err)
	}

	// 7. 编码证书和私钥为PEM格式
	caCertPEM := &bytes.Buffer{}
	if err := pem.Encode(caCertPEM, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes}); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("编码CA证书失败: %v", err)
	}

	caKeyPEM := &bytes.Buffer{}
	if err := pem.Encode(caKeyPEM, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey)}); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("编码CA私钥失败: %v", err)
	}

	serverCertPEM := &bytes.Buffer{}
	if err := pem.Encode(serverCertPEM, &pem.Block{Type: "CERTIFICATE", Bytes: serverBytes}); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("编码服务器证书失败: %v", err)
	}

	serverKeyPEM := &bytes.Buffer{}
	if err := pem.Encode(serverKeyPEM, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey)}); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("编码服务器私钥失败: %v", err)
	}

	return caCertPEM.Bytes(), caKeyPEM.Bytes(), serverCertPEM.Bytes(), serverKeyPEM.Bytes(), nil
}

// 创建自定义的传输层，自动跟随重定向
type redirectFollowingTransport struct {
	*http.Transport
	maxRedirects int
}

func (t *redirectFollowingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 保存原始请求的副本
	origReq := req.Clone(req.Context())

	var resp *http.Response
	var err error

	// 跟随重定向，最多maxRedirects次
	for redirects := 0; redirects < t.maxRedirects; redirects++ {
		resp, err = t.Transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		// 如果不是重定向，直接返回
		if resp.StatusCode != http.StatusTemporaryRedirect &&
			resp.StatusCode != http.StatusMovedPermanently &&
			resp.StatusCode != http.StatusFound &&
			resp.StatusCode != http.StatusSeeOther {
			return resp, nil
		}

		// 获取重定向URL
		location, err := resp.Location()
		if err != nil {
			return resp, nil // 无法获取重定向URL，返回原始响应
		}

		log.Printf("跟随重定向: %s -> %s", req.URL.String(), location.String())

		// 关闭当前响应体
		resp.Body.Close()

		// 创建新的请求
		newReq, err := http.NewRequestWithContext(req.Context(), origReq.Method, location.String(), nil)
		if err != nil {
			return nil, err
		}

		// 复制原始请求的头部
		for key, values := range origReq.Header {
			for _, value := range values {
				newReq.Header.Add(key, value)
			}
		}

		// 更新请求
		req = newReq
	}

	// 如果达到最大重定向次数，返回最后一个响应
	return resp, err
}

func main() {
	// 设置OpenTelemetry导出器
	os.Setenv("OTEL_TRACES_EXPORTER", utils.GetEnvOrDefault("OTEL_TRACES_EXPORTER", "console"))

	// 解析命令行参数
	var (
		listenAddr = flag.String("listen", ":80", "HTTP监听地址")
		listenTLS  = flag.String("listen-tls", ":443", "HTTPS监听地址")
		certFile   = flag.String("cert-file", "", "TLS证书文件路径")
		keyFile    = flag.String("key-file", "", "TLS私钥文件路径")
		configType = flag.String("config-type", "memory", "配置存储类型 (memory, file)")
		configPath = flag.String("config-path", "", "配置文件路径 (仅用于 file 类型)")
		adminAPI   = flag.Bool("admin-api", false, "启用管理API")
		adminAddr  = flag.String("admin-addr", ":5001", "管理API监听地址")
		autoTLS    = flag.Bool("auto-tls", true, "自动生成TLS证书（当未提供证书时）")
	)
	flag.Parse()

	// 创建配置存储
	store, err := config.CreateConfigStore(*configType, *configPath)
	if err != nil {
		log.Fatalf("Failed to create config store: %v", err)
	}

	// 创建仓库管理器
	registryManager := registry.NewManager(store)
	defer registryManager.Close()

	// 创建上下文以支持优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建代理处理器
	proxyHandler := server.CreateProxyHandler(registryManager)

	// 启动HTTP代理服务
	proxyServer := server.StartServer(ctx, *listenAddr, proxyHandler, false, "", "", nil)

	// 启动HTTPS代理服务
	var tlsServer *http.Server

	if *certFile != "" && *keyFile != "" {
		// 使用提供的证书和私钥
		tlsServer = server.StartServer(ctx, *listenTLS, proxyHandler, true, *certFile, *keyFile, registryManager)
		log.Println("使用提供的TLS证书启动HTTPS服务")
	} else if *autoTLS {
		tlsServer = server.StartServer(ctx, *listenTLS, proxyHandler, true, "", "", registryManager)
		log.Println("使用自动生成的TLS证书启动HTTPS服务")
	}

	// 如果启用了管理API，启动管理服务
	var adminServer *http.Server
	if *adminAPI {
		adminServer = server.StartAdminServer(ctx, *adminAddr, registryManager)
	}

	// 处理信号以优雅关闭
	handleSignals([]*http.Server{proxyServer, tlsServer, adminServer}, cancel)

	// 等待服务关闭
	<-ctx.Done()
	log.Println("所有服务已关闭")
}

// handleSignals 处理系统信号以优雅关闭
func handleSignals(servers []*http.Server, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		for _, server := range servers {
			if server != nil {
				server.Shutdown(context.Background())
			}
		}
		cancel()
	}()
}
