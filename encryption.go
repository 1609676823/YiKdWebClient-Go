package yikdwebclient

import (
	"crypto/cipher"
	"crypto/des" // #nosec G505 -- required by Kingdee's legacy login protocol.
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var appSecretPattern = regexp.MustCompile(`^[0-9A-Za-z]{32}$`)

var (
	legacyDESKey = []byte("KingdeeK")
	xorKey       = []byte("0054f397c6234378b09ca7d3e5debce7")
)

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	result := make([]byte, len(data)+padding)
	copy(result, data)
	for index := len(data); index < len(result); index++ {
		result[index] = byte(padding)
	}
	return result
}

// Encode applies the deterministic DES-CBC encoding used by ValidateUserEnDeCode.
func Encode(data any) (string, error) {
	block, err := des.NewCipher(legacyDESKey)
	if err != nil {
		return "", fmt.Errorf("create DES cipher: %w", err)
	}
	plain := pkcs7Pad([]byte(fmt.Sprint(data)), block.BlockSize())
	encrypted := make([]byte, len(plain))
	var mode cipher.BlockMode = cipher.NewCBCEncrypter(block, legacyDESKey)
	mode.CryptBlocks(encrypted, plain)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// EncodeNew1 is a compatibility alias for Encode.
func EncodeNew1(data any) (string, error) { return Encode(data) }

// HmacSHA256 returns standard Base64. When isHex is true, it returns Base64 of
// the lowercase hexadecimal digest, matching the Kingdee header-sign protocol.
func HmacSHA256(message, secret string, isHex ...bool) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	digest := mac.Sum(nil)
	hexMode := false
	if len(isHex) > 0 {
		hexMode = isHex[0]
	}
	if hexMode {
		return base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(digest)))
	}
	return base64.StdEncoding.EncodeToString(digest)
}

// ByteToHexStr returns uppercase hexadecimal text.
func ByteToHexStr(data []byte) string { return strings.ToUpper(hex.EncodeToString(data)) }

func xorEncode(input []byte) ([]byte, error) {
	if len(input) > len(xorKey) {
		return nil, fmt.Errorf("decoded app secret is too long: %d bytes", len(input))
	}
	result := make([]byte, len(input))
	for index, value := range input {
		result[index] = value ^ xorKey[index]
	}
	return result, nil
}

func rot13(value string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'm', r >= 'A' && r <= 'M':
			return r + 13
		case r >= 'n' && r <= 'z', r >= 'N' && r <= 'Z':
			return r - 13
		default:
			return r
		}
	}, value)
}

// EncryptAppSecret applies the reversible encoding used by API Sign v2 IDs.
func EncryptAppSecret(appSecret string) (string, error) {
	if !appSecretPattern.MatchString(appSecret) {
		return rot13(appSecret), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(appSecret)
	if err != nil {
		return "", fmt.Errorf("decode app secret: %w", err)
	}
	encoded, err := xorEncode(decoded)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encoded), nil
}

// DecryptAppSecret reverses the app-secret encoding used by API Sign v2 IDs.
func DecryptAppSecret(appSecret string) (string, error) {
	if len(appSecret) != 32 {
		return rot13(appSecret), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(appSecret)
	if err != nil {
		return "", fmt.Errorf("decode encrypted app secret: %w", err)
	}
	encoded, err := xorEncode(decoded)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encoded), nil
}

// URLEncodeWithUpperCode applies form-style URL encoding with uppercase escapes.
func URLEncodeWithUpperCode(value string) string {
	encoded := []byte(url.QueryEscape(value))
	for index := 0; index+2 < len(encoded); index++ {
		if encoded[index] == '%' {
			encoded[index+1] = byte(strings.ToUpper(string(encoded[index+1]))[0])
			encoded[index+2] = byte(strings.ToUpper(string(encoded[index+2]))[0])
			index += 2
		}
	}
	return string(encoded)
}

// UrlEncodeWithUpperCode is the C#-style spelling kept for migration.
func UrlEncodeWithUpperCode(value string) string { return URLEncodeWithUpperCode(value) }
