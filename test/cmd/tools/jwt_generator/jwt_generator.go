package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-akka/configuration"
)

var pastTime int64 = time.Now().AddDate(0, 0, -1).Unix()
var currentTime int64 = time.Now().Unix()
var futureTime int64 = time.Now().AddDate(0, 0, 1).Unix()

type configData struct {
	signingMethod string
	claims        *customClaims
}

type customClaims struct {
	jwt.StandardClaims
	Scope string `json:"scope"`
}

func main() {

	var err error

	configFilePath, err := getCmdArgs()
	if err != nil {
		fmt.Printf("Failed to get command arguments: %s\n", err.Error())
		os.Exit(1)
	}

	var signingMethod string

	configData, err := loadDataFromConfigFile(configFilePath)
	if err != nil {
		fmt.Printf("Failed to load data from config file\n")
		os.Exit(1)
	}

	switch {

	// RSA 계열
	case strings.HasPrefix(configData.signingMethod, "RS"):
		generateJWTByRSASeries(strRSASigningMethodToJWTGoType(configData.signingMethod), configData.claims)

	// RSAPSS 계열
	case strings.HasPrefix(configData.signingMethod, "PS"):
		generateJWTByRSAPSSSeries(strRSAPSSSigningMethodToJWTGoType(configData.signingMethod), configData.claims)

	// HMAC 계열
	case strings.HasPrefix(configData.signingMethod, "HS"):
		generateJWTByHMAC(strHMACSigningMethodToJWTGoType(configData.signingMethod), configData.claims)

	default:
		fmt.Printf("Unexpected signing method: %#v\n", signingMethod)
		os.Exit(1)
	}
}

func strRSASigningMethodToJWTGoType(in string) *jwt.SigningMethodRSA {

	var out *jwt.SigningMethodRSA

	switch in {
	case "RS256":
		out = jwt.SigningMethodRS256
	case "RS384":
		out = jwt.SigningMethodRS384
	case "RS512":
		out = jwt.SigningMethodRS512
	default:
		out = nil
	}

	return out
}

func strRSAPSSSigningMethodToJWTGoType(in string) *jwt.SigningMethodRSAPSS {

	var out *jwt.SigningMethodRSAPSS

	switch in {
	case "PS256":
		out = jwt.SigningMethodPS256
	case "PS384":
		out = jwt.SigningMethodPS384
	case "PS512":
		out = jwt.SigningMethodPS512
	default:
		out = nil
	}

	return out
}

func strHMACSigningMethodToJWTGoType(in string) *jwt.SigningMethodHMAC {

	var out *jwt.SigningMethodHMAC

	switch in {
	case "HS256":
		out = jwt.SigningMethodHS256
	case "HS384":
		out = jwt.SigningMethodHS384
	case "HS512":
		out = jwt.SigningMethodHS512
	default:
		out = nil
	}

	return out
}

func getCmdArgs() (configFilePath string, err error) {

	defaultConfigFilePath := fmt.Sprintf("./%s.conf", path.Base(os.Args[0]))
	configFilePathPtr := flag.String("c", defaultConfigFilePath, "Config file path")

	flag.Parse()

	configFilePath = *configFilePathPtr

	return configFilePath, err
}

func loadDataFromConfigFile(configFilePath string) (*configData, error) {

	configData := &configData{
		signingMethod: "",
		claims:        &customClaims{},
	}

	bytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	conf := configuration.ParseString(string(bytes))

	configData.signingMethod = conf.GetString("signingMethod", "")

	configData.claims.Audience = conf.GetString("claims.Audience", "")
	configData.claims.ExpiresAt = convertReservedStrToTime(conf.GetString("claims.ExpiresAt", ""))
	configData.claims.Id = conf.GetString("claims.Id", "")
	configData.claims.IssuedAt = convertReservedStrToTime(conf.GetString("claims.IssuedAt", ""))
	configData.claims.Issuer = conf.GetString("claims.Issuer", "")
	configData.claims.NotBefore = convertReservedStrToTime(conf.GetString("claims.NotBefore", ""))
	configData.claims.Subject = conf.GetString("claims.Subject", "")
	configData.claims.Scope = conf.GetString("claims.Scope", "")

	fmt.Printf("Loaded Config: {signingMethod: %#v, claims: %#v}\n", configData.signingMethod, configData.claims)

	return configData, nil
}

func convertReservedStrToTime(in string) int64 {

	var out int64

	switch in {
	case "pastTime":
		out = time.Now().AddDate(0, 0, -1).Unix()
	case "futureTime":
		out = time.Now().AddDate(0, 0, 1).Unix()
	default:
		out = time.Now().Unix()
	}

	return out
}

func generateJWTByRSASeries(signingMethod *jwt.SigningMethodRSA, claims *customClaims) {

	if signingMethod == nil {
		fmt.Println("Invalid signingMethod")
		return
	}

	var encodedToken *jwt.Token
	encodedToken = jwt.NewWithClaims(signingMethod, *claims)

	signingKey := generateSigningKey()
	fmt.Printf("\nSigning Key:\n%s\n", getRSASigningKeyPEM(signingKey))

	signedJWT, err := encodedToken.SignedString(signingKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Encoded JWT:\n%#v\n", signedJWT)

	fmt.Printf("\nVerification Key:\n%s\n", getRSAVerificaionKeyPEM(signingKey))
}

// TODO https://jwt.io/ 에서 검증 실패함. 수정 필요.
func generateJWTByRSAPSSSeries(signingMethod *jwt.SigningMethodRSAPSS, claims *customClaims) {

	if signingMethod == nil {
		fmt.Println("Invalid signingMethod")
		return
	}

	var encodedToken *jwt.Token
	encodedToken = jwt.NewWithClaims(signingMethod, *claims)

	signingKey := generateSigningKey()
	fmt.Printf("\nSigning Key:\n%s\n", getRSASigningKeyPEM(signingKey))

	signedJWT, err := encodedToken.SignedString(signingKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Encoded JWT:\n%#v\n", signedJWT)

	fmt.Printf("\nVerification Key:\n%s\n", getRSAVerificaionKeyPEM(signingKey))
}

// 서명을 위한 개인키를 생성한다.
func generateSigningKey() *rsa.PrivateKey {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	return privateKey
}

func getRSASigningKeyPEM(signingKey *rsa.PrivateKey) string {

	pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(signingKey),
		})

	return string(pem)
}

func getRSAVerificaionKeyPEM(privateKey *rsa.PrivateKey) string {

	pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
		})

	return string(pem)
}

func generateJWTByHMAC(signingMethod *jwt.SigningMethodHMAC, claims *customClaims) {

	if signingMethod == nil {
		fmt.Println("Invalid signingMethod")
		return
	}

	encodedToken := jwt.NewWithClaims(signingMethod, *claims)

	signingKey := generateHMACSigningKey()
	if signingKey == nil {
		fmt.Println("Failed to generate HMAC signing key")
		return
	}

	signedJWT, err := encodedToken.SignedString(signingKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Encoded JWT:\n%#v\n", signedJWT)

	pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "HMAC PRIVATE KEY",
			Bytes: signingKey,
		})

	fmt.Printf("\nSigning & Verification Key:\n%s\n", string(pem))
}

func generateHMACSigningKey() []byte {
	key := make([]byte, 1024)
	_, err := rand.Read(key)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	return key
}
