@echo off
setlocal

set ROOT=%~dp0..
set WIN_DIR=%ROOT%\win
set PACKAGE_DIR=%WIN_DIR%\package
set PORT=3012
set URL=http://127.0.0.1:%PORT%

if not defined EXCELPER_PORT set EXCELPER_PORT=%PORT%
if not defined EXCELPER_DATA_DIR set EXCELPER_DATA_DIR=%PACKAGE_DIR%\data
if not exist "%EXCELPER_DATA_DIR%" mkdir "%EXCELPER_DATA_DIR%"

if exist "%PACKAGE_DIR%\Excelper.exe" (
  start "Excelper" "%PACKAGE_DIR%\Excelper.exe"
  timeout /t 2 /nobreak >nul
  start "" "%URL%"
  exit /b 0
)

set GO_EXE=go
where go >nul 2>nul
if errorlevel 1 (
  if exist "C:\Program Files\Go\bin\go.exe" (
    set GO_EXE=C:\Program Files\Go\bin\go.exe
  ) else (
    echo [ERROR] 未找到 Go，也没有已打包的 %PACKAGE_DIR%\Excelper.exe
    echo 请先运行 win\build.bat 生成 exe，或安装 Go 后再运行本脚本。
    exit /b 1
  )
)

if not exist "%ROOT%\frontend\dist\index.html" (
  where npm >nul 2>nul
  if errorlevel 1 (
    echo [ERROR] 未找到前端构建产物，也未找到 npm。
    echo 请安装 Node/npm 后运行 win\build.bat，或先生成 frontend\dist。
    exit /b 1
  )
  pushd "%ROOT%\frontend"
  call npm install
  if errorlevel 1 exit /b 1
  call npm run build
  if errorlevel 1 exit /b 1
  popd
)

set EXCELPER_FRONTEND_DIR=%ROOT%\frontend\dist
start "Excelper Server" cmd /c "cd /d "%ROOT%" && "%GO_EXE%" run .\main.go"
timeout /t 3 /nobreak >nul
start "" "%URL%"

echo Excelper 已启动：%URL%
echo 数据目录：%EXCELPER_DATA_DIR%
endlocal
