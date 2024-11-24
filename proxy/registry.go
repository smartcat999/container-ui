package proxy

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
)

type registryTLSConfig struct {
	cipherSuites    []string
	certificatePath string
	privateKeyPath  string
	certificate     *tls.Certificate
}

func SetUpRegistry(tlsCfg *registryTLSConfig, addr string, proxyCfg *configuration.Proxy) (*registry.Registry, error) {
	config := &configuration.Configuration{}
	// TODO: this needs to change to something ephemeral as the test will fail if there is any server
	// already listening on port 5000
	config.HTTP.Addr = addr
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	if tlsCfg != nil {
		config.HTTP.TLS.CipherSuites = tlsCfg.cipherSuites
		config.HTTP.TLS.Certificate = tlsCfg.certificatePath
		config.HTTP.TLS.Key = tlsCfg.privateKeyPath
	}
	config.Proxy = *proxyCfg
	config.Log.Level = "info"
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}
	return registry.NewRegistry(context.Background(), config)
}
