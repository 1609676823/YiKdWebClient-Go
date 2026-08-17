package yikdwebclient

import (
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- required by Kingdee's legacy SHA-1 protocol.
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CurrentTimeMillis returns Unix time in milliseconds.
func CurrentTimeMillis() int64 { return time.Now().UnixMilli() }

// GetTimestamp returns Unix time in seconds.
func GetTimestamp() int64 { return time.Now().Unix() }

// GetServerURL normalizes a Kingdee server base URL.
func GetServerURL(value string) string { return NormalizeServerURL(value) }

// GetServerUrl is the C#-style spelling kept for migration.
func GetServerUrl(value string) string { return GetServerURL(value) }

// SHA256Hex returns a lowercase SHA-256 digest.
func SHA256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// Sha256Hex returns a lowercase SHA-256 digest for a UTF-8 string.
func Sha256Hex(value string) string { return SHA256Hex([]byte(value)) }

// ToHexString encodes bytes as hexadecimal text.
func ToHexString(data []byte, lowerCase ...bool) string {
	value := hex.EncodeToString(data)
	toLowerCase := true
	if len(lowerCase) > 0 {
		toLowerCase = lowerCase[0]
	}
	if toLowerCase {
		return value
	}
	return strings.ToUpper(value)
}

// ToBase64 encodes bytes using standard Base64.
func ToBase64(data []byte) string { return base64.StdEncoding.EncodeToString(data) }

func sortedJoin(values []string) string {
	copyOfValues := append([]string(nil), values...)
	sort.Strings(copyOfValues)
	return strings.Join(copyOfValues, "")
}

// GetSignatureSHA1Util is retained for parity with the C# public API.
func GetSignatureSHA1Util(values []string) string { return GetSHA1(values) }

// GetSHA1 sorts, joins, and hashes values with the legacy SHA-1 algorithm.
func GetSHA1(values []string) string {
	digest := sha1.Sum([]byte(sortedJoin(values))) // #nosec G401 -- protocol compatibility.
	return hex.EncodeToString(digest[:])
}

// GetSHA256 sorts, joins, and hashes values with SHA-256. Empty input returns
// an empty string to match the original client.
func GetSHA256(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return Sha256Hex(sortedJoin(values))
}

func newRID() string {
	var value [4]byte
	if _, err := rand.Read(value[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return strconv.FormatInt(int64(int32(binary.LittleEndian.Uint32(value[:]))), 10)
}
