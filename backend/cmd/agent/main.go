package main

import (
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
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
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
	mappingsStr := getEnvOrDefault("REGISTRY_MAPPINGS", "")

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

	// 创建自定义传输层，增加超时设置
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		// 增加超时设置
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, // 连接超时
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second, // 增加TLS握手超时
		ResponseHeaderTimeout: 30 * time.Second, // 响应头超时
		ExpectContinueTimeout: 5 * time.Second,  // 增加100-continue超时
	}

	proxy := httputil.NewSingleHostReverseProxy(remoteURL)
	proxy.Transport = transport

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
		return nil
	}

	return proxy, nil
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
		DNSNames:              []string{hostname, "localhost", "registry-1.docker.io", "docker.io"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
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

func main() {
	// 设置OpenTelemetry导出器
	os.Setenv("OTEL_TRACES_EXPORTER", getEnvOrDefault("OTEL_TRACES_EXPORTER", "console"))

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
		printCerts = flag.Bool("print-certs", false, "打印证书安装指南")
	)
	flag.Parse()

	// 如果请求打印证书安装指南
	if *printCerts {
		fmt.Println("=== Docker 证书安装指南 ===")
		fmt.Println("\n要让Docker信任代理服务器的证书，请执行以下步骤:")
		fmt.Println("1. 将CA证书复制到Docker证书目录:")
		fmt.Println("   sudo mkdir -p /etc/docker/certs.d/registry-1.docker.io")
		fmt.Println("   sudo cp /tmp/registry-proxy-ca.pem /etc/docker/certs.d/registry-1.docker.io/ca.crt")
		fmt.Println("   sudo mkdir -p /etc/docker/certs.d/docker.io")
		fmt.Println("   sudo cp /tmp/registry-proxy-ca.pem /etc/docker/certs.d/docker.io/ca.crt")
		fmt.Println("2. 重启Docker服务:")
		fmt.Println("   sudo systemctl restart docker")
		return
	}

	// 创建配置存储
	store, err := CreateConfigStore(*configType, *configPath)
	if err != nil {
		log.Fatalf("Failed to create config store: %v", err)
	}

	// 创建仓库管理器
	registryManager := NewRegistryManager(store)
	defer registryManager.Close()

	// 创建上下文以支持优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建代理处理器
	proxyHandler := createProxyHandler(registryManager)

	// 启动HTTP代理服务
	proxyServer := startServer(ctx, *listenAddr, proxyHandler, false, "", "", nil)

	// 启动HTTPS代理服务
	var tlsServer *http.Server

	if *certFile != "" && *keyFile != "" {
		// 使用提供的证书和私钥
		tlsServer = startServer(ctx, *listenTLS, proxyHandler, true, *certFile, *keyFile, nil)
		log.Println("使用提供的TLS证书启动HTTPS服务")
	} else if *autoTLS {
		// 定义临时证书文件路径
		tempDir := os.TempDir()
		caCertFile := filepath.Join(tempDir, "registry-proxy-ca.pem")
		caKeyFile := filepath.Join(tempDir, "registry-proxy-ca-key.pem")
		serverCertFile := filepath.Join(tempDir, "registry-proxy-cert.pem")
		serverKeyFile := filepath.Join(tempDir, "registry-proxy-key.pem")

		// 检查证书文件是否已存在
		certFilesExist := fileExists(caCertFile) && fileExists(caKeyFile) &&
			fileExists(serverCertFile) && fileExists(serverKeyFile)

		if certFilesExist {
			// 如果证书文件已存在，直接使用
			log.Println("发现现有的临时TLS证书，将直接使用")
			log.Printf("CA证书位置: %s", caCertFile)
			log.Printf("服务器证书位置: %s", serverCertFile)

			tlsServer = startServer(ctx, *listenTLS, proxyHandler, true, serverCertFile, serverKeyFile, nil)
			log.Println("使用现有临时证书启动HTTPS服务")
			log.Println("要让Docker信任此证书，请运行: ./agent --print-certs")
			log.Printf("或者手动将CA证书 %s 复制到Docker证书目录", caCertFile)
		} else {
			// 如果证书文件不存在，生成新的证书
			log.Println("未找到临时TLS证书，将自动生成新证书")
			caCert, caKey, serverCert, serverKey, err := generateCertificates()
			if err != nil {
				log.Printf("自动生成证书失败: %v", err)
				return
			}
			// 保存证书到文件
			tempDir := os.TempDir()
			caCertFile := filepath.Join(tempDir, "registry-proxy-ca.pem")
			caKeyFile := filepath.Join(tempDir, "registry-proxy-ca-key.pem")
			serverCertFile := filepath.Join(tempDir, "registry-proxy-cert.pem")
			serverKeyFile := filepath.Join(tempDir, "registry-proxy-key.pem")

			if err := os.WriteFile(caCertFile, caCert, 0600); err != nil {
				log.Printf("保存CA证书失败: %v", err)
				return
			}
			if err := os.WriteFile(caKeyFile, caKey, 0600); err != nil {
				log.Printf("保存CA私钥失败: %v", err)
				return
			}
			if err := os.WriteFile(serverCertFile, serverCert, 0600); err != nil {
				log.Printf("保存服务器证书失败: %v", err)
				return
			}
			if err := os.WriteFile(serverKeyFile, serverKey, 0600); err != nil {
				log.Printf("保存服务器私钥失败: %v", err)
				return
			}
			log.Printf("CA证书已保存到: %s", caCertFile)
			log.Printf("服务器证书已保存到: %s", serverCertFile)

			tlsServer = startServer(ctx, *listenTLS, proxyHandler, true, serverCertFile, serverKeyFile, nil)
			log.Println("使用自动生成的证书启动HTTPS服务")
			log.Println("要让Docker信任此证书，请运行: ./agent --print-certs")
			log.Printf("或者手动将CA证书 %s 复制到Docker证书目录", caCertFile)
		}
	}
	// 如果启用了管理API，启动管理服务
	var adminServer *http.Server
	if *adminAPI {
		adminServer = startAdminServer(ctx, *adminAddr, registryManager)
	}

	// 处理信号以优雅关闭
	handleSignals([]*http.Server{proxyServer, tlsServer, adminServer}, cancel)

	// 等待服务关闭
	<-ctx.Done()
	log.Println("所有服务已关闭")
}

// createProxyHandler 创建代理处理器
func createProxyHandler(manager *RegistryManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取请求的主机名
		host := r.Host

		// 如果主机名包含端口，去除端口部分
		if colonIndex := strings.IndexByte(host, ':'); colonIndex != -1 {
			host = host[:colonIndex]
		}

		// 查找对应的配置
		config, ok := manager.GetConfig(host)
		if !ok {
			// 如果没有找到配置，使用默认配置
			config = manager.GetDefaultConfig()
			log.Printf("No mapping found for host: %s, using default: %s", host, config.HostName)
		}

		log.Printf("Proxying request for %s to %s", host, config.RemoteURL)

		// 获取或创建代理处理器
		proxyHandler, err := manager.GetProxyHandler(config)
		if err != nil {
			log.Printf("Error creating proxy for %s: %v", host, err)
			http.Error(w, "Failed to create proxy", http.StatusInternalServerError)
			return
		}

		// 处理请求
		proxyHandler.ServeHTTP(w, r)
	})
}

// startServer 启动HTTP或HTTPS服务器
func startServer(ctx context.Context, listenAddr string, handler http.Handler, useTLS bool, certFile, keyFile string, tlsConfig *tls.Config) *http.Server {
	// 创建HTTP服务器
	server := &http.Server{
		Addr:      listenAddr,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	// 启动服务器
	go func() {
		var err error
		protocol := "HTTP"

		if useTLS {
			protocol = "HTTPS"
			log.Printf("Starting %s registry proxy on %s", protocol, listenAddr)
			err = server.ListenAndServeTLS(certFile, keyFile)
		} else {
			log.Printf("Starting %s registry proxy on %s", protocol, listenAddr)
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Printf("Error starting %s proxy server: %v", protocol, err)
		}
	}()

	// 监听上下文取消
	go func() {
		<-ctx.Done()
		protocol := "HTTP"
		if useTLS {
			protocol = "HTTPS"
		}
		log.Printf("Shutting down %s proxy server...", protocol)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during %s proxy server shutdown: %v", protocol, err)
		}
	}()

	return server
}

// startAdminServer 启动管理API服务
func startAdminServer(ctx context.Context, listenAddr string, manager *RegistryManager) *http.Server {
	mux := http.NewServeMux()

	// 列出所有配置
	mux.HandleFunc("/api/registries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// 列出所有配置
			configs, err := manager.ListConfigs()
			if err != nil {
				http.Error(w, "Failed to list registries: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(configs)
			return
		}

		if r.Method == http.MethodPost {
			// 添加新配置
			var config RegistryConfig
			if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			if config.HostName == "" || config.RemoteURL == "" {
				http.Error(w, "HostName and RemoteURL are required", http.StatusBadRequest)
				return
			}

			if err := manager.AddConfig(config); err != nil {
				http.Error(w, "Failed to add registry: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// 获取、更新或删除特定配置
	mux.HandleFunc("/api/registries/", func(w http.ResponseWriter, r *http.Request) {
		// 提取主机名
		hostName := strings.TrimPrefix(r.URL.Path, "/api/registries/")
		if hostName == "" {
			http.Error(w, "HostName is required", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			// 获取配置
			config, ok := manager.GetConfig(hostName)
			if !ok {
				http.Error(w, "Registry not found", http.StatusNotFound)
				return
			}

			// 不返回敏感信息
			config.Username = ""
			config.Password = ""

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(config)

		case http.MethodPut:
			// 更新配置
			var config RegistryConfig
			if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			// 确保主机名匹配
			config.HostName = hostName

			if config.RemoteURL == "" {
				http.Error(w, "RemoteURL is required", http.StatusBadRequest)
				return
			}

			if err := manager.AddConfig(config); err != nil {
				http.Error(w, "Failed to update registry: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)

		case http.MethodDelete:
			// 删除配置
			removed, err := manager.RemoveConfig(hostName)
			if err != nil {
				http.Error(w, "Failed to remove registry: "+err.Error(), http.StatusInternalServerError)
				return
			}

			if !removed {
				http.Error(w, "Registry not found", http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	// 启动服务器
	go func() {
		log.Printf("Starting admin API on %s", listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting admin server: %v", err)
		}
	}()

	// 监听上下文取消
	go func() {
		<-ctx.Done()
		log.Println("Shutting down admin server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during admin server shutdown: %v", err)
		}
	}()

	return server
}

// handleSignals 处理系统信号以优雅关闭
func handleSignals(servers []*http.Server, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		for _, server := range servers {
			server.Shutdown(context.Background())
		}
		cancel()
	}()
}

// getEnvOrDefault 获取环境变量，如果不存在则返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// fileExists 检查文件是否存在
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	// 确保是文件而不是目录
	return err == nil && !info.IsDir()
}
