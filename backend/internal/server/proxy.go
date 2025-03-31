package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/smartcat999/container-ui/internal/config"
	"github.com/smartcat999/container-ui/internal/registry"
	"github.com/smartcat999/container-ui/internal/storage"
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

// StartServer 启动代理服务器 (兼容旧版API)
func StartServer(ctx context.Context, addr string, handler http.Handler, manager *registry.Manager) *http.Server {
	return StartServerWithOptions(ctx, ServerOptions{
		Addr:    addr,
		Handler: handler,
		Manager: manager,
	})
}

// StartRegistryServer 启动仓库服务器 (兼容旧版API)
func StartRegistryServer(ctx context.Context, addr string, manager *registry.Manager) *http.Server {
	log.Printf("正在初始化仓库服务器，监听地址: %s", addr)

	// 创建存储
	storage, err := storage.NewFileStorage("./tmp")
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	log.Printf("存储初始化成功: %v", storage)

	// 创建注册表处理器
	registryHandler := registry.NewHandler(storage)
	log.Printf("处理器初始化成功: %v", registryHandler)

	// 创建路由器
	router := registry.NewRouter(registryHandler)
	log.Printf("路由器初始化成功: %v", router)

	// 记录服务启动信息
	log.Printf("Registry server is running at %s", addr)

	return StartServerWithOptions(ctx, ServerOptions{
		Addr:    addr,
		Handler: router,
		Manager: manager,
	})
}

// StartAdminServer 启动管理API服务器
func StartAdminServer(ctx context.Context, listenAddr string, manager *registry.Manager) *http.Server {
	// 创建管理API路由
	mux := http.NewServeMux()

	// 健康检查API
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	// 获取所有仓库配置
	mux.HandleFunc("/api/v1/registries", func(w http.ResponseWriter, r *http.Request) {
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
			var cfg config.Config
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if err := manager.AddConfig(cfg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusCreated)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 特定仓库配置
	mux.HandleFunc("/api/v1/registries/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			http.Error(w, "Invalid registry ID", http.StatusBadRequest)
			return
		}

		hostName := parts[3]

		switch r.Method {
		case http.MethodGet:
			config, exists := manager.GetConfig(hostName)
			if !exists {
				http.Error(w, "Registry not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(config)
		case http.MethodPut:
			var cfg config.Config
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			cfg.HostName = hostName
			if err := manager.AddConfig(cfg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
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

	return StartServerWithOptions(ctx, ServerOptions{
		Addr:    listenAddr,
		Handler: mux,
		Manager: manager,
	})
}
