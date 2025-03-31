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
		listenAddr = flag.String("listen", ":80", "HTTP监听地址")
		configType = flag.String("config-type", "memory", "配置存储类型 (memory, file)")
		configPath = flag.String("config-path", "", "配置文件路径 (仅用于 file 类型)")
		adminAPI   = flag.Bool("admin-api", true, "启用管理API")
		adminAddr  = flag.String("admin-addr", ":5001", "管理API监听地址")
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
	proxyServer := server.StartServer(ctx, *listenAddr, proxyHandler, registryManager)

	// 如果启用了管理API，启动管理服务
	var adminServer *http.Server
	if *adminAPI {
		adminServer = server.StartAdminServer(ctx, *adminAddr, registryManager)
	}

	// 处理信号以优雅关闭
	handleSignals([]*http.Server{proxyServer, adminServer}, cancel)

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
