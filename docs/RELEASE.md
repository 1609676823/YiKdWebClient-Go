# Go Module 发布说明

## 1. 版本规则

Go Module 使用语义化版本：

```text
v主版本.次版本.修订版本
```

例如 `v1.0.0`、`v1.0.1`、`v1.1.0`。C# 的 `1.0.0.32` 四段版本不能直接用作稳定 Go Module Tag；C# 基准版本记录在 `CSharpSourceVersion` 和更新日志中。

从 `v2.0.0` 开始，Go 要求 `go.mod` 模块路径增加 `/v2`，导入路径也会随之变化。发布 v2 前必须同步修改模块路径、README、示例和测试。

## 2. 发布前检查

```powershell
.\build-release.bat
git status --short
```

确认：

- `gofmt` 无差异；
- `go vet ./...` 通过；
- `go test -cover ./...` 通过；
- `CHANGELOG.md` 已更新；
- 示例与配置中没有真实密钥；
- 工作区只包含计划发布的改动。

## 3. 提交和 Tag

```bash
git add .
git commit -m "release: v1.0.0"
git tag -a v1.0.0 -m "YiKdWebClient-Go v1.0.0"
```

## 4. 同时推送两个仓库

仓库的 `origin` 配置了两个 push URL：

```bash
git remote get-url --push --all origin
```

输出应包含：

```text
https://gitee.com/lnsyzjw/yi-kd-web-client-go.git
https://github.com/1609676823/YiKdWebClient-Go.git
```

推送分支和 Tag：

```bash
git push
git push origin v1.0.0
```

需要重新配置时运行：

```powershell
.\scripts\configure-remotes.bat
```

## 5. 验证模块

在一个空目录中验证指定版本：

```bash
go mod init example.com/yikd-smoke-test
go get gitee.com/lnsyzjw/yi-kd-web-client-go@v1.0.0
go list -m all
```

Go 的公开模块代理和文档站会根据公开仓库 Tag 获取源码，不需要上传单独的客户端包文件。

## 6. Release 页面与资源文件

对 Go 工具链而言，已推送的语义化 Tag 是发布必需条件；Gitee/GitHub 的 Release 页面是面向人类用户的可选发布说明。`go get` 不会下载 Release Assets，因此无需上传 DLL、`.so` 或预编译的 Go 库。

建议 Release 只包含：

- 与 Tag 一致的发布说明；
- 自动生成的源码 zip/tar.gz；
- 如果手工提供额外工具二进制，同时提供平台、架构和校验和说明。当前客户端库不需要该类工具二进制。

不得进入公开 Release 的运行时资源：

- 真实 `appsettings.xml`；
- CNF 文件或 CNF Base64；
- 应用密钥、用户密码、Cookie 和内网地址；
- 待上传的业务文件；
- 自定义 WebAPI 服务端 DLL 及金蝶运行时 DLL。它们属于目标 K3Cloud 环境的独立部署流程。

各仓库文件职责、XML/CNF/附件的部署方式和自定义 WebAPI DLL 说明见根目录 [`README.md`](../README.md#11-发布release-资源与仓库文件职责)。
