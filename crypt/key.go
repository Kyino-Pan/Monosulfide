package crypt

import (
	"blockEmulator/config"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	"os"
)

func InitPrivateKey() *rsa.PrivateKey {
	if hasPrivateKey() {
		return GenerateKeys()
		// todo
		// all nodes will have same pubKey, need to be fixed
		// return loadPrivateKey()
	} else {
		return GenerateKeys()
	}
}

func GenerateKeys() *rsa.PrivateKey {
	// 生成密钥对
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("生成RSA密钥对失败: %v", err)
	}

	// 将私钥转换为PEM格式
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// 将公钥转换为PEM格式
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Fatalf("无法从私钥中提取公钥: %v", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	// 打印密钥
	//log.Printf("私钥: \n%s\n", privateKeyPEM)
	//log.Printf("公钥: \n%s\n", publicKeyPEM)
	err = os.WriteFile(config.PrivateKeyPEM, privateKeyPEM, 0600)
	if err != nil {
		log.Fatalf("保存私钥失败: %v", err)
	}
	err = os.WriteFile(config.PublicKeyPEM, publicKeyPEM, 0644)
	if err != nil {
		log.Fatalf("保存公钥失败: %v", err)
	}
	return privateKey
}

func fileExist(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}
func hasPrivateKey() bool {
	return fileExist(config.PrivateKeyPEM)
}

func loadPEMFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func loadPrivateKey() *rsa.PrivateKey {
	pkPEM, err := loadPEMFile(config.PrivateKeyPEM)
	if err != nil {
		log.Fatalf("读取私钥文件失败: %v", err)
	}

	// 解码PEM数据
	block, _ := pem.Decode(pkPEM)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		log.Fatal("无法解码私钥PEM数据")
	}

	// 解析PKCS1格式的私钥
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("解析私钥失败: %v", err)
	}

	// 现在 privateKey 是 *rsa.PrivateKey 类型
	log.Printf("成功加载私钥: %+v\n", privateKey)
	return privateKey
}

/*
privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
publicKey := &privateKey.PublicKey
message := []byte("Hello, world!")
signature, err := SignMessage(privateKey, message)
// 验证签名
isValid := VerifySignature(publicKey, message, signature)
*/

// SignMessage 使用私钥对消息进行签名
func SignMessage(privateKey *rsa.PrivateKey, message []byte) ([]byte, error) {
	// 计算消息的SHA-256哈希
	hashed := sha256.Sum256(message)

	// 使用私钥和SHA-256哈希算法对消息的哈希进行签名
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// VerifySignature 验证消息的签名是否有效
func VerifySignature(publicKey *rsa.PublicKey, message []byte, signature []byte) bool {
	// 计算消息的SHA-256哈希
	hashed := sha256.Sum256(message)

	// 使用公钥和SHA-256哈希算法验证签名
	err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature)
	return err == nil
}

func EncodePublicKey(pubKey *rsa.PublicKey) []byte {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		log.Fatalf("无法从私钥中提取公钥: %v", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	return publicKeyPEM
}

// DecodePublicKey 从PEM编码的公钥中解析出*rsa.PublicKey
func DecodePublicKey(pemBytes []byte) *rsa.PublicKey {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		log.Fatalf("公钥PEM解码失败")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		log.Fatalf("公钥解析失败: %v", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		log.Fatalf("非RSA公钥: %v", err)
	}
	return rsaPub
}

func PubKey2Str(pubKey *rsa.PublicKey) string {
	return string(EncodePublicKey(pubKey))
}

// get the digest of request
func GetDigest(r any) []byte {
	b, err := json.Marshal(r)
	if err != nil {
		log.Panic(err)
	}
	hash := sha256.Sum256(b)
	return hash[:]
}
