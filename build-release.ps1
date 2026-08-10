[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath $PSScriptRoot

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go 1.22 or newer was not found in PATH.'
}

$version = (go env GOVERSION)
Write-Host "Using $version"

$goFiles = Get-ChildItem -Recurse -Filter *.go | ForEach-Object FullName
$formatDiff = gofmt -d $goFiles
if ($LASTEXITCODE -ne 0) {
    throw 'gofmt failed.'
}
if ($formatDiff) {
    $formatDiff | Write-Host
    throw 'Go source is not formatted. Run gofmt -w on the listed files.'
}

go vet ./...
if ($LASTEXITCODE -ne 0) {
    throw 'go vet failed.'
}

go test -cover ./...
if ($LASTEXITCODE -ne 0) {
    throw 'go test failed.'
}

go list -m
if ($LASTEXITCODE -ne 0) {
    throw 'go list failed.'
}

Write-Host ''
Write-Host '[SUCCESS] The Go module is ready to release.'
Write-Host 'Publish by creating and pushing a semantic-version tag, for example:'
Write-Host '  git tag -a v1.0.0 -m "YiKdWebClient-Go v1.0.0"'
Write-Host '  git push origin v1.0.0'
