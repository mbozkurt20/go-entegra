package migros

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
)

// AESEncrypt Rijndael AES-256-CBC (zero IV) + PKCS7 ile şifreler (Migros API formatı).
// .NET RijndaelManaged varsayılan olarak CBC + zero IV kullanır.
// Dönen değer: base64(ciphertext)
func AESEncrypt(secretKey string, plaintext []byte) (string, error) {
	key := normalizeKey(secretKey)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("AES cipher hatası: %w", err)
	}

	padded := pkcs7Pad(plaintext, aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	iv := make([]byte, aes.BlockSize) // zero IV (.NET Rijndael varsayılanı)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESDecrypt Rijndael AES-256-CBC (zero IV) + PKCS7 ile şifreli veriyi çözer.
func AESDecrypt(secretKey, encryptedBase64 string) ([]byte, error) {
	key := normalizeKey(secretKey)

	cipherData, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		cipherData, err = base64.URLEncoding.DecodeString(encryptedBase64)
		if err != nil {
			return nil, fmt.Errorf("base64 decode hatası: %w", err)
		}
	}

	if len(cipherData) == 0 || len(cipherData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("şifreli veri geçersiz uzunlukta")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("AES cipher hatası: %w", err)
	}

	plaintext := make([]byte, len(cipherData))
	iv := make([]byte, aes.BlockSize) // zero IV
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, cipherData)

	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return nil, fmt.Errorf("PKCS7 unpad hatası: %w", err)
	}

	return plaintext, nil
}

// normalizeKey secretKey'i 32 byte'a normalize eder
func normalizeKey(key string) []byte {
	raw := []byte(key)
	normalized := make([]byte, 32)
	copy(normalized, raw)
	return normalized
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	pad := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, pad...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("veri boş")
	}
	padding := int(data[len(data)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, fmt.Errorf("geçersiz padding değeri: %d", padding)
	}
	return data[:len(data)-padding], nil
}
