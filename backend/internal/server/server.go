package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/smartcat999/container-ui/internal/registry"
)

// Server 表示HTTP服务器
type Server struct {
	Server  *http.Server
	Handler http.Handler
}

// ServerOptions 配置服务器选项
type ServerOptions struct {
	Addr    string
	Handler http.Handler
	Manager *registry.Manager
}

// StartServerWithOptions 启动HTTP服务器
func StartServerWithOptions(ctx context.Context, options ServerOptions) *http.Server {
	// 创建基本的多路复用器
	mux := http.NewServeMux()

	// 添加处理器
	if options.Handler != nil {
		mux.Handle("/", options.Handler)
	}

	// 创建服务器
	srv := &http.Server{
		Addr:    options.Addr,
		Handler: mux,
	}

	// 启动服务器
	go func() {
		log.Printf("Starting HTTP server on %s", options.Addr)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// 处理上下文取消
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}
	}()

	return srv
}
