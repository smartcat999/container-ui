package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/smartcat999/container-ui/internal/config"
	"github.com/smartcat999/container-ui/internal/registry"
	"github.com/smartcat999/container-ui/internal/server"
	"github.com/smartcat999/container-ui/internal/utils"
)

func main() {
	// 设置OpenTelemetry导出器
	os.Setenv("OTEL_TRACES_EXPORTER", utils.GetEnvOrDefault("OTEL_TRACES_EXPORTER", "console"))

	// 解析命令行参数
	var (
		listenAddr  = flag.String("listen", ":80", "HTTP监听地址")
		listenTLS   = flag.String("listen-tls", ":443", "HTTPS监听地址")
		registryTLS = flag.String("registry-tls", ":7443", "HTTPS监听地址")
		certFile    = flag.String("cert-file", "", "TLS证书文件路径")
		keyFile     = flag.String("key-file", "", "TLS私钥文件路径")
		configType  = flag.String("config-type", "memory", "配置存储类型 (memory, file)")
		configPath  = flag.String("config-path", "", "配置文件路径 (仅用于 file 类型)")
		adminAPI    = flag.Bool("admin-api", true, "启用管理API")
		adminAddr   = flag.String("admin-addr", ":5001", "管理API监听地址")
		autoTLS     = flag.Bool("auto-tls", true, "自动生成TLS证书（当未提供证书时）")
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
	proxyServer := server.StartServer(ctx, *listenAddr, proxyHandler, false, "", "", registryManager)

	// 启动HTTPS代理服务
	var tlsServer *http.Server
	var registryTlsServer *http.Server

	if *certFile != "" && *keyFile != "" {
		// 使用提供的证书和私钥
		tlsServer = server.StartServer(ctx, *listenTLS, proxyHandler, true, *certFile, *keyFile, registryManager)
		registryTlsServer = server.StartRegistryServer(ctx, *registryTLS, true, *certFile, *keyFile, registryManager)
		log.Println("使用提供的TLS证书启动HTTPS服务")
	} else if *autoTLS {
		tlsServer = server.StartServer(ctx, *listenTLS, proxyHandler, true, "", "", registryManager)
		registryTlsServer = server.StartRegistryServer(ctx, *registryTLS, true, "", "", registryManager)
		log.Println("使用自动生成的TLS证书启动HTTPS服务")
	}

	// 如果启用了管理API，启动管理服务
	var adminServer *http.Server
	if *adminAPI {
		adminServer = server.StartAdminServer(ctx, *adminAddr, registryManager)
	}

	// 处理信号以优雅关闭
	handleSignals([]*http.Server{proxyServer, tlsServer, registryTlsServer, adminServer}, cancel)

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
