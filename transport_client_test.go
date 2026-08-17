package yikdwebclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type recordedRequest struct {
	Method string
	Path   string
	Header http.Header
	Body   string
}

type requestRecorder struct {
	mu       sync.Mutex
	requests []recordedRequest
	respond  func(recordedRequest) (int, string, http.Header)
}

func (r *requestRecorder) handler(writer http.ResponseWriter, request *http.Request) {
	body, _ := io.ReadAll(request.Body)
	recorded := recordedRequest{
		Method: request.Method, Path: request.URL.RequestURI(), Header: request.Header.Clone(), Body: string(body),
	}
	r.mu.Lock()
	r.requests = append(r.requests, recorded)
	r.mu.Unlock()
	status, responseBody, headers := http.StatusOK, `{"ok":true}`, make(http.Header)
	if r.respond != nil {
		status, responseBody, headers = r.respond(recorded)
	}
	for key, values := range headers {
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}
	writer.WriteHeader(status)
	_, _ = writer.Write([]byte(responseBody))
}

func (r *requestRecorder) snapshot() []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recordedRequest(nil), r.requests...)
}

func newTestServer(recorder *requestRecorder) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(recorder.handler))
}

func TestWebHelperServices(t *testing.T) {
	recorder := &requestRecorder{respond: func(recordedRequest) (int, string, http.Header) {
		return http.StatusOK, `{"ok":true}`, http.Header{"Set-Cookie": {"next=xyz; Path=/"}}
	}}
	server := newTestServer(recorder)
	defer server.Close()

	helper := NewWebHelperServices()
	helper.Timeout = 5 * time.Second
	helper.RequestHeaders.Set("X-Test", "value")
	parsed, _ := url.Parse(server.URL)
	helper.Cookies.SetCookies(parsed, []*http.Cookie{{Name: "session", Value: "abc", Path: "/"}})
	response, err := helper.SendHttpRequest(server.URL+"/api", `{"id":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if response != `{"ok":true}` {
		t.Fatalf("response = %q", response)
	}
	request := recorder.snapshot()[0]
	if request.Method != http.MethodPost || request.Body != `{"id":1}` || request.Header.Get("X-Test") != "value" {
		t.Fatalf("request = %#v", request)
	}
	if !strings.Contains(request.Header.Get("Cookie"), "session=abc") {
		t.Fatalf("cookie header = %q", request.Header.Get("Cookie"))
	}
	if cookies := helper.Cookies.Cookies(parsed); len(cookies) == 0 {
		t.Fatal("response cookie was not retained")
	}
}

func TestWebHelperBodyTypesAndErrors(t *testing.T) {
	recorder := &requestRecorder{}
	server := newTestServer(recorder)
	defer server.Close()

	query := NewWebHelper()
	query.HTTPMethod = http.MethodGet
	query.QueryParameters.Set("a b", "x+y")
	query.QueryParameters.Set("中文", "值")
	if _, err := query.SendHttpRequest(server.URL + "/query"); err != nil {
		t.Fatal(err)
	}
	if path := recorder.snapshot()[0].Path; !strings.Contains(path, "a+b=x%2By") || !strings.Contains(path, "%E4%B8%AD%E6%96%87") {
		t.Fatalf("query path = %q", path)
	}

	raw := NewWebHelper()
	raw.BodyType = BodyTypeRaw
	raw.BodyRaw = "raw-body"
	raw.RequestMediaType = MediaTypeTextPlain
	if _, err := raw.SendHttpRequest(server.URL + "/raw"); err != nil {
		t.Fatal(err)
	}
	rawRequest := recorder.snapshot()[1]
	if rawRequest.Body != "raw-body" || !strings.Contains(rawRequest.Header.Get("Content-Type"), "text/plain") {
		t.Fatalf("raw request = %#v", rawRequest)
	}

	form := NewWebHelper()
	form.BodyType = BodyTypeURLEncoded
	form.BodyURLEncoded.Set("ap0", `{"id":1}`)
	form.BodyURLEncoded.Set("space", "a b")
	if _, err := form.SendHttpRequest(server.URL + "/form"); err != nil {
		t.Fatal(err)
	}
	formRequest := recorder.snapshot()[2]
	if !strings.Contains(formRequest.Body, "ap0=%7B%22id%22%3A1%7D") || !strings.Contains(formRequest.Body, "space=a+b") {
		t.Fatalf("form body = %q", formRequest.Body)
	}

	multipartHelper := NewWebHelper()
	multipartHelper.BodyType = BodyTypeFormData
	multipartHelper.BodyFormData.AddField("description", "demo").AddFile("file", "sample.txt", "text/plain", []byte("hello"))
	if _, err := multipartHelper.SendHttpRequest(server.URL + "/upload"); err != nil {
		t.Fatal(err)
	}
	multipartRequest := recorder.snapshot()[3]
	if !strings.Contains(multipartRequest.Header.Get("Content-Type"), "multipart/form-data") ||
		!strings.Contains(multipartRequest.Body, `name="description"`) ||
		!strings.Contains(multipartRequest.Body, `filename="sample.txt"`) ||
		!strings.Contains(multipartRequest.Body, "hello") {
		t.Fatalf("multipart request = %#v", multipartRequest)
	}

	failing := &requestRecorder{respond: func(recordedRequest) (int, string, http.Header) {
		return http.StatusBadRequest, "bad", nil
	}}
	failingServer := newTestServer(failing)
	defer failingServer.Close()
	service := NewWebHelperServices()
	response, err := service.SendHttpRequest(failingServer.URL, "{}")
	var httpErr *HTTPError
	if response != "bad" || !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("response=%q err=%v", response, err)
	}
}

func TestClientAutoLoginOperationLogout(t *testing.T) {
	recorder := &requestRecorder{respond: func(request recordedRequest) (int, string, http.Header) {
		if strings.Contains(request.Path, "AuthService.LoginByAppSecret") {
			return http.StatusOK, `{"LoginResultType":1}`, http.Header{"Set-Cookie": {"session=abc; Path=/"}}
		}
		return http.StatusOK, `{"ok":true}`, nil
	}}
	server := newTestServer(recorder)
	defer server.Close()
	client, err := NewClient(
		WithAppSettings(testSettings(server.URL+"/k3cloud/")),
		WithClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.View("TEST_Form", "{}")
	if err != nil {
		t.Fatal(err)
	}
	if response != `{"ok":true}` {
		t.Fatalf("response = %q", response)
	}
	requests := recorder.snapshot()
	if len(requests) != 3 || !strings.Contains(requests[0].Path, "LoginByAppSecret") ||
		!strings.Contains(requests[1].Path, "DynamicFormService.View") ||
		!strings.Contains(requests[2].Path, "AuthService.Logout") {
		t.Fatalf("requests = %#v", requests)
	}
	if !strings.Contains(requests[1].Header.Get("Cookie"), "session=abc") {
		t.Fatalf("operation cookie = %q", requests[1].Header.Get("Cookie"))
	}
}

func TestClientRejectedLoginStopsOperation(t *testing.T) {
	recorder := &requestRecorder{respond: func(recordedRequest) (int, string, http.Header) {
		return http.StatusOK, `{"LoginResultType":0,"Message":"denied"}`, nil
	}}
	server := newTestServer(recorder)
	defer server.Close()
	client, err := NewClient(WithAppSettings(testSettings(server.URL + "/k3cloud/")))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.View("TEST_Form", "{}")
	if response != `{"LoginResultType":0,"Message":"denied"}` || !errors.Is(err, ErrLoginFailed) {
		t.Fatalf("response=%q err=%v", response, err)
	}
	if len(recorder.snapshot()) != 1 {
		t.Fatal("operation was sent after rejected login")
	}
}

func TestAllBusinessRoutesAndEnvelopes(t *testing.T) {
	recorder := &requestRecorder{}
	server := newTestServer(recorder)
	defer server.Close()
	client, err := NewClient(
		WithAppSettings(testSettings(server.URL+"/k3cloud/")),
		WithClientLoginType(LoginTypeAPISignHeaders),
	)
	if err != nil {
		t.Fatal(err)
	}
	options := []CallOption{WithAutoLogin(false), WithAutoLogout(false)}
	const payload = `{"Id":123}`
	type formCall func(string, string, ...CallOption) (string, error)
	formCalls := map[string]formCall{
		"View": client.View, "Save": client.Save, "BatchSave": client.BatchSave,
		"Submit": client.Submit, "Audit": client.Audit, "UnAudit": client.UnAudit,
		"Delete": client.Delete, "Draft": client.Draft, "Allocate": client.Allocate,
		"Push": client.Push, "GroupSave": client.GroupSave, "FlexSave": client.FlexSave,
		"GetSysReportData": client.GetSysReportData, "CancelAllocate": client.CancelAllocate,
		"CancelAssign": client.CancelAssign, "Disassembly": client.Disassembly,
	}
	for operation, call := range formCalls {
		response, callErr := call("TEST_Form", payload, options...)
		if callErr != nil || response != `{"ok":true}` {
			t.Fatalf("%s response=%q err=%v", operation, response, callErr)
		}
		assertLastRequest(t, recorder, dynamicFormPath(operation), []any{"TEST_Form", payload}, false)
	}
	type payloadCall func(string, ...CallOption) (string, error)
	payloadCalls := map[string]payloadCall{
		"ExecuteBillQuery": client.ExecuteBillQuery, "SendMsg": client.SendMsg,
		"SwitchOrg": client.SwitchOrg, "WorkflowAudit": client.WorkflowAudit,
		"GroupDelete": client.GroupDelete, "QueryBusinessInfo": client.QueryBusinessInfo,
		"QueryGroupInfo": client.QueryGroupInfo,
	}
	for operation, call := range payloadCalls {
		response, callErr := call(payload, options...)
		if callErr != nil || response != `{"ok":true}` {
			t.Fatalf("%s response=%q err=%v", operation, response, callErr)
		}
		assertLastRequest(t, recorder, dynamicFormPath(operation), []any{payload}, false)
	}
	rawCalls := map[string]payloadCall{
		"AttachmentUpLoad":   client.AttachmentUpLoad,
		"AttachmentDownLoad": client.AttachmentDownLoad,
		"UploadFile":         client.UploadFile,
	}
	for operation, call := range rawCalls {
		if _, callErr := call(payload, options...); callErr != nil {
			t.Fatal(callErr)
		}
		assertLastRequest(t, recorder, dynamicFormPath(operation), nil, true)
	}

	if _, err := client.ExecuteOperation("TEST_Form", "Forbid", payload, options...); err != nil {
		t.Fatal(err)
	}
	assertLastRequest(t, recorder, dynamicFormPath("ExecuteOperation"), []any{"TEST_Form", "Forbid", payload}, false)
	if _, err := client.CustomBusinessService(payload, "Sample.WebApi.Service.Run,Sample.WebApi", options...); err != nil {
		t.Fatal(err)
	}
	assertLastRequest(t, recorder, "Sample.WebApi.Service.Run,Sample.WebApi.common.kdsvc", []any{payload}, false)
	stub := CustomServicesStubpath{"Sample.WebApi", "Service", "Run"}
	if _, err := client.CustomBusinessServiceByParametersAndStubpath(payload, stub, options...); err != nil {
		t.Fatal(err)
	}
	assertLastRequest(t, recorder, stub.GetCustomServicesStubpathURL(), nil, true)
}

func assertLastRequest(t *testing.T, recorder *requestRecorder, servicePath string, parameters []any, raw bool) {
	t.Helper()
	requests := recorder.snapshot()
	request := requests[len(requests)-1]
	if request.Method != http.MethodPost || request.Path != "/k3cloud/"+servicePath {
		t.Fatalf("request route = %s %s", request.Method, request.Path)
	}
	if request.Header.Get("X-Kd-Appkey") == "" {
		t.Fatal("API Sign headers were not sent")
	}
	if raw {
		if request.Body != `{"Id":123}` {
			t.Fatalf("raw body = %q", request.Body)
		}
		return
	}
	if got := envelopeParameters(t, request.Body); !equalJSONValues(got, parameters) {
		t.Fatalf("parameters = %#v, want %#v", got, parameters)
	}
}

func equalJSONValues(left, right []any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}

func TestGetDataCenterListOverride(t *testing.T) {
	recorder := &requestRecorder{}
	server := newTestServer(recorder)
	defer server.Close()
	client, err := NewClient(
		WithAppSettings(testSettings("http://unused.invalid/")),
		WithClientLoginType(LoginTypeAPISignHeaders),
	)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.GetDataCenterList(server.URL + "/k3cloud")
	if err != nil || response != `{"ok":true}` {
		t.Fatalf("response=%q err=%v", response, err)
	}
	want := "/k3cloud/Kingdee.BOS.ServiceFacade.ServicesStub.Account.AccountService.GetDataCenterList.common.kdsvc"
	if got := recorder.snapshot()[0].Path; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}
