package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	yikdwebclient "gitee.com/lnsyzjw/yi-kd-web-client-go"
)

const viewPayload = `{"IsUserModelInit":"true","Number":"Administrator","IsSortBySeq":"false"}`

func runModuleInstall(*exampleEnvironment) error {
	printSection("当前环境")
	fmt.Println("Go:", runtime.Version())
	fmt.Println("Module: gitee.com/lnsyzjw/yi-kd-web-client-go")
	fmt.Println()
	printSection("用户项目中的安装命令")
	fmt.Println("go mod init example.com/my-k3cloud-app    # 仅新项目执行一次")
	fmt.Println("go get gitee.com/lnsyzjw/yi-kd-web-client-go@latest")
	fmt.Println("go mod tidy")
	fmt.Println()
	printSection("代码中引用")
	fmt.Println(`import yikdwebclient "gitee.com/lnsyzjw/yi-kd-web-client-go"`)
	fmt.Println()
	fmt.Println("说明: 本项目使用 Go Modules 和语义化 Git Tag 发布版本。")
	return nil
}

func runSignSHA256(environment *exampleEnvironment) error {
	return runView(environment, yikdwebclient.LoginTypeSignSHA256, nil)
}

func runSignSHA1(environment *exampleEnvironment) error {
	return runView(environment, yikdwebclient.LoginTypeSignSHA1, nil)
}

func runAppSecret(environment *exampleEnvironment) error {
	return runView(environment, yikdwebclient.LoginTypeAppSecret, nil)
}

func runValidateLogin(environment *exampleEnvironment) error {
	settings, err := environment.validateSettings(false)
	if err != nil {
		return err
	}
	return runView(environment, yikdwebclient.LoginTypeValidateLogin, func(client *yikdwebclient.YiK3CloudClient) {
		client.ValidateLoginSettingsModel = settings
	})
}

func runValidateUserEnDeCode(environment *exampleEnvironment) error {
	settings, err := environment.validateSettings(true)
	if err != nil {
		return err
	}
	return runView(environment, yikdwebclient.LoginTypeValidateUserCode, func(client *yikdwebclient.YiK3CloudClient) {
		client.ValidateLoginSettingsModel = settings
	})
}

func runSimplePassport(environment *exampleEnvironment) error {
	cnfPath, err := environment.requireCNFPath()
	if err != nil {
		return err
	}
	if data, readErr := os.ReadFile(cnfPath); readErr == nil {
		environment.secrets = append(environment.secrets, base64.StdEncoding.EncodeToString(data))
	}
	return runView(environment, yikdwebclient.LoginTypeSimplePassport, func(client *yikdwebclient.YiK3CloudClient) {
		client.LoginBySimplePassportModel = &yikdwebclient.LoginBySimplePassportModel{
			Url: environment.settings.XKDApiServerUrl, CnfFilePath: cnfPath,
			Lcid: 2052, BySimplePassportType: yikdwebclient.SimplePassportCnfFile,
		}
	})
}

func runAPISignHeaders(environment *exampleEnvironment) error {
	return runView(environment, yikdwebclient.LoginTypeAPISignHeaders, nil)
}

func runDynamicConfig(environment *exampleEnvironment) error {
	settings := *environment.settings
	printSection("运行时组装的配置")
	fmt.Println("X-KDApi-AcctID:", settings.XKDApiAcctID)
	fmt.Println("X-KDApi-AppID:", settings.XKDApiAppID)
	fmt.Println("X-KDApi-AppSec: [REDACTED]")
	fmt.Println("X-KDApi-ServerUrl:", settings.XKDApiServerUrl)
	fmt.Println()
	client, err := yikdwebclient.NewClient(
		yikdwebclient.WithAppSettings(&settings),
		yikdwebclient.WithClientLoginType(yikdwebclient.LoginTypeSignSHA256),
	)
	if err != nil {
		return err
	}
	defer client.Close()
	response, err := client.View("SEC_User", viewPayload)
	if err != nil {
		return err
	}
	printClientResult(client, response, environment.secrets)
	return nil
}

func runCustomConfigPath(environment *exampleEnvironment) error {
	printSection("指定配置文件")
	if environment.live {
		fmt.Println(environment.configPath)
	} else {
		fmt.Println("<系统临时目录>/appsettings.mock.xml")
	}
	fmt.Println()
	client, err := yikdwebclient.NewClientFromConfig(
		environment.configPath,
		yikdwebclient.WithClientLoginType(yikdwebclient.LoginTypeSignSHA256),
	)
	if err != nil {
		return err
	}
	defer client.Close()
	response, err := client.View("SEC_User", viewPayload)
	if err != nil {
		return err
	}
	printClientResult(client, response, environment.secrets)
	return nil
}

func runCustomWebAPI(environment *exampleEnvironment) error {
	client, err := yikdwebclient.NewClient(
		yikdwebclient.WithAppSettings(environment.settings),
		yikdwebclient.WithClientLoginType(yikdwebclient.LoginTypeSignSHA256),
	)
	if err != nil {
		return err
	}
	defer client.Close()
	stub := yikdwebclient.CustomServicesStubpath{
		ProjetNamespace:   "GlobalServiceCustom.WebApi",
		ProjetClassName:   "DataServiceHandler",
		ProjetClassMethod: "CommonRunnerService",
	}
	payload := `{"parameters":["SELECT TOP 10 FID,FNUMBER FROM T_SEC_USER"]}`
	printSection("客户端生成的服务路径")
	fmt.Println(stub.GetCustomServicesStubpathURL())
	fmt.Println()
	response, err := client.CustomBusinessServiceByParametersAndStubpath(payload, stub)
	if err != nil {
		return err
	}
	printClientResult(client, response, environment.secrets)
	return nil
}

func runSSOV4(environment *exampleEnvironment) error {
	helper := yikdwebclient.NewSSOHelper(environment.settings)
	result, err := helper.GetSSOURLsV4(environment.settings.XKDApiUserName, "")
	if err != nil {
		return err
	}
	printSection("签名中间值")
	fmt.Println("Timestamp:", helper.Timestamp)
	fmt.Println("ArgJSON:", sanitize(helper.ArgJSON, environment.secrets...))
	fmt.Println("ArgJSONBase64:", helper.ArgJSONBase64)
	fmt.Println()
	printSection("SSO V4 URL")
	fmt.Println("HTML5:", result.HTML5URL)
	fmt.Println("WPF:", result.WPFURL)
	fmt.Println("Silverlight:", result.SilverlightURL)
	return nil
}

func runUploadFile(environment *exampleEnvironment) error {
	return runUpload(environment, false, false)
}

func runUploadProgress(environment *exampleEnvironment) error {
	return runUpload(environment, true, false)
}

func runUploadBase64(environment *exampleEnvironment) error {
	return runUpload(environment, true, true)
}

func runUpload(environment *exampleEnvironment, showProgress, fromBase64 bool) error {
	filePath, err := environment.requireUploadPath()
	if err != nil {
		return err
	}
	client, err := yikdwebclient.NewClient(
		yikdwebclient.WithAppSettings(environment.settings),
		yikdwebclient.WithClientLoginType(yikdwebclient.LoginTypeSignSHA256),
	)
	if err != nil {
		return err
	}
	defer client.Close()
	template := yikdwebclient.NewUploadModel()
	template.Data.FormId = envOr("YIKD_UPLOAD_FORM_ID", "SAL_SaleOrder")
	template.Data.InterId = envOr("YIKD_UPLOAD_INTER_ID", "1001")
	template.Data.BillNO = envOr("YIKD_UPLOAD_BILL_NO", "SAL000001")
	chunkSize, err := strconv.ParseInt(envOr("YIKD_UPLOAD_CHUNK_SIZE", "32"), 10, 64)
	if err != nil || chunkSize <= 0 {
		return fmt.Errorf("YIKD_UPLOAD_CHUNK_SIZE 必须是大于 0 的整数")
	}
	var uploaded int64
	progress := func(chunk *yikdwebclient.FileChunk, _ *yikdwebclient.YiK3CloudClient) error {
		uploaded += int64(len(chunk.Chunkbyte))
		if showProgress {
			fmt.Printf("分块 %d: %d 字节, IsLast=%t, 已上传=%d\n",
				chunk.Chunkindex, len(chunk.Chunkbyte), chunk.IsLast, uploaded)
		}
		return nil
	}
	printSection("上传参数")
	fmt.Println("FormId:", template.Data.FormId)
	fmt.Println("InterId:", template.Data.InterId)
	fmt.Println("BillNO:", template.Data.BillNO)
	fmt.Println("ChunkSize:", chunkSize)
	if fromBase64 {
		fmt.Println("Source: Base64 字符串")
	} else if environment.live {
		fmt.Println("Source:", filePath)
	} else {
		fmt.Println("Source: <系统临时目录>/upload-demo.txt")
	}
	fmt.Println()
	var response string
	if fromBase64 {
		data, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return readErr
		}
		response, err = yikdwebclient.AttachmentUploadByBase64(
			base64.StdEncoding.EncodeToString(data), "upload-demo.txt",
			client, template, chunkSize, progress,
		)
	} else {
		response, err = yikdwebclient.AttachmentUploadByFilePath(
			filePath, client, template, chunkSize, progress,
		)
	}
	if err != nil {
		return err
	}
	if showProgress {
		fmt.Println()
	}
	printClientResult(client, response, environment.secrets)
	return nil
}

func runView(
	environment *exampleEnvironment,
	loginType yikdwebclient.LoginType,
	configure func(*yikdwebclient.YiK3CloudClient),
) error {
	client, err := yikdwebclient.NewClient(
		yikdwebclient.WithAppSettings(environment.settings),
		yikdwebclient.WithClientLoginType(loginType),
	)
	if err != nil {
		return err
	}
	defer client.Close()
	if configure != nil {
		configure(client)
	}
	printSection("调用参数")
	fmt.Println("LoginType:", loginType)
	fmt.Println("FormId: SEC_User")
	fmt.Println("Payload:", strings.ReplaceAll(viewPayload, `\"`, `"`))
	fmt.Println()
	response, err := client.View("SEC_User", viewPayload)
	if err != nil {
		return err
	}
	printClientResult(client, response, environment.secrets)
	return nil
}
