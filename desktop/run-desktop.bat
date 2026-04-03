@echo off
cd /d "%~dp0"

set GOTMPDIR=E:\tmp\go
set TEMP=E:\tmp\go
set TMP=E:\tmp\go

if not exist "E:\tmp\go" mkdir "E:\tmp\go"

where wails >nul 2>&1
if %ERRORLEVEL% EQU 0 (
  wails dev
  exit /b %ERRORLEVEL%
)

go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 dev
exit /b %ERRORLEVEL%
