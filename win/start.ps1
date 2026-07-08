$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$PackageDir = Join-Path $PSScriptRoot "package"
$Port = if ($env:EXCELPER_PORT) { $env:EXCELPER_PORT } else { "3012" }
$Url = "http://127.0.0.1:$Port"

if (-not $env:EXCELPER_PORT) {
    $env:EXCELPER_PORT = $Port
}
if (-not $env:EXCELPER_DATA_DIR) {
    $env:EXCELPER_DATA_DIR = Join-Path $PackageDir "data"
}
if (-not (Test-Path $env:EXCELPER_DATA_DIR)) {
    New-Item -ItemType Directory -Path $env:EXCELPER_DATA_DIR | Out-Null
}

$Exe = Join-Path $PackageDir "Excelper.exe"
if (Test-Path $Exe) {
    Start-Process $Exe
    Start-Sleep -Seconds 2
    Start-Process $Url
    return
}

$GoCommand = Get-Command go -ErrorAction SilentlyContinue
if ($GoCommand) {
    $GoCommand = $GoCommand.Source
} elseif (Test-Path "C:\Program Files\Go\bin\go.exe") {
    $GoCommand = "C:\Program Files\Go\bin\go.exe"
} else {
    throw "未找到 Go，也没有已打包的 $Exe。请先运行 win\build.ps1 生成 exe，或安装 Go 后再运行本脚本。"
}

$FrontendIndex = Join-Path $Root "frontend\dist\index.html"
if (-not (Test-Path $FrontendIndex)) {
    if (-not (Get-Command npm -ErrorAction SilentlyContinue)) {
        throw "未找到前端构建产物，也未找到 npm。请安装 Node/npm 后运行 win\build.ps1，或先生成 frontend\dist。"
    }
    Push-Location (Join-Path $Root "frontend")
    try {
        npm install
        npm run build
    }
    finally {
        Pop-Location
    }
}

$env:EXCELPER_FRONTEND_DIR = Join-Path $Root "frontend\dist"
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$Root'; & '$GoCommand' run .\main.go"
Start-Sleep -Seconds 3
Start-Process $Url

Write-Host "Excelper 已启动：$Url" -ForegroundColor Green
Write-Host "数据目录：$env:EXCELPER_DATA_DIR" -ForegroundColor Cyan
