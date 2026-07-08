@echo off
setlocal

set ROOT=%~dp0..
set WIN_DIR=%ROOT%\win
set PACKAGE_DIR=%WIN_DIR%\package

where npm >nul 2>nul
if errorlevel 1 (
  echo [ERROR] npm is required to build the frontend.
  exit /b 1
)

set GO_EXE=go
where go >nul 2>nul
if errorlevel 1 (
  if exist "C:\Program Files\Go\bin\go.exe" (
    set GO_EXE=C:\Program Files\Go\bin\go.exe
  ) else (
    echo [ERROR] Go 1.22 or newer is required to build Excelper.exe.
    exit /b 1
  )
)

if exist "%PACKAGE_DIR%" rmdir /s /q "%PACKAGE_DIR%"
mkdir "%PACKAGE_DIR%"
mkdir "%PACKAGE_DIR%\data"

pushd "%ROOT%\frontend"
call npm install
if errorlevel 1 exit /b 1
call npm run build
if errorlevel 1 exit /b 1
popd

xcopy /e /i /y "%ROOT%\frontend\dist" "%PACKAGE_DIR%\frontend-dist"
if errorlevel 1 exit /b 1
copy /y "%WIN_DIR%\assets\excelper-icon.svg" "%PACKAGE_DIR%\excelper-icon.svg" >nul
if errorlevel 1 exit /b 1

pushd "%ROOT%"
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
"%GO_EXE%" build -mod=mod -ldflags "-s -w -H=windowsgui" -o "%PACKAGE_DIR%\Excelper.exe" .\main.go
if errorlevel 1 exit /b 1
popd

echo.
echo Excelper Windows package created:
echo %PACKAGE_DIR%
echo.
echo Put .xls files into:
echo %PACKAGE_DIR%\data
echo.
endlocal