package main

import (
	"github.com/smartcat999/registry-agent/proxy"
	"log"
	"os"

	"github.com/distribution/distribution/v3/configuration"
)

func main() {
	// otlp/console/none
	os.Setenv("OTEL_TRACES_EXPORTER", "console")
	proxyCfg := &configuration.Proxy{
		RemoteURL: getEnvOrDefault("REGISTRY_PROXY_REMOTE_URL", "https://registry-1.docker.io"),
		Username:  getEnvOrDefault("REGISTRY_PROXY_USERNAME", ""),
		Password:  getEnvOrDefault("REGISTRY_PROXY_PASSWORD", ""),
		Exec:      nil,
		TTL:       nil,
	}
	registry, err := proxy.SetUpRegistry(nil, "127.0.0.1:5000", proxyCfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting registry proxy on :5000")
	if err := registry.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
