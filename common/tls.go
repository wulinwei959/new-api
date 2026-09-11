package common

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// GetTLSConfig 根据环境变量构建 *tls.Config，返回 nil 表示不启用 TLS
func GetTLSConfig() (*tls.Config, error) {
	tlsEnabled := GetEnvOrDefaultBool("TLS_ENABLED", false)
	if !tlsEnabled {
		return nil, nil
	}

	autoCert := GetEnvOrDefaultBool("TLS_AUTO_CERT", false)
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	var certData, keyData []byte
	var err error

	if autoCert {
		// 自动签发模式：生成或读取自签证书
		dataDir := GetEnvOrDefaultString("DATA_DIR", "")
		tlsDir := filepath.Join(dataDir, "tls")
		certPath := filepath.Join(tlsDir, "cert.pem")
		keyPath := filepath.Join(tlsDir, "key.pem")

		// 确保目录存在
		if err := os.MkdirAll(tlsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create TLS directory: %v", err)
		}

		// 检查证书文件是否存在且未过期
		certExists := fileExists(certPath)
		keyExists := fileExists(keyPath)
		if certExists && keyExists {
			certData, err = os.ReadFile(certPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read cert file: %v", err)
			}
			keyData, err = os.ReadFile(keyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read key file: %v", err)
			}
			// 检查证书是否已过期（提前 30 天续期）
			if block, _ := pem.Decode(certData); block != nil {
				if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
					if time.Now().Add(30*24*time.Hour).After(cert.NotAfter) {
						// 证书即将过期，重新生成
						certData, keyData, err = GenerateSelfSignedCert(certPath, keyPath)
						if err != nil {
							return nil, err
						}
					}
				}
			}
		} else {
			// 生成新证书
			certData, keyData, err = GenerateSelfSignedCert(certPath, keyPath)
			if err != nil {
				return nil, err
			}
		}
	} else if certFile != "" && keyFile != "" {
		// 手动上传模式
		certData, err = os.ReadFile(certFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate file %s: %v", certFile, err)
		}
		keyData, err = os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file %s: %v", keyFile, err)
		}
	} else {
		return nil, fmt.Errorf("TLS_ENABLED=true but no certificate configuration found (set TLS_AUTO_CERT=true or TLS_CERT_FILE + TLS_KEY_FILE)")
	}

	// 加载证书
	cert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TLS certificate: %v", err)
	}

	// 构建 TLS 配置
	minVersion := os.Getenv("TLS_MIN_VERSION")
	var minTLSVersion uint16
	switch minVersion {
	case "1.3":
		minTLSVersion = tls.VersionTLS13
	default:
		minTLSVersion = tls.VersionTLS12
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   minTLSVersion,
	}, nil
}

// GenerateSelfSignedCert 生成 RSA 2048 自签证书
func GenerateSelfSignedCert(certFile, keyFile string) ([]byte, []byte, error) {
	// 生成私钥
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key: %v", err)
	}

	// 构建证书模板
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:   "new-api",
			Organization: []string{"new-api"},
		},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // 1年有效期
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// 自签证书
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	// 编码证书
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// 编码私钥
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	// 写入文件
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return nil, nil, fmt.Errorf("failed to write key file: %v", err)
	}

	return certPEM, keyPEM, nil
}

// StartHTTPRedirectServer 启动 HTTP→HTTPS 301 重定向 server
func StartHTTPRedirectServer(port int, httpsHost string) {
	httpSrv := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := fmt.Sprintf("https://%s%s", httpsHost, r.URL.RequestURI())
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		}),
	}
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			SysError(fmt.Sprintf("HTTP redirect server failed: %v", err))
		}
	}()
	SysLog(fmt.Sprintf("HTTP redirect server started on port %d", port))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
