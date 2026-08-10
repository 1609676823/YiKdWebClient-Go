package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	yikdwebclient "gitee.com/lnsyzjw/yi-kd-web-client-go"
)

func printSection(title string) {
	fmt.Println("=== " + title + " ===")
}

func printClientResult(client *yikdwebclient.YiK3CloudClient, response string, secrets []string) {
	if client.LoginType == yikdwebclient.LoginTypeAPISignHeaders || client.ReturnLoginWebModel.RequestUrl != "" {
		printWebModel("登录请求", client.ReturnLoginWebModel, secrets)
	}
	if strings.TrimSpace(client.RequestHeadersString) != "" {
		printSection("请求签名头")
		fmt.Println(sanitize(client.RequestHeadersString, secrets...))
	}
	printWebModel("业务请求", client.ReturnOperationWebModel, secrets)
	printSection("业务方法返回值")
	fmt.Println(formatJSON(sanitize(response, secrets...)))
}

func printWebModel(title string, model *yikdwebclient.RequestWebModel, secrets []string) {
	if model == nil || (model.RequestUrl == "" && model.RealRequestBody == "" && model.RealResponseBody == "") {
		return
	}
	printSection(title)
	if model.RequestUrl != "" {
		fmt.Println("URL:", sanitize(model.RequestUrl, secrets...))
	}
	if model.RealRequestBody != "" {
		fmt.Println("Body:", formatJSON(sanitize(model.RealRequestBody, secrets...)))
	}
	if model.RealResponseBody != "" {
		fmt.Println("Response:", formatJSON(sanitize(model.RealResponseBody, secrets...)))
	}
	fmt.Println()
}

func printHeaders(headers http.Header, secrets []string) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, value := range headers.Values(key) {
			fmt.Printf("%s: %s\n", key, sanitize(value, secrets...))
		}
	}
}

func formatJSON(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return compactForScreenshot(value)
	}
	indent, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return compactForScreenshot(value)
	}
	return compactForScreenshot(string(indent))
}

func sanitize(value string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[REDACTED]")
		}
	}
	return value
}

func compactForScreenshot(value string) string {
	if os.Getenv("YIKD_SCREENSHOT_MODE") != "1" {
		return value
	}
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	const maxLineLength = 220
	for index, line := range lines {
		if len([]rune(line)) <= maxLineLength {
			continue
		}
		runes := []rune(line)
		lines[index] = string(runes[:maxLineLength]) + " ... [已折叠]"
	}
	const maxLines = 38
	if len(lines) > maxLines {
		remaining := len(lines) - maxLines
		lines = append(lines[:maxLines], fmt.Sprintf("... [已折叠 %d 行]", remaining))
	}
	return strings.Join(lines, "\n")
}
