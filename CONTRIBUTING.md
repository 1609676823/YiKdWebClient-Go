# 参与贡献

本项目以 C# `YiKdWebClient` 1.0.0.32 为行为基准，Java 与 Python 移植版用于交叉核对。修改公开 API、认证算法、JSON 报文或服务路径时，请同步修改测试和 `docs/API_MAPPING.md`。

本地验证：

```powershell
$goFiles = Get-ChildItem -Recurse -Filter *.go | ForEach-Object FullName
gofmt -w $goFiles
go vet ./...
go test -cover ./...
```

也可以在 Windows PowerShell 中运行：

```powershell
.\build-release.bat
```

提交前请确认示例配置中没有真实账套、应用密钥、用户名、CNF 文件或内网地址。
