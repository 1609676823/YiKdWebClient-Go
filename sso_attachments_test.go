package yikdwebclient

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func decodeSSOArguments(t *testing.T, helper *SSOHelper, value string) map[string]any {
	t.Helper()
	parts := strings.SplitN(value, "?ud=", 2)
	if len(parts) != 2 {
		t.Fatalf("SSO URL has no ud: %q", value)
	}
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != helper.ArgJSON {
		t.Fatalf("decoded payload != ArgJSON\n%s\n%s", decoded, helper.ArgJSON)
	}
	var result map[string]any
	if err := json.Unmarshal(decoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSSOVersions(t *testing.T) {
	settings := testSettings("https://configured.test/k3cloud/")
	helper := NewSSOHelper(settings)
	helper.PermitCount = "1"
	v4, err := helper.GetSSOURLsV4("v4-user", "https://override.test/k3cloud")
	if err != nil {
		t.Fatal(err)
	}
	args := decodeSSOArguments(t, helper, v4.HTML5URL)
	values := []string{"account-id", "v4-user", "app-id", "app-secret", args["timestamp"].(string), "1"}
	if args["signeddata"] != Sha256Hex(sortedJoin(values)) || args["otherargs"] != "|{'permitcount':'1'}" {
		t.Fatalf("V4 args = %#v", args)
	}
	if !strings.HasPrefix(v4.HTML5URL, "https://override.test/k3cloud/") || !strings.Contains(v4.WPFURL, "override.test:443") {
		t.Fatalf("V4 URLs = %#v", v4)
	}

	for _, version := range []int{3, 2} {
		helper = NewSSOHelper(settings)
		username := "v" + string(rune('0'+version)) + "-user"
		var urls *SSOLoginURLObject
		var callErr error
		if version == 3 {
			urls, callErr = helper.GetSSOURLsV3(username, "https://override.test/k3cloud/")
		} else {
			urls, callErr = helper.GetSSOURLsV2(username, "https://override.test/k3cloud/")
		}
		if callErr != nil {
			t.Fatal(callErr)
		}
		actual := decodeSSOArguments(t, helper, urls.HTML5URL)
		values = []string{"account-id", username, "app-id", "app-secret", actual["timestamp"].(string)}
		if actual["signeddata"] != GetSignatureSHA1Util(values) {
			t.Fatalf("V%d signeddata = %v", version, actual["signeddata"])
		}
	}

	helper = NewSSOHelper(settings)
	v1, err := helper.GetSSOURLsV1("v1-user", "https://override.test/k3cloud/")
	if err != nil {
		t.Fatal(err)
	}
	encoded := strings.SplitN(v1.HTML5URL, "?ud=", 2)[1]
	payload, _ := base64.StdEncoding.DecodeString(encoded)
	pipeParts := strings.Split(string(payload), "|")
	if len(pipeParts) != 7 || pipeParts[1] != "account-id" || pipeParts[2] != "v1-user" {
		t.Fatalf("V1 payload = %q", payload)
	}
	if want := GetSignatureSHA1Util([]string{"account-id", "v1-user", "app-id", "app-secret", pipeParts[5]}); pipeParts[4] != want {
		t.Fatalf("V1 signature = %q, want %q", pipeParts[4], want)
	}
}

func TestSSOLogoutPayloadAndExecution(t *testing.T) {
	settings := testSettings("https://configured.test/k3cloud/")
	for version, call := range map[int]func(*SSOHelper) (*SSOLogoutObject, error){
		4: func(helper *SSOHelper) (*SSOLogoutObject, error) {
			return helper.GetSSOLogoutAP0V4("logout-user", "https://override.test/k3cloud/")
		},
		3: func(helper *SSOHelper) (*SSOLogoutObject, error) {
			return helper.GetSSOLogoutAP0V3("logout-user", "https://override.test/k3cloud/")
		},
		2: func(helper *SSOHelper) (*SSOLogoutObject, error) {
			return helper.GetSSOLogoutAP0V2V1("logout-user", "https://override.test/k3cloud/")
		},
	} {
		logout, err := call(NewSSOHelper(settings))
		if err != nil {
			t.Fatal(err)
		}
		var root struct {
			SignedData string `json:"SignedData"`
			Timestamp  int64  `json:"Timestamp"`
		}
		if err := json.Unmarshal([]byte(logout.AP0), &root); err != nil {
			t.Fatal(err)
		}
		values := []string{"account-id", "logout-user", "app-id", "app-secret", jsonNumber(root.Timestamp)}
		want := GetSignatureSHA1Util(values)
		if version == 4 {
			want = Sha256Hex(sortedJoin(values))
		}
		if root.SignedData != want || !strings.HasPrefix(logout.RequestLogoutURL, "https://override.test") {
			t.Fatalf("V%d logout = %#v", version, logout)
		}
	}

	recorder := &requestRecorder{respond: func(recordedRequest) (int, string, http.Header) {
		return http.StatusOK, "logout-ok", nil
	}}
	server := newTestServer(recorder)
	defer server.Close()
	helper := NewSSOHelper(settings)
	response, err := helper.SSOExecuteLogout(SSOLogoutObject{
		RequestLogoutURL: server.URL + "/logout", AP0: `{"AcctID":"db","Username":"user"}`,
	})
	if err != nil || response != "logout-ok" {
		t.Fatalf("response=%q err=%v", response, err)
	}
	request := recorder.snapshot()[0]
	values, _ := url.ParseQuery(request.Body)
	if values.Get("ap0") != `{"AcctID":"db","Username":"user"}` ||
		!strings.Contains(request.Header.Get("Content-Type"), MediaTypeApplicationFormURLEncoded) {
		t.Fatalf("logout request = %#v", request)
	}
}

func jsonNumber(value int64) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func TestAttachmentChunkingAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.bin")
	if err := os.WriteFile(path, []byte{0, 1, 2, 3, 4}, 0o600); err != nil {
		t.Fatal(err)
	}
	var chunks []*FileChunk
	if err := ReadFileInChunksByAction(path, func(chunk *FileChunk) error {
		copyOfChunk := *chunk
		copyOfChunk.Chunkbyte = append([]byte(nil), chunk.Chunkbyte...)
		chunks = append(chunks, &copyOfChunk)
		return nil
	}, 2); err != nil {
		t.Fatal(err)
	}
	if got := []int64{chunks[0].Chunkindex, chunks[1].Chunkindex, chunks[2].Chunkindex}; !reflect.DeepEqual(got, []int64{0, 1, 2}) {
		t.Fatalf("chunk indices = %#v", got)
	}
	if !reflect.DeepEqual(chunks[0].Chunkbyte, []byte{0, 1}) || !chunks[2].IsLast || chunks[0].IsLast {
		t.Fatalf("chunks = %#v", chunks)
	}
	var base64Chunks []*FileChunk
	if err := ReadBase64ChunksByAction(base64.StdEncoding.EncodeToString([]byte{0, 1, 2, 3, 4}), "sample.bin", func(chunk *FileChunk) error {
		base64Chunks = append(base64Chunks, chunk)
		return nil
	}, 2); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(chunks[2].Chunkbyte, base64Chunks[2].Chunkbyte) || base64Chunks[0].ChunkBase64 != "AAE=" {
		t.Fatalf("base64 chunks = %#v", base64Chunks)
	}
	if err := ReadBase64ChunksByAction("AQ==", "x", func(*FileChunk) error { return nil }, 0); err == nil {
		t.Fatal("zero chunk size was accepted")
	}

	valid := &UploadModel{Data: UploadModelData{
		FileName: "sample.bin", FormId: "TEST_Form", InterId: "100", EntryinterId: "-1",
		FileId: "file-id", SendByte: "AQ==",
	}}
	if err := CheckUploadModelData(valid); err != nil {
		t.Fatal(err)
	}
	valid.Data.Entrykey = "FEntity"
	valid.Data.EntryinterId = ""
	if err := CheckUploadModelData(valid); err == nil || !strings.Contains(err.Error(), "Entrykey") {
		t.Fatalf("entry validation error = %v", err)
	}
}

func TestAttachmentUpload(t *testing.T) {
	var uploadCalls atomic.Int64
	recorder := &requestRecorder{respond: func(request recordedRequest) (int, string, http.Header) {
		if strings.Contains(request.Path, "AttachmentUpLoad") {
			call := uploadCalls.Add(1)
			body, _ := json.Marshal(map[string]any{
				"Result": map[string]any{
					"ResponseStatus": map[string]any{"IsSuccess": true},
					"FileId":         "file-" + jsonNumber(call),
				},
			})
			return http.StatusOK, string(body), nil
		}
		return http.StatusOK, `{"ok":true}`, nil
	}}
	server := newTestServer(recorder)
	defer server.Close()
	client, err := NewClient(
		WithAppSettings(testSettings(server.URL+"/k3cloud/")),
		WithClientLoginType(LoginTypeAPISignHeaders),
	)
	if err != nil {
		t.Fatal(err)
	}
	template := &UploadModel{Data: UploadModelData{FormId: "TEST_Form", InterId: "100", EntryinterId: "-1"}}
	var progress int
	response, err := AttachmentUploadByBase64(
		base64.StdEncoding.EncodeToString([]byte{0, 1, 2, 3, 4}), "sample.bin",
		client, template, 2,
		func(chunk *FileChunk, _ *YiK3CloudClient) error {
			progress++
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response, `"FileId":"file-3"`) || template.Data.FileId != "file-3" || progress != 3 {
		t.Fatalf("response=%s template=%#v progress=%d", response, template, progress)
	}
	requests := recorder.snapshot()
	var logoutCount int
	for _, request := range requests {
		if strings.Contains(request.Path, "AuthService.Logout") {
			logoutCount++
		}
	}
	if uploadCalls.Load() != 3 || logoutCount != 3 {
		t.Fatalf("upload calls=%d logout calls=%d", uploadCalls.Load(), logoutCount)
	}
}

func TestAttachmentUploadFailurePreservesResponse(t *testing.T) {
	failed := `{"Result":{"ResponseStatus":{"IsSuccess":false},"FileId":""}}`
	recorder := &requestRecorder{respond: func(request recordedRequest) (int, string, http.Header) {
		if strings.Contains(request.Path, "AttachmentUpLoad") {
			return http.StatusOK, failed, nil
		}
		return http.StatusOK, `{"ok":true}`, nil
	}}
	server := newTestServer(recorder)
	defer server.Close()
	client, _ := NewClient(
		WithAppSettings(testSettings(server.URL+"/k3cloud/")),
		WithClientLoginType(LoginTypeAPISignHeaders),
	)
	response, err := AttachmentUploadByBase64(
		"AQ==", "sample.bin", client,
		&UploadModel{Data: UploadModelData{FormId: "F", InterId: "1"}}, 2, nil,
	)
	var uploadErr *UploadResponseError
	if response != failed || !errors.As(err, &uploadErr) {
		t.Fatalf("response=%q err=%v", response, err)
	}
}
