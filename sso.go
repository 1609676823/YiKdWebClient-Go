package yikdwebclient

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SSOHelper builds single-sign-on URLs and logout payloads for protocol V1-V4.
type SSOHelper struct {
	Timestamp                 int64
	ArgJSON                   string
	ArgJSONBase64             string
	PermitCount               string
	UnsafeRelaxedJsonEscaping bool
	SimplePassportLoginArg    SimplePassportLoginArg
	SSOLoginURLObject         SSOLoginURLObject
	AppSettingsModel          *AppSettingsModel
	URL                       string
	SSOLogoutObject           SSOLogoutObject
	HTTPClient                *http.Client
	Timeout                   time.Duration
}

// NewSSOHelper creates an SSO helper. Nil settings use the optional default XML file.
func NewSSOHelper(settings *AppSettingsModel) *SSOHelper {
	if settings == nil {
		settings, _ = LoadAppSettingsIfExists("")
	}
	if settings == nil {
		settings = &AppSettingsModel{}
	}
	settings.Normalize()
	args := NewSimplePassportLoginArg()
	args.DbID = settings.XKDApiAcctID
	args.AppID = settings.XKDApiAppID
	args.Username = settings.XKDApiUserName
	args.Lcid = settings.XKDApiLCID
	return &SSOHelper{
		UnsafeRelaxedJsonEscaping: true,
		SimplePassportLoginArg:    args,
		AppSettingsModel:          settings,
		URL:                       settings.XKDApiServerUrl,
		Timeout:                   30 * time.Second,
	}
}

func (h *SSOHelper) prepare() error {
	if h == nil {
		return fmt.Errorf("SSO helper must not be nil")
	}
	if h.AppSettingsModel == nil {
		h.AppSettingsModel = &AppSettingsModel{}
	}
	h.AppSettingsModel.Normalize()
	h.URL = NormalizeServerURL(h.URL)
	if h.URL == "" {
		h.URL = h.AppSettingsModel.XKDApiServerUrl
	}
	if h.SimplePassportLoginArg.Lcid == "" {
		h.SimplePassportLoginArg.Lcid = "2052"
	}
	if h.SimplePassportLoginArg.OriginType == "" {
		h.SimplePassportLoginArg.OriginType = "SimPas"
	}
	if h.Timeout <= 0 {
		h.Timeout = 30 * time.Second
	}
	return nil
}

func (h *SSOHelper) serverURL(override string) string {
	if strings.TrimSpace(override) != "" {
		return NormalizeServerURL(override)
	}
	return NormalizeServerURL(h.URL)
}

func (h *SSOHelper) signatureValues(username, timestamp string) []string {
	settings := h.AppSettingsModel
	return []string{
		settings.XKDApiAcctID, username, settings.XKDApiAppID,
		settings.XKDApiAppSec, timestamp,
	}
}

func makeSSOURLs(serverURL, encoded string) (SSOLoginURLObject, error) {
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return SSOLoginURLObject{}, fmt.Errorf("parse SSO server URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Hostname() == "" {
		return SSOLoginURLObject{}, fmt.Errorf("invalid SSO server URL %q", serverURL)
	}
	port := parsed.Port()
	if port == "" {
		if strings.EqualFold(parsed.Scheme, "https") {
			port = "443"
		} else {
			port = "80"
		}
	}
	hostPort := net.JoinHostPort(parsed.Hostname(), port)
	return SSOLoginURLObject{
		SilverlightURL: serverURL + "Silverlight/index.aspx?ud=" + encoded,
		HTML5URL:       serverURL + "html5/index.aspx?ud=" + encoded,
		WPFURL: "K3cloud://" + hostPort +
			"/k3cloud/Clientbin/K3cloudclient/K3cloudclient.manifest?Lcid=2052" +
			"&ExeType=WPFRUNTIME&LoginUrl=" + serverURL + "&ud=" + encoded,
	}, nil
}

func (h *SSOHelper) getJSONSSOURLs(version int, username, override string) (*SSOLoginURLObject, error) {
	if err := h.prepare(); err != nil {
		return nil, err
	}
	serverURL := h.serverURL(override)
	if username == "" {
		username = h.AppSettingsModel.XKDApiUserName
	}
	h.Timestamp = GetTimestamp()
	timestamp := strconv.FormatInt(h.Timestamp, 10)
	values := h.signatureValues(username, timestamp)
	if (version == 3 || version == 4) && strings.TrimSpace(h.PermitCount) != "" {
		values = append(values, h.PermitCount)
		h.SimplePassportLoginArg.OtherArgs = "|{'permitcount':'" + h.PermitCount + "'}"
	}
	var signature string
	if version == 4 {
		signature = Sha256Hex(sortedJoin(values))
	} else {
		signature = GetSignatureSHA1Util(values)
	}
	settings := h.AppSettingsModel
	h.SimplePassportLoginArg.DbID = settings.XKDApiAcctID
	h.SimplePassportLoginArg.AppID = settings.XKDApiAppID
	h.SimplePassportLoginArg.Username = username
	h.SimplePassportLoginArg.Lcid = settings.XKDApiLCID
	h.SimplePassportLoginArg.SignedData = signature
	h.SimplePassportLoginArg.Timestamp = timestamp
	h.URL = serverURL
	argJSON, err := marshalCompatibleJSON(
		h.SimplePassportLoginArg, h.UnsafeRelaxedJsonEscaping, false,
	)
	if err != nil {
		return nil, fmt.Errorf("marshal SSO arguments: %w", err)
	}
	h.ArgJSON = argJSON
	h.ArgJSONBase64 = base64.StdEncoding.EncodeToString([]byte(argJSON))
	urls, err := makeSSOURLs(h.URL, h.ArgJSONBase64)
	if err != nil {
		return nil, err
	}
	h.SSOLoginURLObject = urls
	return &h.SSOLoginURLObject, nil
}

func (h *SSOHelper) GetSSOURLsV4(username, serverURL string) (*SSOLoginURLObject, error) {
	return h.getJSONSSOURLs(4, username, serverURL)
}
func (h *SSOHelper) GetSSOURLsV3(username, serverURL string) (*SSOLoginURLObject, error) {
	return h.getJSONSSOURLs(3, username, serverURL)
}
func (h *SSOHelper) GetSSOURLsV2(username, serverURL string) (*SSOLoginURLObject, error) {
	return h.getJSONSSOURLs(2, username, serverURL)
}

func optionalSSOArgs(args []string) (string, string) {
	username, serverURL := "", ""
	if len(args) > 0 {
		username = args[0]
	}
	if len(args) > 1 {
		serverURL = args[1]
	}
	return username, serverURL
}

func (h *SSOHelper) GetSsoUrlsV4(args ...string) (*SSOLoginURLObject, error) {
	username, serverURL := optionalSSOArgs(args)
	return h.GetSSOURLsV4(username, serverURL)
}
func (h *SSOHelper) GetSsoUrlsV3(args ...string) (*SSOLoginURLObject, error) {
	username, serverURL := optionalSSOArgs(args)
	return h.GetSSOURLsV3(username, serverURL)
}
func (h *SSOHelper) GetSsoUrlsV2(args ...string) (*SSOLoginURLObject, error) {
	username, serverURL := optionalSSOArgs(args)
	return h.GetSSOURLsV2(username, serverURL)
}

// GetSSOURLsV1 builds the pipe-delimited legacy SSO payload.
func (h *SSOHelper) GetSSOURLsV1(username, override string) (*SSOLoginURLObject, error) {
	if err := h.prepare(); err != nil {
		return nil, err
	}
	serverURL := h.serverURL(override)
	if username == "" {
		username = h.AppSettingsModel.XKDApiUserName
	}
	h.Timestamp = GetTimestamp()
	timestamp := strconv.FormatInt(h.Timestamp, 10)
	signature := GetSignatureSHA1Util(h.signatureValues(username, timestamp))
	settings := h.AppSettingsModel
	payload := "|" + settings.XKDApiAcctID + "|" + username + "|" +
		settings.XKDApiAppID + "|" + signature + "|" + timestamp + "|" + settings.XKDApiLCID
	h.ArgJSON = "V1版本构建参数(非json): \n" + payload + "\n"
	h.ArgJSONBase64 = base64.StdEncoding.EncodeToString([]byte(payload))
	h.URL = serverURL
	urls, err := makeSSOURLs(serverURL, h.ArgJSONBase64)
	if err != nil {
		return nil, err
	}
	h.SSOLoginURLObject = urls
	return &h.SSOLoginURLObject, nil
}

func (h *SSOHelper) GetSsoUrlsV1(args ...string) (*SSOLoginURLObject, error) {
	username, serverURL := optionalSSOArgs(args)
	return h.GetSSOURLsV1(username, serverURL)
}

func (h *SSOHelper) getLogout(version int, username, override string) (*SSOLogoutObject, error) {
	if err := h.prepare(); err != nil {
		return nil, err
	}
	serverURL := h.serverURL(override)
	if username == "" {
		username = h.AppSettingsModel.XKDApiUserName
	}
	timestamp := GetTimestamp()
	values := h.signatureValues(username, strconv.FormatInt(timestamp, 10))
	signature := GetSignatureSHA1Util(values)
	if version == 4 {
		signature = Sha256Hex(sortedJoin(values))
	}
	payload := struct {
		AcctID     string `json:"AcctID"`
		AppID      string `json:"AppId"`
		Username   string `json:"Username"`
		SignedData string `json:"SignedData"`
		Timestamp  int64  `json:"Timestamp"`
	}{
		AcctID: h.AppSettingsModel.XKDApiAcctID, AppID: h.AppSettingsModel.XKDApiAppID,
		Username: username, SignedData: signature, Timestamp: timestamp,
	}
	ap0, err := marshalCompatibleJSON(payload, h.UnsafeRelaxedJsonEscaping, false)
	if err != nil {
		return nil, fmt.Errorf("marshal SSO logout payload: %w", err)
	}
	h.SSOLogoutObject = SSOLogoutObject{
		RequestLogoutURL: serverURL +
			"Kingdee.BOS.ServiceFacade.ServicesStub.User.UserService.LogoutByOtherSystem.common.kdsvc",
		AP0: ap0,
	}
	return &h.SSOLogoutObject, nil
}

func (h *SSOHelper) GetSSOLogoutAP0V4(username, serverURL string) (*SSOLogoutObject, error) {
	return h.getLogout(4, username, serverURL)
}
func (h *SSOHelper) GetSSOLogoutAP0V3(username, serverURL string) (*SSOLogoutObject, error) {
	return h.getLogout(3, username, serverURL)
}
func (h *SSOHelper) GetSSOLogoutAP0V2V1(username, serverURL string) (*SSOLogoutObject, error) {
	return h.getLogout(2, username, serverURL)
}

func (h *SSOHelper) GetSSOLogoutap0StrV4(args ...string) (*SSOLogoutObject, error) {
	username, serverURL := optionalSSOArgs(args)
	return h.GetSSOLogoutAP0V4(username, serverURL)
}
func (h *SSOHelper) GetSSOLogoutap0StrV3(args ...string) (*SSOLogoutObject, error) {
	username, serverURL := optionalSSOArgs(args)
	return h.GetSSOLogoutAP0V3(username, serverURL)
}
func (h *SSOHelper) GetSSOLogoutap0StrV2V1(args ...string) (*SSOLogoutObject, error) {
	username, serverURL := optionalSSOArgs(args)
	return h.GetSSOLogoutAP0V2V1(username, serverURL)
}

// SSOExecuteLogout posts the ap0 form value to the logout service.
func (h *SSOHelper) SSOExecuteLogout(logout SSOLogoutObject) (string, error) {
	return h.SSOExecuteLogoutContext(context.Background(), logout)
}

func (h *SSOHelper) SSOExecuteLogoutContext(ctx context.Context, logout SSOLogoutObject) (string, error) {
	if err := h.prepare(); err != nil {
		return "", err
	}
	web := NewWebHelper()
	web.Timeout = h.Timeout
	web.HTTPClient = h.HTTPClient
	web.BodyType = BodyTypeURLEncoded
	web.BodyURLEncoded.Set("ap0", logout.AP0)
	return web.SendHttpRequestContext(ctx, logout.RequestLogoutURL)
}

// SSOExcuteLogout retains the original misspelling for migration.
func (h *SSOHelper) SSOExcuteLogout(logout SSOLogoutObject) (string, error) {
	return h.SSOExecuteLogout(logout)
}
