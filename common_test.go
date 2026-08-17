package yikdwebclient

import (
	"crypto/cipher"
	"crypto/des"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testSettings(serverURL string) *AppSettingsModel {
	return &AppSettingsModel{
		XKDApiAcctID: "account-id", XKDApiAppID: "app-id", XKDApiAppSec: "app-secret",
		XKDApiUserName: "api-user", XKDApiLCID: "2052",
		XKDApiServerUrl: NormalizeServerURL(serverURL), XKDApiOrgNum: "100",
	}
}

func envelopeParameters(t *testing.T, envelope string) []any {
	t.Helper()
	var root struct {
		Parameters string `json:"parameters"`
	}
	if err := json.Unmarshal([]byte(envelope), &root); err != nil {
		t.Fatalf("decode envelope: %v\n%s", err, envelope)
	}
	var parameters []any
	if err := json.Unmarshal([]byte(root.Parameters), &parameters); err != nil {
		t.Fatalf("decode parameters: %v\n%s", err, root.Parameters)
	}
	return parameters
}

func TestCommonFunctions(t *testing.T) {
	before := time.Now().UnixMilli()
	actual := CurrentTimeMillis()
	after := time.Now().UnixMilli()
	if actual < before || actual > after {
		t.Fatalf("CurrentTimeMillis = %d, expected [%d,%d]", actual, before, after)
	}
	if got := NormalizeServerURL("https://example.test/k3cloud"); got != "https://example.test/k3cloud/" {
		t.Fatalf("NormalizeServerURL = %q", got)
	}
	if got := Sha256Hex("abc"); got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("Sha256Hex = %q", got)
	}
	if got := GetSHA1([]string{"c", "a", "b"}); got != "a9993e364706816aba3e25717850c26c9cd0d89d" {
		t.Fatalf("GetSHA1 = %q", got)
	}
	if got := GetSHA256([]string{"c", "b", "a"}); got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("GetSHA256 = %q", got)
	}
	if GetSHA256(nil) != "" || ToHexString([]byte{0, 10, 171, 255}, false) != "000AABFF" {
		t.Fatal("hash/hex compatibility failed")
	}
	if ToBase64([]byte{0, 1, 2, 254, 255}) != "AAEC/v8=" {
		t.Fatal("ToBase64 compatibility failed")
	}
}

func TestJSONEnvelopes(t *testing.T) {
	value, err := GetRequestBodyString("TEST_Form", `{"Id":1}`, true, "Forbid")
	if err != nil {
		t.Fatal(err)
	}
	parameters := envelopeParameters(t, value)
	want := []any{"TEST_Form", "Forbid", `{"Id":1}`}
	if !reflect.DeepEqual(parameters, want) {
		t.Fatalf("parameters = %#v, want %#v", parameters, want)
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(value), &root); err != nil {
		t.Fatal(err)
	}
	if root["format"] != float64(1) || root["useragent"] != "ApiClient" || root["v"] != "1.0" {
		t.Fatalf("unexpected envelope: %#v", root)
	}
	relaxed, _ := GetRequestBodyString("FORM", "<tag>", true, "")
	safe, _ := GetRequestBodyString("FORM", "<tag>", false, "")
	if !strings.Contains(relaxed, "<tag>") || strings.Contains(safe, "<tag>") || !strings.Contains(safe, `\\u003Ctag\\u003E`) {
		t.Fatalf("escaping mismatch\nrelaxed=%s\nsafe=%s", relaxed, safe)
	}
}

func TestLegacyEncryptionAndHMAC(t *testing.T) {
	encrypted, err := Encode("hello")
	if err != nil {
		t.Fatal(err)
	}
	again, _ := EncodeNew1("hello")
	if encrypted != again {
		t.Fatalf("DES output is not deterministic: %q != %q", encrypted, again)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := des.NewCipher([]byte("KingdeeK"))
	plain := make([]byte, len(ciphertext))
	var mode cipher.BlockMode = cipher.NewCBCDecrypter(block, []byte("KingdeeK"))
	mode.CryptBlocks(plain, ciphertext)
	plain = plain[:len(plain)-int(plain[len(plain)-1])]
	if string(plain) != "hello" {
		t.Fatalf("decrypted = %q", plain)
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte("message"))
	if got, want := HmacSHA256("message", "secret", false), base64.StdEncoding.EncodeToString(mac.Sum(nil)); got != want {
		t.Fatalf("HmacSHA256 = %q, want %q", got, want)
	}
	if ByteToHexStr([]byte{0, 10, 171, 255}) != "000AABFF" {
		t.Fatal("ByteToHexStr mismatch")
	}
	if UrlEncodeWithUpperCode("a b+c/?") != "a+b%2Bc%2F%3F" {
		t.Fatalf("URL encoding = %q", UrlEncodeWithUpperCode("a b+c/?"))
	}
}

func TestConfigurationAndModels(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "settings.xml")
	content := `<?xml version="1.0"?><configuration><appSettings>
<add key="X-KDApi-AcctID" value="account-from-test" />
<add key="X-KDApi-AppID" value="app-from-test" />
<add key="X-KDApi-AppSec" value="secret-from-test" />
<add key="X-KDApi-UserName" value="user-from-test" />
<add key="X-KDApi-LCID" value="2052" />
<add key="X-KDApi-ServerUrl" value="https://example.test/k3cloud" />
<add key="X-KDApi-OrgNum" value="100" />
</appSettings></configuration>`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := LoadAppSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if settings.XKDApiAcctID != "account-from-test" || settings.XKDApiServerUrl != "https://example.test/k3cloud/" {
		t.Fatalf("settings = %#v", settings)
	}
	stub := CustomServicesStubpath{" Sample .WebApi ", " Service ", " Run "}
	if got := stub.GetCustomServicesStubpathURL(); got != "Sample.WebApi.Service.Run,Sample.WebApi.common.kdsvc" {
		t.Fatalf("custom stub = %q", got)
	}
}

func TestAuthenticationPayloadsAndHeaders(t *testing.T) {
	settings := testSettings("https://example.test/k3cloud/")
	appSecret := &LoginByAppSecret{}
	payload, err := appSecret.GetLoginJSON(settings, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := envelopeParameters(t, payload); !reflect.DeepEqual(got, []any{"account-id", "api-user", "app-id", "app-secret", "2052"}) {
		t.Fatalf("app-secret parameters = %#v", got)
	}

	sign := &LoginBySign{LoginType: LoginTypeSignSHA256}
	payload, err = sign.GetLoginJSON(settings, true)
	if err != nil {
		t.Fatal(err)
	}
	parameters := envelopeParameters(t, payload)
	timestamp := parameters[3].(string)
	wantSignature := GetSHA256([]string{"account-id", "api-user", "app-id", "app-secret", timestamp})
	if parameters[4] != wantSignature {
		t.Fatalf("signature = %q, want %q", parameters[4], wantSignature)
	}

	headers, err := GetAPIHeaders(settings, "https://example.test/k3cloud/service?x=1")
	if err != nil {
		t.Fatal(err)
	}
	if headers.Get("X-Kd-Appkey") != "app-id" {
		t.Fatalf("app key = %q", headers.Get("X-Kd-Appkey"))
	}
	appDataBytes, _ := base64.StdEncoding.DecodeString(headers.Get("X-Kd-Appdata"))
	if string(appDataBytes) != "account-id,api-user,2052,100" {
		t.Fatalf("app data = %q", appDataBytes)
	}

	settings.XKDApiAppID = "client_frperg"
	v2, err := GetAPIHeaders(settings, "https://example.test/k3cloud/service?a=hello world")
	if err != nil {
		t.Fatal(err)
	}
	if v2.Get("X-Api-ClientID") != "client" || len(v2.Get("X-Api-Nonce")) != 32 {
		t.Fatalf("v2 headers = %#v", v2)
	}
	message := "POST\n" + UrlEncodeWithUpperCode("/k3cloud/service?a=hello%20world") +
		"\n\nx-api-nonce:" + v2.Get("X-Api-Nonce") +
		"\nx-api-timestamp:" + v2.Get("X-Api-Timestamp") + "\n"
	if got, want := v2.Get("X-Api-Signature"), HmacSHA256(message, "secret", true); got != want {
		t.Fatalf("v2 signature = %q, want %q", got, want)
	}
}
