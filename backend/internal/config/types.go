package config

// RegistryConfig 表示单个镜像仓库的配置
type Config struct {
	HostName  string `json:"hostName"`
	RemoteURL string `json:"remoteUrl"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
}
