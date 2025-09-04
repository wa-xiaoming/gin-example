package cryptoaes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
)

// Encrypt 加密算法
func Encrypt(key, plaintext string) (string, error) {
	keyByte := []byte(key)
	plaintextByte := []byte(plaintext)

	// 创建一个 AES 块
	block, err := aes.NewCipher(keyByte)
	if err != nil {
		return "", err
	}

	// 对明文进行填充
	padding := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	padText := append(plaintextByte, bytes.Repeat([]byte{byte(padding)}, padding)...)

	ciphertext := make([]byte, aes.BlockSize+len(padText))

	// 生成随机的 IV
	iv := ciphertext[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	// 解密数据
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], padText)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密算法
func Decrypt(key, ciphertext string) (string, error) {
	ciphertextByte, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	// 创建一个 AES 块
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		panic(err)
	}

	// 提取 IV
	iv := ciphertextByte[:aes.BlockSize]

	// 提取密文
	ciphertextByteWithoutIV := ciphertextByte[aes.BlockSize:]

	// 创建一个 CBC 模式的 AES 解密器
	mode := cipher.NewCBCDecrypter(block, iv)

	// 解密数据
	decrypted := make([]byte, len(ciphertextByteWithoutIV))
	mode.CryptBlocks(decrypted, ciphertextByteWithoutIV)

	// 去除填充字节
	padding := int(decrypted[len(decrypted)-1])
	decrypted = decrypted[:len(decrypted)-padding]

	return string(decrypted), nil
}
