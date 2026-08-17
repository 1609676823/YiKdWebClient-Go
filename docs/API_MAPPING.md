# C# / Java / Python / Go API 对照

本移植以 C# `YiKdWebClient` 1.0.0.32 为行为基准。Go 模块为：

```text
gitee.com/lnsyzjw/yi-kd-web-client-go
```

## 核心类型

| C# / Java / Python | Go |
| --- | --- |
| `YiK3CloudClient` | `YiK3CloudClient`，短别名 `Client` |
| `AppSettingsModel` | `AppSettingsModel` |
| `ValidateLoginSettingsModel` | `ValidateLoginSettingsModel` |
| `LoginBySimplePassportModel` | `LoginBySimplePassportModel` |
| `CustomServicesStubpath` | `CustomServicesStubpath` |
| `RequestWebModel` | `RequestWebModel` |
| `UploadModel` / `UploadModelData` | `UploadModel` / `UploadModelData` |
| `FileChunk` | `FileChunk` |
| `SSOHelper` | `SSOHelper` |
| `SSOLoginUrlObject` | `SSOLoginURLObject`，兼容别名 `SSOLoginUrlObject` |
| `SSOLogoutObject` | `SSOLogoutObject` |

## 认证类型

| C# 枚举成员 | Go 常量 |
| --- | --- |
| `LoginBySignSHA256` | `LoginTypeSignSHA256` |
| `LoginBySignSHA1` | `LoginTypeSignSHA1` |
| `LoginByAppSecret` | `LoginTypeAppSecret` |
| `LoginByApiSignHeaders` | `LoginTypeAPISignHeaders` |
| `ValidateLogin` | `LoginTypeValidateLogin` |
| `LoginBySimplePassport` | `LoginTypeSimplePassport` |
| `ValidateUserEnDeCode` | `LoginTypeValidateUserCode` |

认证服务类仍保留为 Go 类型：`LoginByAppSecret`、`LoginBySign`、`ValidateLogin`、`ValidateUserEnDeCode`、`LoginBySimplePassport`。静态 `LoginByApiSignHeaders` 映射为包级函数：

- `GetAPIHeaders` / `GetApiHeaders`
- `GetAPIHeadersString` / `GetApiHeadersStr`

## 业务方法

以下方法名称保持一致：

| C# 方法 | Go 方法 | 请求体形式 |
| --- | --- | --- |
| `ExecApiDynamicFormService` | `ExecAPIDynamicFormService` / `ExecApiDynamicFormService` | 标准或 `WithRawJSON(true)` |
| `ExecuteOperation` | `ExecuteOperation` | `formID, opNumber, payload` |
| `View` | `View` | FormId + payload |
| `Save` | `Save` | FormId + payload |
| `BatchSave` | `BatchSave` | FormId + payload |
| `Submit` | `Submit` | FormId + payload |
| `Audit` | `Audit` | FormId + payload |
| `UnAudit` | `UnAudit` | FormId + payload |
| `Delete` | `Delete` | FormId + payload |
| `ExecuteBillQuery` | `ExecuteBillQuery` | payload |
| `Draft` | `Draft` | FormId + payload |
| `Allocate` | `Allocate` | FormId + payload |
| `Push` | `Push` | FormId + payload |
| `GroupSave` | `GroupSave` | FormId + payload |
| `FlexSave` | `FlexSave` | FormId + payload |
| `SendMsg` | `SendMsg` | payload |
| `SwitchOrg` | `SwitchOrg` | payload |
| `WorkflowAudit` | `WorkflowAudit` | payload |
| `GetSysReportData` | `GetSysReportData` | FormId + payload |
| `AttachmentUpLoad` | `AttachmentUpLoad` | 原始 JSON |
| `AttachmentDownLoad` | `AttachmentDownLoad` | 原始 JSON |
| `UploadFile` | `UploadFile` | 原始 JSON |
| `GroupDelete` | `GroupDelete` | payload |
| `CancelAllocate` | `CancelAllocate` | FormId + payload |
| `CancelAssign` | `CancelAssign` | FormId + payload |
| `Disassembly` | `Disassembly` | FormId + payload |
| `QueryBusinessInfo` | `QueryBusinessInfo` | payload |
| `QueryGroupInfo` | `QueryGroupInfo` | payload |
| `GetDataCenterList` | `GetDataCenterList` | 无业务 JSON |

C# 的两个 `CustomBusinessService` 重载映射为：

- `CustomBusinessService(payload, apiName, options...)`
- `CustomBusinessServiceByStubpath(payload, stub, options...)`

C# 的两个 `CustomBusinessServiceByParameters` 重载映射为：

- `CustomBusinessServiceByParameters(payload, apiName, options...)`
- `CustomBusinessServiceByParametersAndStubpath(payload, stub, options...)`

## 通用工具

Go 不使用静态工具类，方法被提升为包级函数：

| C# | Go |
| --- | --- |
| `CommonFunctionHelper.CurrentTimeMillis` | `CurrentTimeMillis` |
| `CommonFunctionHelper.GetTimestamp` | `GetTimestamp` |
| `CommonFunctionHelper.GetServerUrl` | `GetServerURL` / `GetServerUrl` |
| `CommonFunctionHelper.Sha256Hex` | `Sha256Hex` / `SHA256Hex` |
| `CommonFunctionHelper.ToHexString` | `ToHexString` |
| `CommonFunctionHelper.ToBase64` | `ToBase64` |
| `CommonFunctionHelper.GetSHA1` | `GetSHA1` |
| `CommonFunctionHelper.GetSHA256` | `GetSHA256` |
| `JsonHelperServices.getRequestBodystring` | `GetRequestBodyString` |
| `JsonHelperServices.getLoginRequestBodystring` | `GetLoginRequestBodyString` |
| `JsonHelperServices.getLoginRequestBodystringByParameters` | `GetLoginRequestBodyStringByParameters` |
| `XmlConfigHelper.GetAllCfgDic` | `GetAllCfgDic` |
| `EnDecode.Encode` | `Encode` |
| `EnDecode.EncodeNew1` | `EncodeNew1` |
| `EnDecode.HmacSHA256` | `HmacSHA256` |
| `EnDecode.ByteToHexStr` | `ByteToHexStr` |
| `EnDecode.UrlEncodeWithUpperCode` | `URLEncodeWithUpperCode` / `UrlEncodeWithUpperCode` |

## 附件与 SSO

| C# | Go |
| --- | --- |
| `AttachmentHelper.ReadFileInChunksByAction` | `ReadFileInChunksByAction` |
| `AttachmentHelper.ReadBase64ChunksByAction` | `ReadBase64ChunksByAction` |
| `AttachmentHelper.AttachmentUploadByFilePath` | `AttachmentUploadByFilePath` |
| `AttachmentHelper.AttachmentUploadByBase64` | `AttachmentUploadByBase64` |
| `AttachmentHelper.CheckUploadModelData` | `CheckUploadModelData` |
| `SSOHelper.GetSsoUrlsV4` | `GetSSOURLsV4` / `GetSsoUrlsV4` |
| `SSOHelper.GetSsoUrlsV3` | `GetSSOURLsV3` / `GetSsoUrlsV3` |
| `SSOHelper.GetSsoUrlsV2` | `GetSSOURLsV2` / `GetSsoUrlsV2` |
| `SSOHelper.GetSsoUrlsV1` | `GetSSOURLsV1` / `GetSsoUrlsV1` |
| `SSOHelper.GetSSOLogoutap0StrV4` | `GetSSOLogoutAP0V4` / `GetSSOLogoutap0StrV4` |
| `SSOHelper.GetSSOLogoutap0StrV3` | `GetSSOLogoutAP0V3` / `GetSSOLogoutap0StrV3` |
| `SSOHelper.GetSSOLogoutap0StrV2V1` | `GetSSOLogoutAP0V2V1` / `GetSSOLogoutap0StrV2V1` |
| `SSOHelper.SSOExcuteLogout` | `SSOExecuteLogout` / `SSOExcuteLogout` |

## 运行时类型

| .NET / Java / Python | Go |
| --- | --- |
| `CookieContainer` / `CookieManager` / `RequestsCookieJar` | `http.CookieJar` |
| `TimeSpan` / `Duration` / 秒数 | `time.Duration` |
| `Dictionary<string,string>` / `Map` / `dict` | `http.Header`、`map[string]string`、`url.Values` |
| `Action<T>` / `Consumer<T>` / callable | `func(*FileChunk) error` |
| `IDisposable` / `AutoCloseable` / context manager | `Close() error` |
| `Task<string>` / `CompletableFuture<String>` / coroutine | 同步方法 + `context.Context`；调用方可用 goroutine |

## 有意采用的 Go 差异

- 网络、解析和验证失败使用 `error`，不把异常文本伪装成成功字符串。
- C# 可选布尔参数改为 `WithAutoLogin`、`WithAutoLogout` 和 `WithRawJSON`。
- 不复制全局单例模式；`Instance` 仅为迁移兼容，业务代码应显式持有客户端。
- Go 标准库已经提供连接池与并发执行，因此不额外提供 `Async` 后缀方法。
- `URL`、`JSON`、`API` 使用 Go 初始缩写规范；常用旧拼写只保留兼容入口。
