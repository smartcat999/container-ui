package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/smartcat999/container-ui/internal/server"
)

func main() {
	// 解析命令行参数
	var (
		listenAddr = flag.String("listen", ":5050", "HTTP监听地址")
	)
	flag.Parse()

	// 创建上下文以支持优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动仓库服务器
	registryServer := server.StartRegistryServer(ctx, *listenAddr, nil)

	// 处理信号以优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		registryServer.Shutdown(context.Background())
		cancel()
	}()

	// 等待服务关闭
	<-ctx.Done()
	log.Println("Registry server has shut down")
}
