param()

Set-Location $PSScriptRoot

if (Get-Command wails -ErrorAction SilentlyContinue) {
  wails dev
  exit $LASTEXITCODE
}

go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 dev
exit $LASTEXITCODE
