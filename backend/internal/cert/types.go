package cert

// CertificateFiles 包含证书和密钥文件的路径
type CertificateFiles struct {
	CACertFile     string
	CAKeyFile      string
	ServerCertFile string
	ServerKeyFile  string
}
