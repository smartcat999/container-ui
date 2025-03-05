package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/smartcat999/container-ui/internal/config"

	"github.com/smartcat999/container-ui/internal/cert"

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

// StartServer 启动代理服务器
func StartServer(ctx context.Context, addr string, handler http.Handler, useTLS bool, certFile, keyFile string, manager *registry.Manager) *http.Server {
	// 创建服务器
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// 如果使用TLS，配置TLS
	if useTLS {
		// 获取证书管理器
		certManager := cert.GetManager()

		// 创建TLS配置
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			ClientAuth: tls.NoClientCert,
			GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
				// 如果提供了证书文件，使用提供的证书
				if certFile != "" && keyFile != "" {
					cert, err := tls.LoadX509KeyPair(certFile, keyFile)
					if err != nil {
						return nil, fmt.Errorf("failed to load certificate: %v", err)
					}
					return &tls.Config{
						Certificates: []tls.Certificate{cert},
					}, nil
				}

				// 如果没有提供证书文件，使用证书管理器
				if manager != nil {
					// 查找匹配的配置
					config, exists := manager.GetConfig(info.ServerName)
					if !exists {
						return nil, fmt.Errorf("no config found for host: %s", info.ServerName)
					}
					// 获取或创建证书
					cert, err := certManager.GetOrCreateCert(info.ServerName, config.GetDNSNames())
					if err != nil {
						return nil, fmt.Errorf("failed to get or create certificate: %v", err)
					}

					return &tls.Config{
						Certificates: []tls.Certificate{*cert},
					}, nil
				}

				// 如果没有配置，使用默认证书
				cert, err := certManager.GetOrCreateCert(info.ServerName, []string{info.ServerName})
				if err != nil {
					return nil, fmt.Errorf("failed to get or create default certificate: %v", err)
				}

				return &tls.Config{
					Certificates: []tls.Certificate{*cert},
				}, nil
			},
		}
		srv.TLSConfig = tlsConfig
	}

	// 启动服务器
	go func() {
		var err error
		if useTLS {
			err = srv.ListenAndServeTLS("", "")
		} else {
			err = srv.ListenAndServe()
		}
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
