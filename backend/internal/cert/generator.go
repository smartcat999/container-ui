package cert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/smartcat999/container-ui/internal/utils"
)

// GenerateCertificates 生成CA证书和服务器证书
func GenerateCertificates() (caCert, caKey, serverCert, serverKey []byte, err error) {
	// 1. 生成CA私钥
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("生成CA私钥失败: %v", err)
	}

	// 2. 创建CA证书模板
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

	// 3. 创建CA证书
	caBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("创建CA证书失败: %v", err)
	}

	// 4. 生成服务器私钥
	serverPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("生成服务器私钥失败: %v", err)
	}

	// 5. 创建服务器证书模板
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	serverTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Registry Proxy Server"},
			CommonName:   hostname,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{hostname, "localhost", "registry-1.docker.io", "docker.io", "auth.docker.io"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("172.31.19.16")},
	}

	// 6. 使用CA证书签名服务器证书
	serverBytes, err := x509.CreateCertificate(rand.Reader, &serverTemplate, &caTemplate, &serverPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("创建服务器证书失败: %v", err)
	}

	// 7. 编码证书和私钥为PEM格式
	caCertPEM := &bytes.Buffer{}
	if err := pem.Encode(caCertPEM, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes}); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("编码CA证书失败: %v", err)
	}

	caKeyPEM := &bytes.Buffer{}
	if err := pem.Encode(caKeyPEM, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey)}); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("编码CA私钥失败: %v", err)
	}

	serverCertPEM := &bytes.Buffer{}
	if err := pem.Encode(serverCertPEM, &pem.Block{Type: "CERTIFICATE", Bytes: serverBytes}); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("编码服务器证书失败: %v", err)
	}

	serverKeyPEM := &bytes.Buffer{}
	if err := pem.Encode(serverKeyPEM, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey)}); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("编码服务器私钥失败: %v", err)
	}

	return caCertPEM.Bytes(), caKeyPEM.Bytes(), serverCertPEM.Bytes(), serverKeyPEM.Bytes(), nil
}

// SaveCertificates 保存证书到文件
func SaveCertificates(caCert, caKey, serverCert, serverKey []byte, certFiles CertificateFiles) error {
	if err := os.WriteFile(certFiles.CACertFile, caCert, 0600); err != nil {
		return fmt.Errorf("保存CA证书失败: %v", err)
	}
	if err := os.WriteFile(certFiles.CAKeyFile, caKey, 0600); err != nil {
		return fmt.Errorf("保存CA私钥失败: %v", err)
	}
	if err := os.WriteFile(certFiles.ServerCertFile, serverCert, 0600); err != nil {
		return fmt.Errorf("保存服务器证书失败: %v", err)
	}
	if err := os.WriteFile(certFiles.ServerKeyFile, serverKey, 0600); err != nil {
		return fmt.Errorf("保存服务器私钥失败: %v", err)
	}
	return nil
}

// CertificatesExist 检查证书文件是否存在
func CertificatesExist(certFiles CertificateFiles) bool {
	return utils.FileExists(certFiles.CACertFile) &&
		utils.FileExists(certFiles.CAKeyFile) &&
		utils.FileExists(certFiles.ServerCertFile) &&
		utils.FileExists(certFiles.ServerKeyFile)
}
