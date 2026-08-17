# Go 截图工具使用说明

本文件专门说明同目录 `generate-readme-screenshots.ps1` 的使用方法。脚本运行 `examples/readme` 场景，并使用 Microsoft Edge 无头模式把控制台输出渲染为 PNG。

## 文件与输出位置

- 生成脚本：`docs/generate-readme-screenshots.ps1`
- 示例运行器：`examples/readme`
- 输出目录：`docs/screenshots`
- 默认模式：本地可复现 mock

默认会运行 15 个场景并覆盖 `00-go-modules-install.png` 至 `14-validate-user-endecode.png`。

## 前置条件

1. Windows PowerShell 5.1 或 PowerShell 7。
2. Go 1.22 或更高版本。
3. Microsoft Edge 安装在标准 Program Files 路径。
4. 项目目录可写。

默认 mock 模式不需要金蝶地址、账号、密码或 CNF。脚本会先运行 `go test ./...`，测试失败时不会继续生成图片。

## 默认生成全部场景

在项目根目录执行：

~~~powershell
& .\docs\generate-readme-screenshots.ps1
~~~

也可以使用独立 PowerShell 进程：

~~~powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File .\docs\generate-readme-screenshots.ps1
~~~

## 只生成指定场景

~~~powershell
& .\docs\generate-readme-screenshots.ps1 `
  -ExampleCommand sign-sha256
~~~

多个场景：

~~~powershell
& .\docs\generate-readme-screenshots.ps1 `
  -ExampleCommand "sign-sha256","app-secret"
~~~

未知场景名称会直接报错。

## 指定 Go 可执行文件

脚本默认从 `PATH` 查找 `go`。也可以显式指定：

~~~powershell
& .\docs\generate-readme-screenshots.ps1 `
  -GoExecutable "C:\Go\bin\go.exe"
~~~

`-GoExecutable` 必须指向实际存在的文件。

## 真实测试环境模式

只有显式传入 `-Live` 才会使用真实配置：

~~~powershell
$env:YIKD_CONFIG_PATH = "D:\secure\appsettings.xml"
$env:YIKD_VALIDATE_PASSWORD = "请替换为测试用户的真实密码"
$env:YIKD_CNF_PATH = "D:\secure\API测试.cnf"
$env:YIKD_UPLOAD_FILE = "D:\test-data\upload-demo.txt"

& .\docs\generate-readme-screenshots.ps1 -Live

Remove-Item Env:\YIKD_CONFIG_PATH -ErrorAction SilentlyContinue
Remove-Item Env:\YIKD_VALIDATE_PASSWORD -ErrorAction SilentlyContinue
Remove-Item Env:\YIKD_CNF_PATH -ErrorAction SilentlyContinue
Remove-Item Env:\YIKD_UPLOAD_FILE -ErrorAction SilentlyContinue
~~~

真实模式可使用的主要变量：

| 环境变量 | 用途 |
| --- | --- |
| `YIKD_CONFIG_PATH` | XML 配置路径；未设置时使用客户端默认路径 |
| `YIKD_VALIDATE_DBID` | 旧版登录数据中心 ID；未设置时读取 XML |
| `YIKD_VALIDATE_USERNAME` | 旧版登录用户名；未设置时读取 XML |
| `YIKD_VALIDATE_PASSWORD` | `validate-login` 和 `validate-user-endecode` 的密码 |
| `YIKD_VALIDATE_LCID` | 旧版登录语系；未设置时读取 XML |
| `YIKD_CNF_PATH` | `simple-passport` 使用的真实 CNF 文件 |
| `YIKD_UPLOAD_FILE` | 上传场景的本地文件 |
| `YIKD_UPLOAD_FORM_ID` | 上传目标表单；默认 `SAL_SaleOrder` |
| `YIKD_UPLOAD_INTER_ID` | 上传目标单据内码 |
| `YIKD_UPLOAD_BILL_NO` | 上传目标单据编号 |
| `YIKD_UPLOAD_CHUNK_SIZE` | 上传分块字节数，必须大于 0 |

> 真实模式可能读取或写入目标业务系统。上传场景只能对专用测试账套和测试单据执行。

## 参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-GoExecutable` | 从 `PATH` 查找 | Go 可执行文件完整路径 |
| `-ExampleCommand` | 全部场景 | 只生成一个或多个场景 |
| `-Live` | 关闭 | 使用真实测试环境；未指定时始终使用本地 mock |

## 支持的场景

| 命令 | 输出文件 |
| --- | --- |
| `module-install` | `00-go-modules-install.png` |
| `sign-sha256` | `01-sign-sha256.png` |
| `sign-sha1` | `02-sign-sha1.png` |
| `app-secret` | `03-app-secret.png` |
| `validate-login` | `04-validate-login.png` |
| `simple-passport` | `05-simple-passport.png` |
| `api-sign-headers` | `06-api-sign-headers.png` |
| `dynamic-config` | `07-dynamic-config.png` |
| `custom-config-path` | `08-custom-config-path.png` |
| `custom-webapi` | `09-custom-webapi.png` |
| `sso-v4` | `10-sso-v4.png` |
| `upload-file` | `11-upload-file.png` |
| `upload-progress` | `12-upload-progress.png` |
| `upload-base64` | `13-upload-base64.png` |
| `validate-user-endecode` | `14-validate-user-endecode.png` |

## 执行与清理行为

1. 运行全部 Go 测试。
2. 设置 `YIKD_SCREENSHOT_MODE=1` 和相应示例模式。
3. 逐个执行 `go run ./examples/readme <命令>`。
4. 再次替换密码，检查场景退出码。
5. 将控制台内容写入临时 HTML，并由 Edge 生成 PNG。
6. 删除临时 HTML，恢复脚本进入前的进程环境变量。

mock 模式中的临时 HTTP 服务、XML、CNF 和上传文件由示例运行器创建并清理。端口、时间戳、nonce 和签名每次运行都可能不同。

## 失败排查

- 找不到 Go：将 `go` 加入 `PATH`，或使用 `-GoExecutable`。
- Go 测试失败：先运行 `go test ./...` 查看完整错误。
- live 模式缺少密码、CNF 或文件：只为对应场景设置所需变量。
- Edge 未生成文件：确认 Edge 位于标准路径，并检查 `docs/screenshots` 写权限。
- 场景返回业务失败：检查配置、权限、表单、单据和目标服务端能力。

生成后应人工检查图片和 `git diff`，再决定是否提交。
