package config

// RegistryConfig 表示单个镜像仓库的配置
type Config struct {
	HostName  string `json:"hostName"`
	RemoteURL string `json:"remoteUrl"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	DNSNames  []string `json:"dnsNames,omitempty"`
}

func (c *Config) GetDNSNames() []string {
	if c.DNSNames == nil {
		return []string{c.HostName}
	}
	return c.DNSNames
}
