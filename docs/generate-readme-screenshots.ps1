[CmdletBinding()]
param(
    [string]$GoExecutable = "",
    [string[]]$ExampleCommand = @(),
    [switch]$Live
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$screenshotDirectory = Join-Path $PSScriptRoot "screenshots"

$edgeCandidates = @(
    (Join-Path ${env:ProgramFiles(x86)} "Microsoft\Edge\Application\msedge.exe"),
    (Join-Path $env:ProgramFiles "Microsoft\Edge\Application\msedge.exe")
)
$edgePath = $edgeCandidates | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
if (-not $edgePath) {
    throw "找不到 Microsoft Edge。该脚本使用 Edge 无头模式把示例的实际控制台输出保存为 PNG。"
}

if ([string]::IsNullOrWhiteSpace($GoExecutable)) {
    $goCommand = Get-Command go -ErrorAction SilentlyContinue
    if (-not $goCommand) {
        throw "找不到 go 命令。请安装 Go 1.22 或更高版本，或通过 -GoExecutable 指定 go.exe。"
    }
    $GoExecutable = $goCommand.Source
}
if (-not (Test-Path -LiteralPath $GoExecutable)) {
    throw "Go 可执行文件不存在：$GoExecutable"
}

$examples = @(
    @{ Command = "module-install";          File = "00-go-modules-install.png" },
    @{ Command = "sign-sha256";             File = "01-sign-sha256.png" },
    @{ Command = "sign-sha1";               File = "02-sign-sha1.png" },
    @{ Command = "app-secret";              File = "03-app-secret.png" },
    @{ Command = "validate-login";          File = "04-validate-login.png" },
    @{ Command = "simple-passport";         File = "05-simple-passport.png" },
    @{ Command = "api-sign-headers";        File = "06-api-sign-headers.png" },
    @{ Command = "dynamic-config";          File = "07-dynamic-config.png" },
    @{ Command = "custom-config-path";      File = "08-custom-config-path.png" },
    @{ Command = "custom-webapi";           File = "09-custom-webapi.png" },
    @{ Command = "sso-v4";                  File = "10-sso-v4.png" },
    @{ Command = "upload-file";             File = "11-upload-file.png" },
    @{ Command = "upload-progress";         File = "12-upload-progress.png" },
    @{ Command = "upload-base64";           File = "13-upload-base64.png" },
    @{ Command = "validate-user-endecode";  File = "14-validate-user-endecode.png" }
)

if ($ExampleCommand.Count -gt 0) {
    $unknownCommands = @($ExampleCommand | Where-Object { $_ -notin $examples.Command })
    if ($unknownCommands.Count -gt 0) {
        throw "未知示例命令：$($unknownCommands -join ', ')"
    }
    $examples = @($examples | Where-Object { $_.Command -in $ExampleCommand })
}

New-Item -ItemType Directory -Path $screenshotDirectory -Force | Out-Null

$originalScreenshotMode = $env:YIKD_SCREENSHOT_MODE
$originalExampleMode = $env:YIKD_EXAMPLE_MODE
Push-Location $repositoryRoot
try {
    & $GoExecutable test ./...
    if ($LASTEXITCODE -ne 0) {
        throw "Go 测试失败，无法生成 README 截图。"
    }

    $env:YIKD_SCREENSHOT_MODE = "1"
    $env:YIKD_EXAMPLE_MODE = if ($Live) { "live" } else { "mock" }
    $modeLabel = if ($Live) { "live K3Cloud" } else { "local reproducible mock" }

    foreach ($example in $examples) {
        $command = [string]$example.Command
        $fileName = [string]$example.File
        Write-Host "运行并截图：$command"

        $output = (& $GoExecutable run ./examples/readme $command 2>&1 | Out-String)
        $exitCode = $LASTEXITCODE

        # 双重保护：即使示例的脱敏逻辑被意外修改，也不把真实密码写入图片。
        if (-not [string]::IsNullOrEmpty($env:YIKD_VALIDATE_PASSWORD)) {
            $output = $output.Replace($env:YIKD_VALIDATE_PASSWORD, "[REDACTED]")
        }
        if ($exitCode -ne 0) {
            throw "示例 $command 运行失败（退出码 $exitCode）。输出：`n$output"
        }

        $encodedOutput = [System.Net.WebUtility]::HtmlEncode($output)
        $encodedCommand = [System.Net.WebUtility]::HtmlEncode($command)
        $encodedMode = [System.Net.WebUtility]::HtmlEncode($modeLabel)
        $html = @"
<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<title>YiKdWebClient-Go - $encodedCommand</title>
<style>
  * { box-sizing: border-box; }
  html, body { margin: 0; background: #0b1120; color: #d7e0ee; }
  body { padding: 18px; font-family: "Segoe UI", "Microsoft YaHei", sans-serif; }
  .window { overflow: hidden; border: 1px solid #334155; border-radius: 10px; background: #0f172a; box-shadow: 0 18px 48px rgba(0,0,0,.35); }
  .titlebar { display: flex; align-items: center; justify-content: space-between; min-height: 52px; padding: 0 20px; background: #1e293b; border-bottom: 1px solid #334155; }
  .title { font-size: 16px; font-weight: 650; color: #f8fafc; }
  .meta { font-size: 12px; color: #94a3b8; }
  pre { margin: 0; padding: 22px; white-space: pre-wrap; overflow-wrap: anywhere; tab-size: 4; font: 14px/1.55 "Cascadia Mono", "Microsoft YaHei UI", Consolas, monospace; color: #d7e0ee; }
</style>
</head>
<body>
  <section class="window">
    <header class="titlebar">
      <div class="title">YiKdWebClient-Go · $encodedCommand · 实际运行输出</div>
      <div class="meta">go run · $encodedMode</div>
    </header>
    <pre>$encodedOutput</pre>
  </section>
</body>
</html>
"@

        $temporaryHtmlPath = Join-Path ([System.IO.Path]::GetTempPath()) ("yikd-go-" + [guid]::NewGuid().ToString("N") + ".html")
        [System.IO.File]::WriteAllText($temporaryHtmlPath, $html, [System.Text.UTF8Encoding]::new($false))

        try {
            $visualLineCount = 0
            foreach ($line in ($output -split "`r?`n")) {
                $visualLineCount += [Math]::Max(1, [Math]::Ceiling($line.Length / 132.0))
            }
            $imageHeight = [Math]::Min(10000, [Math]::Max(560, [int]($visualLineCount * 23 + 120)))
            $outputPath = Join-Path $screenshotDirectory $fileName
            $pageUri = ([Uri]$temporaryHtmlPath).AbsoluteUri

            & $edgePath --headless=new --disable-gpu --disable-sync --no-first-run --no-default-browser-check `
                --hide-scrollbars --force-device-scale-factor=1 --log-level=3 `
                "--window-size=1440,$imageHeight" "--screenshot=$outputPath" $pageUri | Out-Null
            if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $outputPath)) {
                throw "Edge 未能生成截图：$outputPath"
            }
        }
        finally {
            Remove-Item -LiteralPath $temporaryHtmlPath -Force -ErrorAction SilentlyContinue
        }
    }
}
finally {
    $env:YIKD_SCREENSHOT_MODE = $originalScreenshotMode
    $env:YIKD_EXAMPLE_MODE = $originalExampleMode
    Pop-Location
}

Write-Host "全部截图已生成：$screenshotDirectory"
