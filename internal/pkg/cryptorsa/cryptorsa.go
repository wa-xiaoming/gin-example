package cryptorsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

// GenerateKey 生成一个 2048 位的 RSA 密钥对
func GenerateKey() {
	// 生成一个 2048 位的 RSA 密钥对
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	// 获取私钥的字节表示
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)

	// 将私钥字节放入 PEM 块中
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	// 将 PEM 块编码为字符串
	privateKeyStr := string(pem.EncodeToMemory(privateKeyPEM))

	// 获取公钥的字节表示
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		panic(err)
	}

	// 将公钥字节放入 PEM 块中
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	// 将 PEM 块编码为字符串
	publicKeyStr := string(pem.EncodeToMemory(publicKeyPEM))

	fmt.Println(fmt.Sprintf("RSA 公钥：\n%s", publicKeyStr))
	fmt.Println(fmt.Sprintf("RSA 私钥：\n%s", privateKeyStr))
	fmt.Println("你可以将这些字符串保存到文件中或传递给其他程序使用，记得妥善保管私钥，避免泄露！")

}

// PublicKeyEncrypt 公钥加密
func PublicKeyEncrypt(publicKey, plaintext string) (string, error) {
	// pem 解码
	block, _ := pem.Decode([]byte(publicKey))

	// x509 解码
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	//对明文进行加密
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, publicKeyInterface.(*rsa.PublicKey), []byte(plaintext))
	if err != nil {
		return "", err
	}

	//返回密文
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// PrivateKeyDecrypt 私钥解密
func PrivateKeyDecrypt(privateKey, ciphertext string) (string, error) {
	// pem 解码
	block, _ := pem.Decode([]byte(privateKey))

	// X509 解码
	rsaPrivateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	ciphertextBytes, err := base64.URLEncoding.DecodeString(ciphertext)

	//对密文进行解密
	plaintext, _ := rsa.DecryptPKCS1v15(rand.Reader, rsaPrivateKey, ciphertextBytes)

	//返回明文
	return string(plaintext), nil
}
