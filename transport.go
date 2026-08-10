package yikdwebclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HTTPError describes a non-2xx response and preserves its response body.
type HTTPError struct {
	StatusCode int
	Status     string
	URL        string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP request to %s failed: %s", e.URL, e.Status)
}

func cloneHeader(source http.Header) http.Header {
	if source == nil {
		return make(http.Header)
	}
	return source.Clone()
}

func newCookieJar() http.CookieJar {
	jar, _ := cookiejar.New(nil)
	return jar
}

func effectiveHTTPClient(base *http.Client, timeout time.Duration, jar http.CookieJar) *http.Client {
	client := http.Client{}
	if base != nil {
		client = *base
	}
	if timeout > 0 {
		client.Timeout = timeout
	}
	if jar != nil {
		client.Jar = jar
	}
	return &client
}

// WebHelperServices is the JSON HTTP transport used by authentication and API calls.
type WebHelperServices struct {
	Cookies         http.CookieJar
	RequestHeaders  http.Header
	ResponseHeaders http.Header
	HTTPMethod      string
	Timeout         time.Duration
	HTTPClient      *http.Client
}

// NewWebHelperServices creates the JSON transport with a cookie jar and POST method.
func NewWebHelperServices() *WebHelperServices {
	return &WebHelperServices{
		Cookies: newCookieJar(), RequestHeaders: make(http.Header), HTTPMethod: http.MethodPost,
	}
}

func (h *WebHelperServices) prepare() {
	if h.Cookies == nil {
		if h.HTTPClient != nil && h.HTTPClient.Jar != nil {
			h.Cookies = h.HTTPClient.Jar
		} else {
			h.Cookies = newCookieJar()
		}
	}
	if h.HTTPMethod == "" {
		h.HTTPMethod = http.MethodPost
	}
}

// SendHttpRequest sends a request using context.Background. postData defaults to empty.
func (h *WebHelperServices) SendHttpRequest(requestURL string, postData ...string) (string, error) {
	payload := ""
	if len(postData) > 0 {
		payload = postData[0]
	}
	return h.SendHttpRequestContext(context.Background(), requestURL, payload)
}

// SendHttpRequestContext sends a JSON request and updates response headers and cookies.
func (h *WebHelperServices) SendHttpRequestContext(ctx context.Context, requestURL, postData string) (string, error) {
	h.prepare()
	request, err := http.NewRequestWithContext(ctx, h.HTTPMethod, requestURL, strings.NewReader(postData))
	if err != nil {
		return "", fmt.Errorf("create HTTP request: %w", err)
	}
	request.Header.Set("Accept-Charset", "utf-8")
	request.Header.Set("Content-Type", MediaTypeApplicationJSON+"; charset=utf-8")
	for key, values := range h.RequestHeaders {
		request.Header.Del(key)
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	response, err := effectiveHTTPClient(h.HTTPClient, h.Timeout, h.Cookies).Do(request)
	if err != nil {
		return "", fmt.Errorf("send HTTP request: %w", err)
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(response.Body)
	h.ResponseHeaders = response.Header.Clone()
	if readErr != nil {
		return "", fmt.Errorf("read HTTP response: %w", readErr)
	}
	text := string(body)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return text, &HTTPError{
			StatusCode: response.StatusCode, Status: response.Status, URL: requestURL, Body: text,
		}
	}
	return text, nil
}

// MultipartFile is one in-memory file part.
type MultipartFile struct {
	FieldName   string
	FileName    string
	ContentType string
	Data        []byte
}

// MultipartFormData contains fields and file parts for WebHelper.
type MultipartFormData struct {
	Fields url.Values
	Files  []MultipartFile
}

// NewMultipartFormData creates an empty multipart body.
func NewMultipartFormData() *MultipartFormData {
	return &MultipartFormData{Fields: make(url.Values)}
}

// AddField appends a text form field.
func (m *MultipartFormData) AddField(name, value string) *MultipartFormData {
	if m.Fields == nil {
		m.Fields = make(url.Values)
	}
	m.Fields.Add(name, value)
	return m
}

// AddFile appends an in-memory file part.
func (m *MultipartFormData) AddFile(name, fileName, contentType string, data []byte) *MultipartFormData {
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	m.Files = append(m.Files, MultipartFile{
		FieldName: name, FileName: fileName, ContentType: contentType,
		Data: append([]byte(nil), data...),
	})
	return m
}

// AddFilePath reads a file and appends it as a multipart part.
func (m *MultipartFormData) AddFilePath(name, path, contentType string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read multipart file %q: %w", path, err)
	}
	m.AddFile(name, filepath.Base(path), contentType, data)
	return nil
}

func (m *MultipartFormData) encode() ([]byte, string, error) {
	var output bytes.Buffer
	writer := multipart.NewWriter(&output)
	for key, values := range m.Fields {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return nil, "", fmt.Errorf("write multipart field %q: %w", key, err)
			}
		}
	}
	for _, file := range m.Files {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, file.FieldName, file.FileName))
		header.Set("Content-Type", file.ContentType)
		part, err := writer.CreatePart(header)
		if err != nil {
			return nil, "", fmt.Errorf("create multipart file %q: %w", file.FileName, err)
		}
		if _, err := part.Write(file.Data); err != nil {
			return nil, "", fmt.Errorf("write multipart file %q: %w", file.FileName, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart body: %w", err)
	}
	return output.Bytes(), writer.FormDataContentType(), nil
}

// WebHelper is the general HTTP helper used by SSO logout and custom requests.
type WebHelper struct {
	QueryParameters  url.Values
	RequestCookies   http.CookieJar
	ResponseCookies  http.CookieJar
	RequestHeaders   http.Header
	ResponseHeaders  http.Header
	HTTPMethod       string
	BodyType         BodyType
	BodyFormData     *MultipartFormData
	BodyURLEncoded   url.Values
	BodyRaw          string
	RequestMediaType string
	Timeout          time.Duration
	HTTPClient       *http.Client
}

// NewWebHelper creates a general HTTP helper with C#-equivalent defaults.
func NewWebHelper() *WebHelper {
	return &WebHelper{
		QueryParameters: make(url.Values), RequestCookies: newCookieJar(),
		RequestHeaders: make(http.Header), HTTPMethod: http.MethodPost,
		BodyType: BodyTypeNone, BodyFormData: NewMultipartFormData(),
		BodyURLEncoded: make(url.Values), RequestMediaType: MediaTypeApplicationJSON,
		Timeout: 120 * time.Second,
	}
}

func (h *WebHelper) prepare() {
	if h.RequestCookies == nil {
		if h.HTTPClient != nil && h.HTTPClient.Jar != nil {
			h.RequestCookies = h.HTTPClient.Jar
		} else {
			h.RequestCookies = newCookieJar()
		}
	}
	if h.HTTPMethod == "" {
		h.HTTPMethod = http.MethodPost
	}
	if h.RequestMediaType == "" {
		h.RequestMediaType = MediaTypeApplicationJSON
	}
}

// CreateQueryString returns a deterministically sorted URL query string.
func CreateQueryString(parameters map[string]string) string {
	values := make(url.Values, len(parameters))
	for key, value := range parameters {
		values.Set(key, value)
	}
	return values.Encode()
}

// SendHttpRequest sends a request using context.Background.
func (h *WebHelper) SendHttpRequest(requestURL string) (string, error) {
	return h.SendHttpRequestContext(context.Background(), requestURL)
}

// SendHttpRequestContext sends a general request and updates response state.
func (h *WebHelper) SendHttpRequestContext(ctx context.Context, requestURL string) (string, error) {
	h.prepare()
	parsed, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("parse request URL: %w", err)
	}
	query := parsed.Query()
	for key, values := range h.QueryParameters {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()

	var body io.Reader
	contentType := ""
	switch h.BodyType {
	case BodyTypeNone, "":
	case BodyTypeRaw:
		body = strings.NewReader(h.BodyRaw)
		contentType = h.RequestMediaType + "; charset=utf-8"
	case BodyTypeURLEncoded:
		body = strings.NewReader(h.BodyURLEncoded.Encode())
		contentType = MediaTypeApplicationFormURLEncoded
	case BodyTypeFormData:
		if h.BodyFormData == nil {
			h.BodyFormData = NewMultipartFormData()
		}
		encoded, mediaType, encodeErr := h.BodyFormData.encode()
		if encodeErr != nil {
			return "", encodeErr
		}
		body = bytes.NewReader(encoded)
		contentType = mediaType
	default:
		return "", fmt.Errorf("unsupported body type %q", h.BodyType)
	}

	request, err := http.NewRequestWithContext(ctx, h.HTTPMethod, parsed.String(), body)
	if err != nil {
		return "", fmt.Errorf("create HTTP request: %w", err)
	}
	request.Header.Set("Accept-Charset", "utf-8")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	for key, values := range h.RequestHeaders {
		request.Header.Del(key)
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	response, err := effectiveHTTPClient(h.HTTPClient, h.Timeout, h.RequestCookies).Do(request)
	if err != nil {
		return "", fmt.Errorf("send HTTP request: %w", err)
	}
	defer response.Body.Close()
	responseBody, readErr := io.ReadAll(response.Body)
	h.ResponseHeaders = response.Header.Clone()
	h.ResponseCookies = h.RequestCookies
	if readErr != nil {
		return "", fmt.Errorf("read HTTP response: %w", readErr)
	}
	text := string(responseBody)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return text, &HTTPError{
			StatusCode: response.StatusCode, Status: response.Status, URL: parsed.String(), Body: text,
		}
	}
	return text, nil
}
