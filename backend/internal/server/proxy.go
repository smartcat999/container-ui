package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/smartcat999/container-ui/internal/config"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/smartcat999/container-ui/internal/registry"
)

// CreateProxyHandler 创建代理处理器
func CreateProxyHandler(manager *registry.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if colonIndex := strings.IndexByte(host, ':'); colonIndex != -1 {
			host = host[:colonIndex]
		}

		config, ok := manager.GetConfig(host)
		if !ok {
			config = manager.GetDefaultConfig()
			log.Printf("No mapping found for host: %s, using default: %s", host, config.HostName)
		}

		log.Printf("Proxying request for %s to %s", host, config.RemoteURL)

		proxyHandler, err := manager.GetProxyHandler(config)
		if err != nil {
			log.Printf("Error creating proxy for %s: %v", host, err)
			http.Error(w, "Failed to create proxy", http.StatusInternalServerError)
			return
		}

		proxyHandler.ServeHTTP(w, r)
	})
}

// StartServer 启动HTTP或HTTPS服务器
func StartServer(ctx context.Context, listenAddr string, handler http.Handler, useTLS bool, certFile, keyFile string, tlsConfig *tls.Config) *http.Server {
	server := &http.Server{
		Addr:      listenAddr,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

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

// StartAdminServer 启动管理API服务器
func StartAdminServer(ctx context.Context, listenAddr string, manager *registry.Manager) *http.Server {
	// 创建管理API路由
	mux := http.NewServeMux()

	// 获取所有仓库配置
	mux.HandleFunc("/api/registries", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			configs, err := manager.ListConfigs()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(configs)
		case http.MethodPost:
			var config config.Config
			if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := manager.AddConfig(config); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 获取/删除特定仓库配置
	mux.HandleFunc("/api/registries/", func(w http.ResponseWriter, r *http.Request) {
		hostName := strings.TrimPrefix(r.URL.Path, "/api/registries/")
		if hostName == "" {
			http.Error(w, "Host name is required", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			config, exists := manager.GetConfig(hostName)
			if !exists {
				http.Error(w, "Registry not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(config)
		case http.MethodDelete:
			removed, err := manager.RemoveConfig(hostName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
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

	// 创建管理服务器
	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	// 启动服务器
	go func() {
		log.Printf("Starting admin API server on %s", listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Error starting admin API server: %v", err)
		}
	}()

	// 处理优雅关闭
	go func() {
		<-ctx.Done()
		log.Println("Shutting down admin API server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during admin API server shutdown: %v", err)
		}
	}()

	return server
}
