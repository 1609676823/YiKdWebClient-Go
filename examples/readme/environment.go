package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	yikdwebclient "gitee.com/lnsyzjw/yi-kd-web-client-go"
)

type exampleEnvironment struct {
	live       bool
	server     *httptest.Server
	tempDir    string
	configPath string
	cnfPath    string
	uploadPath string
	settings   *yikdwebclient.AppSettingsModel
	password   string
	secrets    []string
}

func newExampleEnvironment() (*exampleEnvironment, error) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("YIKD_EXAMPLE_MODE")), "live") {
		return newLiveEnvironment()
	}
	return newMockEnvironment()
}

func newLiveEnvironment() (*exampleEnvironment, error) {
	configPath := envOr("YIKD_CONFIG_PATH", yikdwebclient.DefaultConfigPath())
	settings, err := yikdwebclient.LoadAppSettings(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取真实配置: %w", err)
	}
	password := os.Getenv("YIKD_VALIDATE_PASSWORD")
	return &exampleEnvironment{
		live:       true,
		configPath: configPath,
		cnfPath:    os.Getenv("YIKD_CNF_PATH"),
		uploadPath: os.Getenv("YIKD_UPLOAD_FILE"),
		settings:   settings,
		password:   password,
		secrets:    nonEmpty(settings.XKDApiAppSec, password),
	}, nil
}

func newMockEnvironment() (*exampleEnvironment, error) {
	environment := &exampleEnvironment{password: "mock-password"}
	var uploadNumber atomic.Int64
	environment.server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		path := request.URL.Path
		switch {
		case strings.Contains(path, "AuthService.Logout"):
			_, _ = response.Write([]byte(`{"LogoutResultType":1}`))
		case strings.Contains(path, "AuthService."):
			http.SetCookie(response, &http.Cookie{Name: "ASP.NET_SessionId", Value: "readme-mock-session", Path: "/"})
			_, _ = response.Write([]byte(`{"LoginResultType":1,"Context":{"UserName":"Administrator"},"Message":"local mock login accepted"}`))
		case strings.Contains(path, "AttachmentUpLoad"):
			fileID := uploadNumber.Add(1)
			_, _ = fmt.Fprintf(response, `{"Result":{"ResponseStatus":{"IsSuccess":true},"FileId":"mock-file-%d"}}`, fileID)
		case strings.Contains(path, "GlobalServiceCustom.WebApi"):
			_, _ = response.Write([]byte(`{"Result":{"ResponseStatus":{"IsSuccess":true},"Rows":[{"FID":1001,"FNUMBER":"Administrator"}]}}`))
		default:
			_, _ = response.Write([]byte(`{"Result":{"ResponseStatus":{"IsSuccess":true},"Result":{"Id":1001,"Number":"Administrator","Name":"Administrator"}}}`))
		}
	}))
	tempDir, err := os.MkdirTemp("", "yikd-readme-")
	if err != nil {
		environment.server.Close()
		return nil, err
	}
	environment.tempDir = tempDir
	environment.settings = &yikdwebclient.AppSettingsModel{
		XKDApiAcctID:    "mock-account-id",
		XKDApiAppID:     "readme-client_frperg",
		XKDApiAppSec:    "mock-app-secret",
		XKDApiUserName:  "Administrator",
		XKDApiLCID:      "2052",
		XKDApiServerUrl: environment.server.URL + "/k3cloud/",
		XKDApiOrgNum:    "100",
	}
	environment.secrets = nonEmpty(environment.settings.XKDApiAppSec, environment.password)
	environment.configPath = filepath.Join(tempDir, "appsettings.mock.xml")
	environment.cnfPath = filepath.Join(tempDir, "API测试.cnf.example")
	environment.uploadPath = filepath.Join(tempDir, "upload-demo.txt")
	if err := os.WriteFile(environment.configPath, []byte(mockConfigXML(environment.settings)), 0o600); err != nil {
		environment.close()
		return nil, err
	}
	if err := os.WriteFile(environment.cnfPath, []byte("README mock CNF; do not use in production"), 0o600); err != nil {
		environment.close()
		return nil, err
	}
	uploadContent := "YiKdWebClient-Go README attachment upload demo.\n" +
		"Each request is sent to the local reproducible mock server.\n"
	if err := os.WriteFile(environment.uploadPath, []byte(uploadContent), 0o600); err != nil {
		environment.close()
		return nil, err
	}
	return environment, nil
}

func (environment *exampleEnvironment) close() {
	if environment == nil {
		return
	}
	if environment.server != nil {
		environment.server.Close()
	}
	if environment.tempDir != "" {
		_ = os.RemoveAll(environment.tempDir)
	}
}

func (environment *exampleEnvironment) modeLabel() string {
	if environment.live {
		return "真实 K3Cloud 环境"
	}
	return "本地可复现 mock 服务（无真实凭据）"
}

func (environment *exampleEnvironment) validateSettings(encoded bool) (*yikdwebclient.ValidateLoginSettingsModel, error) {
	password := environment.password
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("当前示例需要 YIKD_VALIDATE_PASSWORD")
	}
	if encoded {
		value, err := yikdwebclient.Encode(password)
		if err != nil {
			return nil, err
		}
		environment.secrets = append(environment.secrets, value)
		password = value
	}
	userName := envOr("YIKD_VALIDATE_USERNAME", environment.settings.XKDApiUserName)
	dbID := envOr("YIKD_VALIDATE_DBID", environment.settings.XKDApiAcctID)
	lcid, err := strconv.Atoi(envOr("YIKD_VALIDATE_LCID", environment.settings.XKDApiLCID))
	if err != nil {
		return nil, fmt.Errorf("YIKD_VALIDATE_LCID 必须是数字: %w", err)
	}
	return &yikdwebclient.ValidateLoginSettingsModel{
		Url: environment.settings.XKDApiServerUrl, DbId: dbID,
		UserName: userName, Password: password, Lcid: lcid,
	}, nil
}

func (environment *exampleEnvironment) requireCNFPath() (string, error) {
	if strings.TrimSpace(environment.cnfPath) == "" {
		return "", fmt.Errorf("当前示例需要 YIKD_CNF_PATH")
	}
	return environment.cnfPath, nil
}

func (environment *exampleEnvironment) requireUploadPath() (string, error) {
	if strings.TrimSpace(environment.uploadPath) == "" {
		return "", fmt.Errorf("当前示例需要 YIKD_UPLOAD_FILE")
	}
	return environment.uploadPath, nil
}

func mockConfigXML(settings *yikdwebclient.AppSettingsModel) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <appSettings>
    <add key="X-KDApi-AcctID" value="%s" />
    <add key="X-KDApi-AppID" value="%s" />
    <add key="X-KDApi-AppSec" value="%s" />
    <add key="X-KDApi-UserName" value="%s" />
    <add key="X-KDApi-LCID" value="%s" />
    <add key="X-KDApi-ServerUrl" value="%s" />
    <add key="X-KDApi-OrgNum" value="%s" />
  </appSettings>
</configuration>`, settings.XKDApiAcctID, settings.XKDApiAppID, settings.XKDApiAppSec,
		settings.XKDApiUserName, settings.XKDApiLCID, settings.XKDApiServerUrl, settings.XKDApiOrgNum)
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
