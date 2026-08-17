package main

import (
	"fmt"
	"os"
	"sort"
)

type readmeExample struct {
	command     string
	title       string
	description string
	run         func(*exampleEnvironment) error
}

var examples = []readmeExample{
	{"module-install", "Go Modules 安装", "展示用户项目的标准安装和导入流程", runModuleInstall},
	{"sign-sha256", "SHA256 签名登录 + View", "使用 LoginBySignSHA256 调用 View", runSignSHA256},
	{"sign-sha1", "SHA1 签名登录 + View", "使用 LoginBySignSHA1 调用 View", runSignSHA1},
	{"app-secret", "AppSecret 登录 + View", "使用 LoginByAppSecret 调用 View", runAppSecret},
	{"validate-login", "账号密码验证登录 + View", "使用 ValidateLogin 调用 View", runValidateLogin},
	{"simple-passport", "CNF 简易账号本登录 + View", "使用 LoginBySimplePassport 调用 View", runSimplePassport},
	{"api-sign-headers", "API Sign Headers + View", "无会话调用，每次请求重新签名", runAPISignHeaders},
	{"dynamic-config", "代码动态配置", "不读 XML，直接创建 AppSettingsModel", runDynamicConfig},
	{"custom-config-path", "自定义 XML 配置路径", "使用 NewClientFromConfig 读取指定文件", runCustomConfigPath},
	{"custom-webapi", "自定义 WebAPI", "调用服务端已部署的自定义 WebAPI", runCustomWebAPI},
	{"sso-v4", "SSO V4 登录 URL", "生成 HTML5、WPF 和 Silverlight 单点登录 URL", runSSOV4},
	{"upload-file", "文件分块上传", "从文件路径读取并上传附件", runUploadFile},
	{"upload-progress", "文件分块上传进度", "在每个分块成功后获得进度回调", runUploadProgress},
	{"upload-base64", "Base64 分块上传", "将 Base64 内容解码后分块上传", runUploadBase64},
	{"validate-user-endecode", "加密密码验证登录 + View", "使用 ValidateUserEnDeCode 调用 View", runValidateUserEnDeCode},
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help" {
		printUsage()
		return
	}
	command := os.Args[1]
	for _, example := range examples {
		if example.command != command {
			continue
		}
		environment, err := newExampleEnvironment()
		if err != nil {
			fatal(err)
		}
		defer environment.close()
		printBanner(example, environment)
		if err := example.run(environment); err != nil {
			fatal(err)
		}
		return
	}
	fmt.Fprintf(os.Stderr, "未知示例命令: %s\n\n", command)
	printUsage()
	os.Exit(2)
}

func printUsage() {
	fmt.Println("YiKdWebClient-Go README 可运行示例")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run ./examples/readme <command>")
	fmt.Println()
	fmt.Println("默认使用本地可复现的 mock K3Cloud 服务，不需要账号。")
	fmt.Println("设置 YIKD_EXAMPLE_MODE=live 后使用真实配置。")
	fmt.Println()
	fmt.Println("可用命令:")
	items := append([]readmeExample(nil), examples...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].command < items[j].command })
	for _, example := range items {
		fmt.Printf("  %-24s %s\n", example.command, example.description)
	}
}

func printBanner(example readmeExample, environment *exampleEnvironment) {
	fmt.Println("YiKdWebClient-Go README 示例")
	fmt.Printf("场景: %s\n", example.title)
	fmt.Printf("模式: %s\n", environment.modeLabel())
	fmt.Println()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "执行失败:", err)
	os.Exit(1)
}
