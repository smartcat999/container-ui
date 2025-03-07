package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager 证书管理器
type Manager struct {
	caCert    *x509.Certificate
	caKey     *rsa.PrivateKey
	certFiles CertificateFiles
	certCache sync.Map
}

var (
	manager *Manager
	once    sync.Once
)

// GetManager 获取证书管理器单例
func GetManager() *Manager {
	once.Do(func() {
		manager = &Manager{
			certFiles: CertificateFiles{
				CACertFile:     filepath.Join(os.TempDir(), "registry-proxy-ca.pem"),
				CAKeyFile:      filepath.Join(os.TempDir(), "registry-proxy-ca-key.pem"),
				ServerCertFile: filepath.Join(os.TempDir(), "registry-proxy-cert.pem"),
				ServerKeyFile:  filepath.Join(os.TempDir(), "registry-proxy-key.pem"),
			},
		}
		if err := manager.ensureCA(); err != nil {
			log.Fatal(err)
		}
		log.Printf("Using CA cert: %s", manager.certFiles.CACertFile)
	})
	return manager
}

// GetOrCreateCert 获取或创建证书
func (m *Manager) GetOrCreateCert(hostName string, dnsNames []string) (*tls.Certificate, error) {
	// 检查缓存
	if cert, ok := m.certCache.Load(hostName); ok {
		return cert.(*tls.Certificate), nil
	}

	// 确保CA证书存在
	if err := m.ensureCA(); err != nil {
		return nil, fmt.Errorf("failed to ensure CA: %v", err)
	}

	// 生成新的服务器证书
	cert, err := m.generateServerCert(hostName, dnsNames)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server cert: %v", err)
	}

	// 存入缓存
	m.certCache.Store(hostName, cert)
	return cert, nil
}

// ensureCA 确保CA证书存在
func (m *Manager) ensureCA() error {
	// 检查CA证书文件是否存在
	if _, err := os.Stat(m.certFiles.CACertFile); err == nil {
		// 加载CA证书
		return m.loadCA()
	}

	// 生成新的CA证书
	return m.generateCA()
}

// loadCA 加载CA证书
func (m *Manager) loadCA() error {
	// 读取CA证书
	caCertPEM, err := ioutil.ReadFile(m.certFiles.CACertFile)
	if err != nil {
		return err
	}

	// 读取CA私钥
	caKeyPEM, err := ioutil.ReadFile(m.certFiles.CAKeyFile)
	if err != nil {
		return err
	}

	// 解码CA证书
	block, _ := pem.Decode(caCertPEM)
	if block == nil {
		return fmt.Errorf("failed to decode CA certificate")
	}
	m.caCert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}

	// 解码CA私钥
	block, _ = pem.Decode(caKeyPEM)
	if block == nil {
		return fmt.Errorf("failed to decode CA key")
	}
	m.caKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	return nil
}

// generateCA 生成CA证书
func (m *Manager) generateCA() error {
	// 生成CA私钥
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate CA key: %v", err)
	}

	// 创建CA证书模板
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Registry Proxy CA"},
			CommonName:   "Registry Proxy Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// 创建CA证书
	caBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %v", err)
	}

	// 保存CA证书和私钥
	if err := m.savePEM(m.certFiles.CACertFile, "CERTIFICATE", caBytes); err != nil {
		return err
	}
	if err := m.savePEM(m.certFiles.CAKeyFile, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(caKey)); err != nil {
		return err
	}

	m.caCert = &caTemplate
	m.caKey = caKey
	return nil
}

// generateServerCert 生成服务器证书
func (m *Manager) generateServerCert(hostName string, dnsNames []string) (*tls.Certificate, error) {
	// 生成服务器私钥
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate server key: %v", err)
	}

	// 创建服务器证书模板
	serverTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Registry Proxy Server"},
			CommonName:   hostName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              append([]string{hostName}, dnsNames...),
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	// 使用CA证书签名服务器证书
	serverBytes, err := x509.CreateCertificate(rand.Reader, &serverTemplate, m.caCert, &serverKey.PublicKey, m.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create server certificate: %v", err)
	}

	// 创建证书文件路径
	certFile := filepath.Join(os.TempDir(), fmt.Sprintf("registry-proxy-%s-cert.pem", hostName))
	keyFile := filepath.Join(os.TempDir(), fmt.Sprintf("registry-proxy-%s-key.pem", hostName))

	// 保存服务器证书和私钥
	if err := m.savePEM(certFile, "CERTIFICATE", serverBytes); err != nil {
		return nil, err
	}
	if err := m.savePEM(keyFile, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(serverKey)); err != nil {
		return nil, err
	}

	// 加载证书
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %v", err)
	}

	return &cert, nil
}

// savePEM 保存PEM格式的文件
func (m *Manager) savePEM(filename, blockType string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, &pem.Block{
		Type:  blockType,
		Bytes: data,
	})
}

// GetCACertFile 获取CA证书文件路径
func (m *Manager) GetCACertFile() string {
	return m.certFiles.CACertFile
}
