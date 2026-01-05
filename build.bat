@echo off
REM Build script untuk w2app

echo ========================================
echo   w2app Build Script
echo ========================================
echo.

echo [1/2] Building stub...
go build -ldflags="-s -w -H windowsgui" -o internal\generator\stubs\stub-windows-amd64.exe .\cmd\stub
if %errorlevel% neq 0 (
    echo [ERROR] Failed to build stub
    exit /b 1
)
echo       Done!

echo [2/2] Building w2app...
go build -ldflags="-s -w" -o w2app.exe .\cmd\w2app
if %errorlevel% neq 0 (
    echo [ERROR] Failed to build w2app
    exit /b 1
)
echo       Done!

echo.
echo ========================================
echo   Build Complete!
echo ========================================
echo.
echo Output: w2app.exe
echo.
echo Quick Start:
echo   w2app --url https://example.com --name MyApp
echo.
echo For help:
echo   w2app --help
echo   w2app create --help
