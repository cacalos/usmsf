package common

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"camel.uangel.com/ua5g/ulib.git/exec"
	"camel.uangel.com/ua5g/ulib.git/uconf"
	"golang.org/x/crypto/pkcs12"
)

// CertInfo Certification 정보
type CertInfo struct {
	CommonName      string
	PrivateKeyPath  string
	CertificatePath string
	PeerCAsPath     string
	PkcsPath        string
	PkcsKeyPath     string
	Cert            tls.Certificate
	X509Cert        *x509.Certificate
	CAsCertPool     *x509.CertPool
	TouchedAt       time.Time
}

// CertManager Certification 관리자
type CertManager struct {
	mutex           sync.RWMutex
	certMap         map[string]*CertInfo
	defaultCi       *CertInfo
	CommonName      string
	PrivateKeyPath  string
	CertificatePath string
	PkcsPath        string
	PkcsKeyPath     string
	PeerCAsPath     string
	TLSConfig       *tls.Config
	TLSClientConfig *tls.Config
}

var loggers = SamsungLoggers()

////////////////////////////////////////////////////////////////////////////////
// CertInfo
////////////////////////////////////////////////////////////////////////////////

// NewCertInfo 전달된 정보를 가진 새로운 CertInfo 객체를 반환한다.
func NewCertInfo(cname, keyPath, certPath, peerCAsPath string) (*CertInfo, error) {
	ci := &CertInfo{
		CommonName:      cname,
		PrivateKeyPath:  keyPath,
		CertificatePath: certPath,
		PeerCAsPath:     peerCAsPath}

	err := ci.Load()
	if err != nil {
		return nil, err
	}

	return ci, err
}

// NewCertInfoWithPkcs PKCS 파일을 기반으로 Certification 정보를 만든다.
func NewCertInfoWithPkcs(cname, pkcsPath, pkcsKeyPath, peerCAsPath string) (*CertInfo, error) {
	ci := &CertInfo{
		CommonName:  cname,
		PkcsPath:    pkcsPath,
		PkcsKeyPath: pkcsKeyPath,
		PeerCAsPath: peerCAsPath}

	err := ci.Load()
	if err != nil {
		return nil, err
	}

	return ci, err
}

// NewCertInfoByCfg Configuration 파일에 의해 Certification 정보의 생성해 반환한다.
func NewCertInfoByCfg(cfg uconf.Config) (*CertInfo, error) {
	//Forwarder 사용시 내부 네트워크용과 외부 네트워크용 Certification File 목록을 분리해서 관리해야 하지 않을까?
	cname := cfg.GetString("cname", "")
	svrkey := cfg.GetString("key-file", "")
	svrcert := cfg.GetString("cert-file", "")
	peercerts := cfg.GetString("peer-certs-file", "")
	pkcs := cfg.GetString("pkcs-file", "")
	pkcskey := cfg.GetString("pkcs-key-file", "")
	loggers.InfoLogger().Comment("pkcs=%s, pkcskey=%s", pkcs, pkcskey)
	if pkcs != "" && pkcskey != "" {
		return NewCertInfoWithPkcs(cname, pkcs, pkcskey, peercerts)
	}
	return NewCertInfo(cname, svrkey, svrcert, peercerts)
}

// Load Certfication 정보를 로딩한다.
func (ci *CertInfo) Load() error {
	var err error
	loggers.InfoLogger().Comment("Loading Certificates %s, CN=%s\n", ci.CertificatePath, ci.CommonName)

	if ci.PkcsPath != "" && ci.PkcsKeyPath != "" {
		pkcs, err := ioutil.ReadFile(filepath.FromSlash(ci.PkcsPath))
		if err != nil {
			loggers.ErrorLogger().Major("Failed to read file: error=%#v, pkcsfile=%#v, cn=%#v", err.Error(), ci.PkcsPath, ci.CommonName)
			return err
		}
		pkcsKeyByte, err := ioutil.ReadFile(filepath.FromSlash(ci.PkcsKeyPath))
		if err != nil {
			loggers.ErrorLogger().Major("Failed to read file: error=%#v, pkcskeyfile=%#v, cn=%#v", err.Error(), ci.PkcsKeyPath, ci.CommonName)
			return err
		}
		pkcsKey := strings.TrimSuffix(string(pkcsKeyByte), "\n")
		pkcsKey = strings.TrimSuffix(pkcsKey, "\r")
		loggers.InfoLogger().Comment("PKCS KEY=[%s]", pkcsKey)
		blocks, err := pkcs12.ToPEM(pkcs, pkcsKey)
		if err != nil {
			if err != nil {
				loggers.ErrorLogger().Major("Failed to convert to PEM: error=%#v, pkcsfile=%#v, pkcskeyfile=%#v, cn=%#v",
					err.Error(), ci.PkcsPath, ci.PkcsKeyPath, ci.CommonName)
				return err
			}
		}
		var pemData []byte
		for _, b := range blocks {
			pemData = append(pemData, pem.EncodeToMemory(b)...)
		}
		ci.Cert, err = tls.X509KeyPair(pemData, pemData)
		if err != nil {
			loggers.ErrorLogger().Major("Couldn't load certificate files: error=%#v, pkcsfile=%#v, pkcskeyfile=%#v, cn=%#v",
				err.Error(), ci.PkcsPath, ci.PkcsKeyPath, ci.CommonName)
			return err
		}
	} else {
		certpath := filepath.FromSlash(ci.CertificatePath)
		_, err = os.Stat(certpath)
		if err != nil {
			loggers.ErrorLogger().Major("Cert file not found: error=%#v, file=%#v, cn=%#v", err.Error(), certpath, ci.CommonName)
			return err
		}

		privpath := filepath.FromSlash(ci.PrivateKeyPath)
		_, err = os.Stat(privpath)
		if err != nil {
			loggers.ErrorLogger().Major("Private key file not found: error=%#v, file=%#v, cn=%#v", err.Error(), privpath, ci.CommonName)
			return err
		}

		//TODO 추후 로딩 방식을 지정해 DB로 부터 해당 Host 이름의 Key와 Certification을 얻어와 로딩할 수도 있음.
		//func X509KeyPair(certPEMBlock, keyPEMBlock []byte) (Certificate, error)
		ci.Cert, err = tls.LoadX509KeyPair(certpath, privpath)
		if err != nil {
			loggers.ErrorLogger().Major("Failed to load certificate file: error=%#v, certfile=%#v, keyfile=%#v, cn=%#v", err.Error(), certpath, privpath, ci.CommonName)
			return err
		}
	}

	ci.X509Cert, err = x509.ParseCertificate(ci.Cert.Certificate[0])
	if err != nil {
		loggers.ErrorLogger().Major("Failed to parse certificate file: error=%#v, certfile=%#v, keyfile=%#v, pkcsfile=%#v, pkcskeyfile=%#v, cn=%#v",
			err.Error(), ci.CertificatePath, ci.PrivateKeyPath, ci.PkcsPath, ci.PkcsKeyPath, ci.CommonName)
		return err
	}

	//TODO Client CA 인증서 로딩, 현재 기본적으로 하나의 파일을 로딩했으나, 동적으로 로딩되도록 보완 필요.
	//GetConfigForClient 함수를 통해 ServerName + Remote.Addr 정보를 기반으로 ClientCAs와,
	capath := filepath.FromSlash(ci.PeerCAsPath)
	caCert, err := ioutil.ReadFile(capath)
	if err != nil {
		loggers.ErrorLogger().Major("Couldn't load peer CA certificate file: error=%#v, cn=%#v, capath=%#v", err.Error(), ci.CommonName, capath)
		return err
	}
	ci.CAsCertPool = x509.NewCertPool()
	ci.CAsCertPool.AppendCertsFromPEM(caCert)

	ci.TouchedAt = time.Now()

	return nil
}

// GetClientTLSConfig Client용 TLSCertfication 정보를 로딩한다.
func (ci *CertInfo) GetClientTLSConfig() *tls.Config {
	return &tls.Config{
		RootCAs:            ci.CAsCertPool,
		Certificates:       []tls.Certificate{ci.Cert},
		InsecureSkipVerify: true,
	}
}

// GetServerTLSConfig Server용 TLS Configu 정보를 로딩한다.
func (ci *CertInfo) GetServerTLSConfig() *tls.Config {
	return &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    ci.CAsCertPool,
		Certificates: []tls.Certificate{ci.Cert},
	}
}

////////////////////////////////////////////////////////////////////////////////
// CertManger
////////////////////////////////////////////////////////////////////////////////

// NewCertManager 새로운 Certficiation Manager를 반환한다.
func NewCertManager(cname, keyPath, certPath, peerCAsPath string) (*CertManager, error) {
	var err error

	cm := &CertManager{
		CommonName:      cname,
		PrivateKeyPath:  keyPath,
		CertificatePath: certPath,
		PeerCAsPath:     peerCAsPath,
		certMap:         make(map[string]*CertInfo)}

	ci, err := NewCertInfo(cname, keyPath, certPath, peerCAsPath)
	if err != nil {
		return nil, err
	}
	if cname != "" {
		cm.certMap[cname] = ci
	}

	cm.defaultCi = ci
	cm.TLSConfig = &tls.Config{
		ClientAuth:         tls.RequireAndVerifyClientCert,
		ClientCAs:          ci.CAsCertPool,
		GetCertificate:     cm.GetCertificate,
		GetConfigForClient: cm.GetConfigForClient,
	}

	cm.TLSClientConfig = &tls.Config{
		RootCAs:            ci.CAsCertPool,
		Certificates:       []tls.Certificate{ci.Cert},
		InsecureSkipVerify: true,
	}

	return cm, nil
}

//NewCertManagerWithPkcs PCKS 정보를 기반으로 Certificatio Manager를 생성해 반환한다.
func NewCertManagerWithPkcs(
	cname, pkcsPath,
	pkcsKeyPath,
	peerCAsPath string) (*CertManager, error) {
	var err error

	cm := &CertManager{
		CommonName:  cname,
		PkcsPath:    pkcsPath,
		PkcsKeyPath: pkcsKeyPath,
		PeerCAsPath: peerCAsPath,
		certMap:     make(map[string]*CertInfo)}

	ci, err := NewCertInfoWithPkcs(cname, pkcsPath, pkcsKeyPath, peerCAsPath)
	if err != nil {
		return nil, err
	}
	if cname != "" {
		cm.certMap[cname] = ci
	}

	cm.defaultCi = ci
	cm.TLSConfig = &tls.Config{
		ClientAuth:         tls.RequireAndVerifyClientCert,
		ClientCAs:          ci.CAsCertPool,
		GetCertificate:     cm.GetCertificate,
		GetConfigForClient: cm.GetConfigForClient,
	}

	cm.TLSClientConfig = &tls.Config{
		RootCAs:            ci.CAsCertPool,
		Certificates:       []tls.Certificate{ci.Cert},
		InsecureSkipVerify: true,
	}

	return cm, nil
}

// NewCertManagerWithCfg 전달된 Configuration으로 부터 새로운 Certficiation Manager를 반환한다.
func NewCertManagerWithCfg(cfg uconf.Config) (*CertManager, error) {
	cname := cfg.GetString("cname", "")
	svrkey := cfg.GetString("key-file", "")
	svrcert := cfg.GetString("cert-file", "")
	peercerts := cfg.GetString("peer-certs-file", "")
	pkcs := cfg.GetString("pkcs-file", "")
	pkcskey := cfg.GetString("pkcs-key-file", "")
	loggers.InfoLogger().Comment("pkcs=%s, pkcskey=%s", pkcs, pkcskey)
	if pkcs != "" && pkcskey != "" {
		return NewCertManagerWithPkcs(cname, pkcs, pkcskey, peercerts)
	}
	return NewCertManager(cname, svrkey, svrcert, peercerts)
}

// GetCertFromCache Cache로 부터 Certificatio 정보를 획득
func (cm *CertManager) GetCertFromCache(cname string) *CertInfo {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	r, ok := cm.certMap[cname]
	if ok {
		//TODO 만약 해당 Cert에 대한 정보가 Expire 되었다면, 해당 데이터를 Cache에서 삭제하고 nil을 반환해야한다.
		return r
	}
	return nil
}

// GetCertInfo 해당 Server Common Name에 대한 Certification 정보를 반환한다.
func (cm *CertManager) GetCertInfo(cname string) *CertInfo {
	ci := cm.GetCertFromCache(cname)
	if ci == nil {
		//TODO 만약 Cache에 값이 없다면 DB로 부터 정보를 찾아 볼 수도 있을 것이다.

		ci = func() *CertInfo {
			cm.mutex.RLock()
			defer cm.mutex.RUnlock()
			return cm.defaultCi
		}()
	}

	return ci
}

// GetCertificate 전달된 정보의 ClientHello의 Certificate 정보를 반환한다.
func (cm *CertManager) GetCertificate(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	ci := cm.GetCertInfo(chi.ServerName)
	return &ci.Cert, nil
}

// GetConfigForClient Client Hello로 부터 전달된 인증 정보를 반환한다.
func (cm *CertManager) GetConfigForClient(chi *tls.ClientHelloInfo) (*tls.Config, error) {
	ci := cm.GetCertInfo(chi.ServerName)
	r := cm.TLSConfig.Clone()
	r.ClientCAs = ci.CAsCertPool
	return r, nil
}

// VerifyPeerCertificate Peer 인증서를 검증한다. 이는 추후 필요 시 사용될 예정이다.
func (cm *CertManager) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	loggers.InfoLogger().Comment("VerifyPeerCertificate(certs.len=%v, chains=%v)", len(rawCerts), len(verifiedChains))
	if len(verifiedChains) > 0 {
		loggers.InfoLogger().Comment("cert[0].len=%v", len(verifiedChains[0]))
		if len(verifiedChains[0]) > 0 {
			cert := verifiedChains[0][0]
			loggers.InfoLogger().Comment("cert.Issuer=%+v, cert.Subject=%+v", cert.Issuer.CommonName, cert.Subject.CommonName)
		}
	}
	return nil
}

// Reload 전달된 Certification 정보를 다시 로딩 한다.
func (cm *CertManager) Reload() error {
	ci, err := NewCertInfo(cm.CommonName, cm.PrivateKeyPath, cm.CertificatePath, cm.PeerCAsPath)
	if err != nil {
		return err
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	if ci.CommonName != "" {
		cm.certMap[ci.CommonName] = ci
	}
	if ci.CommonName == cm.CommonName {
		cm.defaultCi = ci
	}
	return nil
}

// StartSignalReloadr SIGHUP 시그널을 받았을 때 다시 Certification을 로딩하는 기능을 시작한다.
func (cm *CertManager) StartSignalReloadr() {
	exec.SafeGo(func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)
		for range c {
			loggers.InfoLogger().Comment("Received SIGHUP, reloading TLS certificate information")
			if err := cm.Reload(); err != nil {
				loggers.InfoLogger().Comment("Keeping old TLS certificate because the new one could not be loaded: %v", err)
			}
		}
	})
}
