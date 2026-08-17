package yikdwebclient

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	loginByAppSecretPath      = "Kingdee.BOS.WebApi.ServicesStub.AuthService.LoginByAppSecret.common.kdsvc"
	loginBySignPath           = "Kingdee.BOS.WebApi.ServicesStub.AuthService.LoginBySign.common.kdsvc"
	validateLoginPath         = "Kingdee.BOS.WebApi.ServicesStub.AuthService.ValidateUser.common.kdsvc"
	validateUserEnDeCodePath  = "Kingdee.BOS.WebApi.ServicesStub.AuthService.ValidateUserEnDeCode.common.kdsvc"
	loginBySimplePassportPath = "Kingdee.BOS.WebApi.ServicesStub.AuthService.LoginBySimplePassport.common.kdsvc"
)

// AuthenticationService contains transport settings shared by login services.
type AuthenticationService struct {
	Timeout        time.Duration
	RequestHeaders http.Header
	HTTPClient     *http.Client
}

func (s *AuthenticationService) effectiveTimeout() time.Duration {
	if s.Timeout > 0 {
		return s.Timeout
	}
	return 30 * time.Second
}

func (s *AuthenticationService) login(
	ctx context.Context, serverURL, payload, servicePath string,
) (*RequestWebModel, error) {
	requestURL := NormalizeServerURL(serverURL) + servicePath
	result := &RequestWebModel{RequestUrl: requestURL, RealRequestBody: payload}
	web := NewWebHelperServices()
	web.Timeout = s.effectiveTimeout()
	web.RequestHeaders = cloneHeader(s.RequestHeaders)
	web.HTTPClient = s.HTTPClient
	response, err := web.SendHttpRequestContext(ctx, requestURL, payload)
	result.RealResponseBody = response
	result.Cookie = web.Cookies
	if err != nil {
		return result, err
	}
	return result, nil
}

// LoginByAppSecret implements LoginByAppSecret authentication.
type LoginByAppSecret struct{ AuthenticationService }

// GetLoginJSON builds a LoginByAppSecret request body.
func (s *LoginByAppSecret) GetLoginJSON(settings *AppSettingsModel, unsafeRelaxed bool) (string, error) {
	if settings == nil {
		return "", fmt.Errorf("app settings must not be nil")
	}
	parameters, err := marshalCompatibleJSON([]string{
		settings.XKDApiAcctID, settings.XKDApiUserName, settings.XKDApiAppID,
		settings.XKDApiAppSec, settings.XKDApiLCID,
	}, unsafeRelaxed, false)
	if err != nil {
		return "", fmt.Errorf("marshal LoginByAppSecret parameters: %w", err)
	}
	return GetLoginRequestBodyString(parameters, unsafeRelaxed, true)
}

// GetLoginJson is the C#-style spelling kept for migration.
func (s *LoginByAppSecret) GetLoginJson(settings *AppSettingsModel, unsafeRelaxed bool) (string, error) {
	return s.GetLoginJSON(settings, unsafeRelaxed)
}

func (s *LoginByAppSecret) Login(serverURL, payload string) (*RequestWebModel, error) {
	return s.LoginContext(context.Background(), serverURL, payload)
}

func (s *LoginByAppSecret) LoginContext(ctx context.Context, serverURL, payload string) (*RequestWebModel, error) {
	return s.AuthenticationService.login(ctx, serverURL, payload, loginByAppSecretPath)
}

// LoginBySign implements SHA-256 and legacy SHA-1 login signatures.
type LoginBySign struct {
	AuthenticationService
	LoginType LoginType
}

func (s *LoginBySign) effectiveLoginType() LoginType {
	if s.LoginType == "" {
		return LoginTypeSignSHA256
	}
	return s.LoginType
}

// GetLoginJSON builds a LoginBySign request body.
func (s *LoginBySign) GetLoginJSON(settings *AppSettingsModel, unsafeRelaxed bool) (string, error) {
	if settings == nil {
		return "", fmt.Errorf("app settings must not be nil")
	}
	timestamp := strconv.FormatInt(GetTimestamp(), 10)
	signedValues := []string{
		settings.XKDApiAcctID, settings.XKDApiUserName, settings.XKDApiAppID,
		settings.XKDApiAppSec, timestamp,
	}
	var signature string
	switch s.effectiveLoginType() {
	case LoginTypeSignSHA256:
		signature = GetSHA256(signedValues)
	case LoginTypeSignSHA1:
		signature = GetSHA1(signedValues)
	default:
		return "", fmt.Errorf("unsupported sign login type %q", s.LoginType)
	}
	parameters, err := marshalCompatibleJSON([]string{
		settings.XKDApiAcctID, settings.XKDApiUserName, settings.XKDApiAppID,
		timestamp, signature, settings.XKDApiLCID,
	}, unsafeRelaxed, false)
	if err != nil {
		return "", fmt.Errorf("marshal LoginBySign parameters: %w", err)
	}
	return GetLoginRequestBodyStringByParameters(parameters, unsafeRelaxed, false)
}

func (s *LoginBySign) GetLoginJson(settings *AppSettingsModel, unsafeRelaxed bool) (string, error) {
	return s.GetLoginJSON(settings, unsafeRelaxed)
}

func (s *LoginBySign) Login(serverURL, payload string) (*RequestWebModel, error) {
	return s.LoginContext(context.Background(), serverURL, payload)
}

func (s *LoginBySign) LoginContext(ctx context.Context, serverURL, payload string) (*RequestWebModel, error) {
	return s.AuthenticationService.login(ctx, serverURL, payload, loginBySignPath)
}

// ValidateLogin implements plain username/password validation login.
type ValidateLogin struct{ AuthenticationService }

func (s *ValidateLogin) GetLoginJSON(settings *ValidateLoginSettingsModel, unsafeRelaxed bool) (string, error) {
	if settings == nil {
		return "", fmt.Errorf("validate login settings must not be nil")
	}
	settings.Normalize()
	parameters, err := marshalCompatibleJSON([]any{
		settings.DbId, settings.UserName, settings.Password, settings.Lcid,
	}, unsafeRelaxed, false)
	if err != nil {
		return "", fmt.Errorf("marshal ValidateLogin parameters: %w", err)
	}
	return GetLoginRequestBodyString(parameters, unsafeRelaxed, true)
}

func (s *ValidateLogin) GetLoginJson(settings *ValidateLoginSettingsModel, unsafeRelaxed bool) (string, error) {
	return s.GetLoginJSON(settings, unsafeRelaxed)
}

func (s *ValidateLogin) Login(serverURL, payload string) (*RequestWebModel, error) {
	return s.LoginContext(context.Background(), serverURL, payload)
}

func (s *ValidateLogin) LoginContext(ctx context.Context, serverURL, payload string) (*RequestWebModel, error) {
	return s.AuthenticationService.login(ctx, serverURL, payload, validateLoginPath)
}

// ValidateUserEnDeCode implements the legacy DES-encoded validation login.
type ValidateUserEnDeCode struct{ AuthenticationService }

func (s *ValidateUserEnDeCode) GetLoginJSON(settings *ValidateLoginSettingsModel, unsafeRelaxed bool) (string, error) {
	if settings == nil {
		return "", fmt.Errorf("validate login settings must not be nil")
	}
	settings.Normalize()
	username, err := Encode(settings.UserName)
	if err != nil {
		return "", fmt.Errorf("encode username: %w", err)
	}
	password, err := Encode(settings.Password)
	if err != nil {
		return "", fmt.Errorf("encode password: %w", err)
	}
	parameters, err := marshalCompatibleJSON([]any{settings.DbId, username, password, settings.Lcid}, unsafeRelaxed, false)
	if err != nil {
		return "", fmt.Errorf("marshal ValidateUserEnDeCode parameters: %w", err)
	}
	return GetLoginRequestBodyString(parameters, unsafeRelaxed, true)
}

func (s *ValidateUserEnDeCode) GetLoginJson(settings *ValidateLoginSettingsModel, unsafeRelaxed bool) (string, error) {
	return s.GetLoginJSON(settings, unsafeRelaxed)
}

func (s *ValidateUserEnDeCode) Login(serverURL, payload string) (*RequestWebModel, error) {
	return s.LoginContext(context.Background(), serverURL, payload)
}

func (s *ValidateUserEnDeCode) LoginContext(ctx context.Context, serverURL, payload string) (*RequestWebModel, error) {
	return s.AuthenticationService.login(ctx, serverURL, payload, validateUserEnDeCodePath)
}

// LoginBySimplePassport implements CNF-file and pre-encoded simple-passport login.
type LoginBySimplePassport struct{ AuthenticationService }

func (s *LoginBySimplePassport) GetCnfBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CNF file %q: %w", path, err)
	}
	return data, nil
}

func (s *LoginBySimplePassport) GetPassportForBase64(path string) (string, error) {
	data, err := s.GetCnfBytes(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (s *LoginBySimplePassport) GetLoginJSON(settings *LoginBySimplePassportModel, unsafeRelaxed bool) (string, error) {
	if settings == nil {
		return "", fmt.Errorf("simple passport settings must not be nil")
	}
	settings.Normalize()
	var passport string
	switch settings.BySimplePassportType {
	case SimplePassportCnfFile:
		if strings.TrimSpace(settings.CnfFilePath) == "" {
			return "", fmt.Errorf("CnfFile 类型下 CnfFilePath 必须传值")
		}
		value, err := s.GetPassportForBase64(settings.CnfFilePath)
		if err != nil {
			return "", err
		}
		settings.SimplePassportForBase64 = value
		passport = value
	case SimplePassportForBase64:
		if strings.TrimSpace(settings.SimplePassportForBase64) == "" {
			return "", fmt.Errorf("ForBase64 类型下 SimplePassportForBase64 必须传值")
		}
		passport = settings.SimplePassportForBase64
	default:
		return "", fmt.Errorf("unknown BySimplePassportType %q", settings.BySimplePassportType)
	}
	parameters, err := marshalCompatibleJSON([]any{passport, settings.Lcid}, unsafeRelaxed, false)
	if err != nil {
		return "", fmt.Errorf("marshal simple passport parameters: %w", err)
	}
	return GetLoginRequestBodyString(parameters, unsafeRelaxed, true)
}

func (s *LoginBySimplePassport) GetLoginJson(settings *LoginBySimplePassportModel, unsafeRelaxed bool) (string, error) {
	return s.GetLoginJSON(settings, unsafeRelaxed)
}

func (s *LoginBySimplePassport) Login(serverURL, payload string) (*RequestWebModel, error) {
	return s.LoginContext(context.Background(), serverURL, payload)
}

func (s *LoginBySimplePassport) LoginContext(ctx context.Context, serverURL, payload string) (*RequestWebModel, error) {
	return s.AuthenticationService.login(ctx, serverURL, payload, loginBySimplePassportPath)
}

func newNonce() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate API nonce: %w", err)
	}
	return hex.EncodeToString(data), nil
}

func requoteQuery(raw string) string {
	var result strings.Builder
	for index := 0; index < len(raw); index++ {
		value := raw[index]
		if value == '%' && index+2 < len(raw) {
			if _, err := hex.DecodeString(raw[index+1 : index+3]); err == nil {
				result.WriteString(raw[index : index+3])
				index += 2
				continue
			}
		}
		if value >= 0x80 || value == ' ' || value < 0x20 || value == 0x7f {
			result.WriteString(fmt.Sprintf("%%%02X", value))
			continue
		}
		result.WriteByte(value)
	}
	return result.String()
}

func pathAndQueryForSignature(parsed *url.URL) string {
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	if parsed.ForceQuery || parsed.RawQuery != "" {
		path += "?" + requoteQuery(parsed.RawQuery)
	}
	return path
}

// GetAPIHeaders creates API Sign request headers for the supplied endpoint.
func GetAPIHeaders(settings *AppSettingsModel, requestURL string) (http.Header, error) {
	if settings == nil {
		return nil, fmt.Errorf("app settings must not be nil")
	}
	parsed, err := url.Parse(requestURL)
	if err != nil {
		return nil, fmt.Errorf("parse API URL: %w", err)
	}
	headers := make(http.Header)
	timestamp := strconv.FormatInt(CurrentTimeMillis(), 10)
	nonce, err := newNonce()
	if err != nil {
		return nil, err
	}

	if clientID, encodedSecret, found := strings.Cut(settings.XKDApiAppID, "_"); found {
		secret, decryptErr := DecryptAppSecret(encodedSecret)
		if decryptErr != nil {
			return nil, decryptErr
		}
		if strings.TrimSpace(clientID) != "" {
			headers.Set("X-Api-ClientID", clientID)
			headers.Set("X-Api-Auth-Version", "2.0")
			headers.Set("x-api-signheaders", "x-api-timestamp,x-api-nonce")
			headers.Set("x-api-nonce", nonce)
			headers.Set("x-api-timestamp", timestamp)
			message := "POST\n" + URLEncodeWithUpperCode(pathAndQueryForSignature(parsed)) +
				"\n\nx-api-nonce:" + nonce + "\nx-api-timestamp:" + timestamp + "\n"
			headers.Set("X-Api-Signature", HmacSHA256(message, secret, true))
		}
	}

	appData := strings.Join([]string{
		settings.XKDApiAcctID, settings.XKDApiUserName, settings.XKDApiLCID, settings.XKDApiOrgNum,
	}, ",")
	headers.Set("X-Kd-Appkey", settings.XKDApiAppID)
	headers.Set("X-Kd-Appdata", base64.StdEncoding.EncodeToString([]byte(appData)))
	headers.Set("X-Kd-Signature", HmacSHA256(
		settings.XKDApiAppID+appData, settings.XKDApiAppSec, true,
	))
	return headers, nil
}

// GetApiHeaders is the C#-style spelling kept for migration.
func GetApiHeaders(settings *AppSettingsModel, requestURL string) (http.Header, error) {
	return GetAPIHeaders(settings, requestURL)
}

// GetAPIHeadersString formats headers deterministically as key:value lines.
func GetAPIHeadersString(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var result strings.Builder
	for _, key := range keys {
		for _, value := range headers[key] {
			result.WriteString(key)
			result.WriteByte(':')
			result.WriteString(value)
			result.WriteString("\n")
		}
	}
	return result.String()
}

// GetApiHeadersStr is the C#-style spelling kept for migration.
func GetApiHeadersStr(headers http.Header) string { return GetAPIHeadersString(headers) }
