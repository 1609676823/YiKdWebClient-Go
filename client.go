package yikdwebclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const dynamicFormPrefix = "Kingdee.BOS.WebApi.ServicesStub.DynamicFormService."

// Instance is updated by ExecAPIDynamicFormService for compatibility with the
// original static property. New code should keep its own client reference.
var Instance *YiK3CloudClient

var ErrLoginFailed = errors.New("Kingdee login was not accepted")

// LoginError contains the raw response returned by a rejected login.
type LoginError struct{ Response string }

func (e *LoginError) Error() string {
	if e.Response == "" {
		return ErrLoginFailed.Error()
	}
	return ErrLoginFailed.Error() + ": " + e.Response
}

func (e *LoginError) Unwrap() error { return ErrLoginFailed }

type callSettings struct {
	autoLogin  bool
	autoLogout bool
	rawJSON    bool
}

// CallOption customizes one business operation.
type CallOption func(*callSettings)

// WithAutoLogin controls automatic login. The default is true.
func WithAutoLogin(enabled bool) CallOption {
	return func(settings *callSettings) { settings.autoLogin = enabled }
}

// WithAutoLogout controls automatic logout. The default is true.
func WithAutoLogout(enabled bool) CallOption {
	return func(settings *callSettings) { settings.autoLogout = enabled }
}

// WithRawJSON sends the supplied payload without a Kingdee parameters envelope.
func WithRawJSON(enabled bool) CallOption {
	return func(settings *callSettings) { settings.rawJSON = enabled }
}

func resolveCallOptions(options []CallOption) callSettings {
	settings := callSettings{autoLogin: true, autoLogout: true}
	for _, option := range options {
		if option != nil {
			option(&settings)
		}
	}
	return settings
}

// ClientOption customizes a newly constructed client.
type ClientOption func(*YiK3CloudClient) error

// WithAppSettings supplies authentication and endpoint configuration.
func WithAppSettings(settings *AppSettingsModel) ClientOption {
	return func(client *YiK3CloudClient) error {
		if settings == nil {
			return fmt.Errorf("app settings must not be nil")
		}
		settings.Normalize()
		client.AppSettingsModel = settings
		return nil
	}
}

// WithHTTPClient injects an HTTP client, which is useful for custom transports.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *YiK3CloudClient) error {
		if httpClient == nil {
			return fmt.Errorf("HTTP client must not be nil")
		}
		client.HTTPClient = httpClient
		if httpClient.Jar != nil {
			client.Cookie = httpClient.Jar
		}
		return nil
	}
}

// WithClientLoginType selects the authentication mode.
func WithClientLoginType(loginType LoginType) ClientOption {
	return func(client *YiK3CloudClient) error {
		client.LoginType = loginType
		return nil
	}
}

// WithClientTimeout sets the per-request timeout.
func WithClientTimeout(timeout time.Duration) ClientOption {
	return func(client *YiK3CloudClient) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be greater than zero")
		}
		client.Timeout = timeout
		return nil
	}
}

// YiK3CloudClient is the stateful Kingdee K3 Cloud WebAPI client.
// A client should not be mutated concurrently while requests are in flight.
type YiK3CloudClient struct {
	Cookie                     http.CookieJar
	RequestHeaders             http.Header
	RequestHeadersString       string
	LoginType                  LoginType
	AppSettingsModel           *AppSettingsModel
	ReturnLoginWebModel        *RequestWebModel
	ReturnOperationWebModel    *RequestWebModel
	UnsafeRelaxedJsonEscaping  bool
	Timeout                    time.Duration
	ValidateLoginSettingsModel *ValidateLoginSettingsModel
	LoginBySimplePassportModel *LoginBySimplePassportModel
	HTTPClient                 *http.Client
	closed                     bool
}

// Client is the idiomatic short alias for YiK3CloudClient.
type Client = YiK3CloudClient

// NewClient creates a client. If the default XML configuration exists, it is
// loaded; a missing default file is not an error.
func NewClient(options ...ClientOption) (*YiK3CloudClient, error) {
	settings, err := LoadAppSettingsIfExists("")
	if err != nil {
		// The original parameterless constructor tolerates a missing or malformed
		// default configuration. Explicit NewClientFromConfig calls still report it.
		settings = &AppSettingsModel{}
	}
	client := &YiK3CloudClient{
		Cookie: newCookieJar(), RequestHeaders: make(http.Header),
		LoginType: LoginTypeAppSecret, AppSettingsModel: settings,
		ReturnLoginWebModel: &RequestWebModel{}, ReturnOperationWebModel: &RequestWebModel{},
		UnsafeRelaxedJsonEscaping: true, Timeout: 60 * time.Second,
	}
	for _, option := range options {
		if option != nil {
			if err := option(client); err != nil {
				return nil, err
			}
		}
	}
	client.prepare()
	return client, nil
}

// NewClientFromConfig creates a client from an explicit XML configuration path.
func NewClientFromConfig(path string, options ...ClientOption) (*YiK3CloudClient, error) {
	settings, err := LoadAppSettings(path)
	if err != nil {
		return nil, err
	}
	options = append([]ClientOption{WithAppSettings(settings)}, options...)
	return NewClient(options...)
}

func (c *YiK3CloudClient) prepare() {
	if c.Cookie == nil {
		if c.HTTPClient != nil && c.HTTPClient.Jar != nil {
			c.Cookie = c.HTTPClient.Jar
		} else {
			c.Cookie = newCookieJar()
		}
	}
	if c.RequestHeaders == nil {
		c.RequestHeaders = make(http.Header)
	}
	if c.AppSettingsModel == nil {
		c.AppSettingsModel = &AppSettingsModel{}
	}
	c.AppSettingsModel.Normalize()
	if c.ReturnLoginWebModel == nil {
		c.ReturnLoginWebModel = &RequestWebModel{}
	}
	if c.ReturnOperationWebModel == nil {
		c.ReturnOperationWebModel = &RequestWebModel{}
	}
	if c.Timeout <= 0 {
		c.Timeout = 60 * time.Second
	}
}

func (c *YiK3CloudClient) ensureOpen() error {
	if c == nil {
		return fmt.Errorf("client must not be nil")
	}
	if c.closed || c.LoginType == "" {
		return fmt.Errorf("client is closed or LoginType is empty")
	}
	c.prepare()
	return nil
}

func (c *YiK3CloudClient) operationBaseURL() (string, error) {
	switch c.LoginType {
	case LoginTypeValidateLogin, LoginTypeValidateUserCode:
		if c.ValidateLoginSettingsModel == nil {
			return "", fmt.Errorf("ValidateLoginSettingsModel 需要实例化赋值")
		}
		c.ValidateLoginSettingsModel.Normalize()
		return c.ValidateLoginSettingsModel.Url, nil
	case LoginTypeSimplePassport:
		if c.LoginBySimplePassportModel == nil {
			return "", fmt.Errorf("LoginBySimplePassportModel 需要实例化赋值")
		}
		c.LoginBySimplePassportModel.Normalize()
		return c.LoginBySimplePassportModel.Url, nil
	default:
		return NormalizeServerURL(c.AppSettingsModel.XKDApiServerUrl), nil
	}
}

func (c *YiK3CloudClient) execServiceContext(
	ctx context.Context, formID, payload, servicePath, opNumber string, rawJSON bool,
) (string, error) {
	baseURL, err := c.operationBaseURL()
	if err != nil {
		return "", err
	}
	requestURL := baseURL + servicePath
	web := NewWebHelperServices()
	web.Timeout = c.Timeout
	web.HTTPClient = c.HTTPClient
	if c.LoginType == LoginTypeAPISignHeaders {
		headers, headerErr := GetAPIHeaders(c.AppSettingsModel, requestURL)
		if headerErr != nil {
			return "", headerErr
		}
		c.RequestHeaders = headers
		c.RequestHeadersString = GetAPIHeadersString(headers)
	} else {
		web.Cookies = c.Cookie
	}
	web.RequestHeaders = cloneHeader(c.RequestHeaders)
	requestBody := payload
	if !rawJSON {
		requestBody, err = GetRequestBodyString(formID, payload, c.UnsafeRelaxedJsonEscaping, opNumber)
		if err != nil {
			return "", err
		}
	}
	c.ReturnOperationWebModel.RequestUrl = requestURL
	c.ReturnOperationWebModel.RealRequestBody = requestBody
	response, requestErr := web.SendHttpRequestContext(ctx, requestURL, requestBody)
	c.ReturnOperationWebModel.RealResponseBody = response
	c.ReturnOperationWebModel.Cookie = web.Cookies
	c.Cookie = web.Cookies
	if requestErr != nil {
		return response, requestErr
	}
	return response, nil
}

func acceptedLogin(response string) bool {
	var body struct {
		LoginResultType json.RawMessage `json:"LoginResultType"`
	}
	if err := json.Unmarshal([]byte(response), &body); err != nil {
		return false
	}
	value := strings.TrimSpace(string(body.LoginResultType))
	value = strings.Trim(value, `"`)
	return strings.EqualFold(value, "1")
}

func (c *YiK3CloudClient) ensureLoggedIn(ctx context.Context, autoLogin bool) error {
	if c.LoginType == LoginTypeAPISignHeaders {
		return nil
	}
	if autoLogin {
		if _, err := c.LoginContext(ctx); err != nil {
			return err
		}
	}
	if !acceptedLogin(c.ReturnLoginWebModel.RealResponseBody) {
		return &LoginError{Response: c.ReturnLoginWebModel.RealResponseBody}
	}
	return nil
}

// Login authenticates using context.Background.
func (c *YiK3CloudClient) Login() (*RequestWebModel, error) {
	return c.LoginContext(context.Background())
}

// LoginContext authenticates using the configured LoginType.
func (c *YiK3CloudClient) LoginContext(ctx context.Context) (*RequestWebModel, error) {
	if err := c.ensureOpen(); err != nil {
		return nil, err
	}
	if c.LoginType == LoginTypeAPISignHeaders {
		result := &RequestWebModel{}
		c.ReturnLoginWebModel = result
		return result, nil
	}
	transport := AuthenticationService{
		Timeout: c.Timeout, RequestHeaders: cloneHeader(c.RequestHeaders), HTTPClient: c.HTTPClient,
	}
	var (
		payload   string
		serverURL string
		result    *RequestWebModel
		err       error
	)
	switch c.LoginType {
	case LoginTypeAppSecret:
		service := &LoginByAppSecret{AuthenticationService: transport}
		payload, err = service.GetLoginJSON(c.AppSettingsModel, c.UnsafeRelaxedJsonEscaping)
		serverURL = c.AppSettingsModel.XKDApiServerUrl
		if err == nil {
			result, err = service.LoginContext(ctx, serverURL, payload)
		}
	case LoginTypeSignSHA256, LoginTypeSignSHA1:
		service := &LoginBySign{AuthenticationService: transport, LoginType: c.LoginType}
		payload, err = service.GetLoginJSON(c.AppSettingsModel, c.UnsafeRelaxedJsonEscaping)
		serverURL = c.AppSettingsModel.XKDApiServerUrl
		if err == nil {
			result, err = service.LoginContext(ctx, serverURL, payload)
		}
	case LoginTypeValidateLogin:
		if err = c.requireValidateSettings("ValidateLogin"); err == nil {
			service := &ValidateLogin{AuthenticationService: transport}
			payload, err = service.GetLoginJSON(c.ValidateLoginSettingsModel, c.UnsafeRelaxedJsonEscaping)
			serverURL = c.ValidateLoginSettingsModel.Url
			if err == nil {
				result, err = service.LoginContext(ctx, serverURL, payload)
			}
		}
	case LoginTypeValidateUserCode:
		if err = c.requireValidateSettings("ValidateUserEnDeCode"); err == nil {
			service := &ValidateUserEnDeCode{AuthenticationService: transport}
			payload, err = service.GetLoginJSON(c.ValidateLoginSettingsModel, c.UnsafeRelaxedJsonEscaping)
			serverURL = c.ValidateLoginSettingsModel.Url
			if err == nil {
				result, err = service.LoginContext(ctx, serverURL, payload)
			}
		}
	case LoginTypeSimplePassport:
		if err = c.requireSimplePassportSettings(); err == nil {
			service := &LoginBySimplePassport{AuthenticationService: transport}
			payload, err = service.GetLoginJSON(c.LoginBySimplePassportModel, c.UnsafeRelaxedJsonEscaping)
			serverURL = c.LoginBySimplePassportModel.Url
			if err == nil {
				result, err = service.LoginContext(ctx, serverURL, payload)
			}
		}
	default:
		err = fmt.Errorf("unsupported LoginType %q", c.LoginType)
	}
	if result == nil {
		result = &RequestWebModel{RealRequestBody: payload}
	}
	c.ReturnLoginWebModel = result
	if result.Cookie != nil {
		c.Cookie = result.Cookie
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func (c *YiK3CloudClient) requireValidateSettings(name string) error {
	if c.ValidateLoginSettingsModel == nil {
		return fmt.Errorf("LoginType 使用 %s 时 ValidateLoginSettingsModel 需要实例化赋值", name)
	}
	c.ValidateLoginSettingsModel.Normalize()
	if strings.TrimSpace(c.ValidateLoginSettingsModel.Url) == "" {
		return fmt.Errorf("LoginType 使用 %s 时 ValidateLoginSettingsModel.Url 需要赋值", name)
	}
	return nil
}

func (c *YiK3CloudClient) requireSimplePassportSettings() error {
	if c.LoginBySimplePassportModel == nil {
		return fmt.Errorf("LoginType 使用 LoginBySimplePassport 时 LoginBySimplePassportModel 需要实例化赋值")
	}
	c.LoginBySimplePassportModel.Normalize()
	if strings.TrimSpace(c.LoginBySimplePassportModel.Url) == "" {
		return fmt.Errorf("LoginType 使用 LoginBySimplePassport 时 LoginBySimplePassportModel.Url 需要赋值")
	}
	return nil
}

// Logout sends the Kingdee logout request. Business methods intentionally do
// not replace a successful business response when automatic logout fails.
func (c *YiK3CloudClient) Logout() error {
	return c.LogoutContext(context.Background())
}

// LogoutContext sends the Kingdee logout request.
func (c *YiK3CloudClient) LogoutContext(ctx context.Context) error {
	if err := c.ensureOpen(); err != nil {
		return err
	}
	web := NewWebHelperServices()
	web.Cookies = c.Cookie
	web.Timeout = c.Timeout
	web.RequestHeaders = cloneHeader(c.RequestHeaders)
	web.HTTPClient = c.HTTPClient
	requestURL := NormalizeServerURL(c.AppSettingsModel.XKDApiServerUrl) +
		"Kingdee.BOS.WebApi.ServicesStub.AuthService.Logout.common.kdsvc"
	_, err := web.SendHttpRequestContext(ctx, requestURL, "")
	c.Cookie = web.Cookies
	return err
}

// ExecAPIDynamicFormService invokes an arbitrary Kingdee service stub.
func (c *YiK3CloudClient) ExecAPIDynamicFormService(
	formID, payload, servicePath string, options ...CallOption,
) (string, error) {
	return c.ExecAPIDynamicFormServiceContext(
		context.Background(), formID, payload, servicePath, options...,
	)
}

// ExecApiDynamicFormService is the C#-style spelling kept for migration.
func (c *YiK3CloudClient) ExecApiDynamicFormService(
	formID, payload, servicePath string, options ...CallOption,
) (string, error) {
	return c.ExecAPIDynamicFormService(formID, payload, servicePath, options...)
}

// ExecAPIDynamicFormServiceContext invokes an arbitrary stub with cancellation.
func (c *YiK3CloudClient) ExecAPIDynamicFormServiceContext(
	ctx context.Context, formID, payload, servicePath string, options ...CallOption,
) (string, error) {
	if err := c.ensureOpen(); err != nil {
		return "", err
	}
	Instance = c
	settings := resolveCallOptions(options)
	if err := c.ensureLoggedIn(ctx, settings.autoLogin); err != nil {
		return c.ReturnLoginWebModel.RealResponseBody, err
	}
	response, err := c.execServiceContext(ctx, formID, payload, servicePath, "", settings.rawJSON)
	if err != nil {
		return response, err
	}
	if settings.autoLogout {
		_ = c.LogoutContext(ctx)
	}
	return response, nil
}

func dynamicFormPath(operation string) string {
	return dynamicFormPrefix + operation + ".common.kdsvc"
}

func (c *YiK3CloudClient) callForm(operation, formID, payload string, options ...CallOption) (string, error) {
	return c.ExecAPIDynamicFormService(formID, payload, dynamicFormPath(operation), options...)
}

func (c *YiK3CloudClient) callPayload(operation, payload string, options ...CallOption) (string, error) {
	return c.ExecAPIDynamicFormService("", payload, dynamicFormPath(operation), options...)
}

func (c *YiK3CloudClient) callRaw(operation, payload string, options ...CallOption) (string, error) {
	options = append(options, WithRawJSON(true))
	return c.ExecAPIDynamicFormService("", payload, dynamicFormPath(operation), options...)
}

// ExecuteOperation invokes DynamicFormService.ExecuteOperation.
func (c *YiK3CloudClient) ExecuteOperation(formID, opNumber, payload string, options ...CallOption) (string, error) {
	return c.ExecuteOperationContext(context.Background(), formID, opNumber, payload, options...)
}

func (c *YiK3CloudClient) ExecuteOperationContext(
	ctx context.Context, formID, opNumber, payload string, options ...CallOption,
) (string, error) {
	if err := c.ensureOpen(); err != nil {
		return "", err
	}
	settings := resolveCallOptions(options)
	if err := c.ensureLoggedIn(ctx, settings.autoLogin); err != nil {
		return c.ReturnLoginWebModel.RealResponseBody, err
	}
	response, err := c.execServiceContext(
		ctx, formID, payload, dynamicFormPath("ExecuteOperation"), opNumber, false,
	)
	if err != nil {
		return response, err
	}
	if settings.autoLogout {
		_ = c.LogoutContext(ctx)
	}
	return response, nil
}

func (c *YiK3CloudClient) View(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("View", formID, payload, options...)
}
func (c *YiK3CloudClient) Save(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("Save", formID, payload, options...)
}
func (c *YiK3CloudClient) BatchSave(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("BatchSave", formID, payload, options...)
}
func (c *YiK3CloudClient) Submit(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("Submit", formID, payload, options...)
}
func (c *YiK3CloudClient) Audit(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("Audit", formID, payload, options...)
}
func (c *YiK3CloudClient) UnAudit(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("UnAudit", formID, payload, options...)
}
func (c *YiK3CloudClient) Delete(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("Delete", formID, payload, options...)
}
func (c *YiK3CloudClient) ExecuteBillQuery(payload string, options ...CallOption) (string, error) {
	return c.callPayload("ExecuteBillQuery", payload, options...)
}
func (c *YiK3CloudClient) Draft(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("Draft", formID, payload, options...)
}
func (c *YiK3CloudClient) Allocate(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("Allocate", formID, payload, options...)
}
func (c *YiK3CloudClient) Push(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("Push", formID, payload, options...)
}
func (c *YiK3CloudClient) GroupSave(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("GroupSave", formID, payload, options...)
}
func (c *YiK3CloudClient) FlexSave(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("FlexSave", formID, payload, options...)
}
func (c *YiK3CloudClient) SendMsg(payload string, options ...CallOption) (string, error) {
	return c.callPayload("SendMsg", payload, options...)
}
func (c *YiK3CloudClient) SwitchOrg(payload string, options ...CallOption) (string, error) {
	return c.callPayload("SwitchOrg", payload, options...)
}
func (c *YiK3CloudClient) WorkflowAudit(payload string, options ...CallOption) (string, error) {
	return c.callPayload("WorkflowAudit", payload, options...)
}
func (c *YiK3CloudClient) GetSysReportData(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("GetSysReportData", formID, payload, options...)
}
func (c *YiK3CloudClient) AttachmentUpLoad(payload string, options ...CallOption) (string, error) {
	return c.callRaw("AttachmentUpLoad", payload, options...)
}
func (c *YiK3CloudClient) AttachmentDownLoad(payload string, options ...CallOption) (string, error) {
	return c.callRaw("AttachmentDownLoad", payload, options...)
}
func (c *YiK3CloudClient) UploadFile(payload string, options ...CallOption) (string, error) {
	return c.callRaw("UploadFile", payload, options...)
}
func (c *YiK3CloudClient) GroupDelete(payload string, options ...CallOption) (string, error) {
	return c.callPayload("GroupDelete", payload, options...)
}
func (c *YiK3CloudClient) CancelAllocate(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("CancelAllocate", formID, payload, options...)
}
func (c *YiK3CloudClient) CancelAssign(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("CancelAssign", formID, payload, options...)
}
func (c *YiK3CloudClient) Disassembly(formID, payload string, options ...CallOption) (string, error) {
	return c.callForm("Disassembly", formID, payload, options...)
}
func (c *YiK3CloudClient) QueryBusinessInfo(payload string, options ...CallOption) (string, error) {
	return c.callPayload("QueryBusinessInfo", payload, options...)
}
func (c *YiK3CloudClient) QueryGroupInfo(payload string, options ...CallOption) (string, error) {
	return c.callPayload("QueryGroupInfo", payload, options...)
}

// CustomBusinessService invokes a named custom stub using a standard envelope.
func (c *YiK3CloudClient) CustomBusinessService(payload, apiName string, options ...CallOption) (string, error) {
	return c.ExecAPIDynamicFormService("", payload, EnsureSuffixServicesStub(apiName), options...)
}

// CustomBusinessServiceByStubpath invokes a structured custom stub.
func (c *YiK3CloudClient) CustomBusinessServiceByStubpath(
	payload string, stub CustomServicesStubpath, options ...CallOption,
) (string, error) {
	return c.ExecAPIDynamicFormService("", payload, stub.GetCustomServicesStubpathURL(), options...)
}

// CustomBusinessServiceByParameters sends a raw parameters payload.
func (c *YiK3CloudClient) CustomBusinessServiceByParameters(
	payload, apiName string, options ...CallOption,
) (string, error) {
	options = append(options, WithRawJSON(true))
	return c.ExecAPIDynamicFormService("", payload, EnsureSuffixServicesStub(apiName), options...)
}

// CustomBusinessServiceByParametersAndStubpath sends raw parameters to a structured stub.
func (c *YiK3CloudClient) CustomBusinessServiceByParametersAndStubpath(
	payload string, stub CustomServicesStubpath, options ...CallOption,
) (string, error) {
	options = append(options, WithRawJSON(true))
	return c.ExecAPIDynamicFormService("", payload, stub.GetCustomServicesStubpathURL(), options...)
}

// GetDataCenterList obtains available Kingdee data centers. An optional URL
// overrides AppSettingsModel.XKDApiServerUrl.
func (c *YiK3CloudClient) GetDataCenterList(serverURL ...string) (string, error) {
	return c.GetDataCenterListContext(context.Background(), serverURL...)
}

func (c *YiK3CloudClient) GetDataCenterListContext(ctx context.Context, serverURL ...string) (string, error) {
	if err := c.ensureOpen(); err != nil {
		return "", err
	}
	baseURL := c.AppSettingsModel.XKDApiServerUrl
	if len(serverURL) > 0 && strings.TrimSpace(serverURL[0]) != "" {
		baseURL = NormalizeServerURL(serverURL[0])
	}
	requestURL := NormalizeServerURL(baseURL) +
		"Kingdee.BOS.ServiceFacade.ServicesStub.Account.AccountService.GetDataCenterList.common.kdsvc"
	web := NewWebHelperServices()
	web.Cookies = c.Cookie
	web.Timeout = c.Timeout
	web.RequestHeaders = cloneHeader(c.RequestHeaders)
	web.HTTPClient = c.HTTPClient
	response, err := web.SendHttpRequestContext(ctx, requestURL, "")
	c.Cookie = web.Cookies
	return response, err
}

// GetServerURL normalizes a server URL.
func (c *YiK3CloudClient) GetServerURL(value string) string { return NormalizeServerURL(value) }

// GetServerUrl is the C#-style spelling kept for migration.
func (c *YiK3CloudClient) GetServerUrl(value string) string { return c.GetServerURL(value) }

// EnsureSuffixServicesStub appends a service suffix when it is absent.
func EnsureSuffixServicesStub(input string, suffix ...string) string {
	actualSuffix := ".common.kdsvc"
	if len(suffix) > 0 {
		actualSuffix = suffix[0]
	}
	if strings.HasSuffix(input, actualSuffix) {
		return input
	}
	return input + actualSuffix
}

// Close marks the stateful client closed. net/http resources remain reusable.
func (c *YiK3CloudClient) Close() error {
	if c == nil || c.closed {
		return nil
	}
	c.closed = true
	c.LoginType = ""
	return nil
}

// Dispose is the C#-style alias for Close.
func (c *YiK3CloudClient) Dispose() { _ = c.Close() }
