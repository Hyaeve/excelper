param(
    [string]$Configuration = "Release"
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$PackageDir = Join-Path $PSScriptRoot "package"
$FrontendDist = Join-Path $Root "frontend\dist"

function Require-Command {
    param([string]$Name)

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "$Name is required but was not found in PATH."
    }
}

function Resolve-GoCommand {
    $command = Get-Command "go" -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }
    $defaultGo = "C:\Program Files\Go\bin\go.exe"
    if (Test-Path $defaultGo) {
        return $defaultGo
    }
    throw "Go 1.22 or newer is required but was not found in PATH."
}

Require-Command "npm"
$GoCommand = Resolve-GoCommand

if (Test-Path $PackageDir) {
    Remove-Item $PackageDir -Recurse -Force
}
New-Item -ItemType Directory -Path $PackageDir | Out-Null
New-Item -ItemType Directory -Path (Join-Path $PackageDir "data") | Out-Null

Push-Location (Join-Path $Root "frontend")
try {
    npm install
    npm run build
}
finally {
    Pop-Location
}

Copy-Item $FrontendDist (Join-Path $PackageDir "frontend-dist") -Recurse -Force

Push-Location $Root
try {
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    & $GoCommand build -mod=mod -ldflags "-s -w -H=windowsgui" -o (Join-Path $PackageDir "Excelper.exe") .\main.go
}
finally {
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
    Pop-Location
}

Copy-Item (Join-Path $PSScriptRoot "assets\excelper-icon.svg") (Join-Path $PackageDir "excelper-icon.svg") -Force

Write-Host ""
Write-Host "Excelper Windows package created:" -ForegroundColor Green
Write-Host $PackageDir
Write-Host ""
Write-Host "Put .xls files into:" -ForegroundColor Cyan
Write-Host (Join-Path $PackageDir "data")
Write-Host ""
Write-Host "Start with:" -ForegroundColor Cyan
Write-Host (Join-Path $PackageDir "Excelper.exe")
