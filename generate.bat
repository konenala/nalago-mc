@echo off
REM Windows 批处理脚本 - 生成所有封包
setlocal enabledelayedexpansion

echo ========================================
echo 🚀 Minecraft Protocol Packet Generator
echo ========================================
echo.

set PROTOCOL_JSON=E:\bot編寫\go-mc\minecraft-data-pc-1_21_10\data\pc\1.21.10\protocol.json
set GENERATOR=tools\enhanced-generator\main_v2.go
set OUTPUT_BASE=pkg\protocol\packet\game

REM 检查 protocol.json
if not exist "%PROTOCOL_JSON%" (
    echo ❌ 找不到 protocol.json: %PROTOCOL_JSON%
    exit /b 1
)

REM 生成 Client 封包
echo ========================================
echo 📦 生成 Client 封包
echo ========================================
go run %GENERATOR% -protocol "%PROTOCOL_JSON%" -output "%OUTPUT_BASE%\client" -direction client -codec=true -v
if errorlevel 1 (
    echo ❌ Client 封包生成失败
    exit /b 1
)
echo.

REM 生成 Server 封包
echo ========================================
echo 📦 生成 Server 封包
echo ========================================
go run %GENERATOR% -protocol "%PROTOCOL_JSON%" -output "%OUTPUT_BASE%\server" -direction server -codec=true -v
if errorlevel 1 (
    echo ❌ Server 封包生成失败
    exit /b 1
)
echo.

REM 修复变量名
echo ========================================
echo 🔧 修复子结构体变量名
echo ========================================
powershell -Command "Get-ChildItem -Path '%OUTPUT_BASE%\client\packet_*.go' | ForEach-Object { (Get-Content $_.FullName) -replace '(\s+)(temp, err = \(\*pk\.\w+\)\(&)p\.', '$1$2s.' -replace '(\s+)p\.(\w+) = ', '$1s.$2 = ' -replace '(\s+)(temp, err = )p\.', '$1$2s.' | Set-Content $_.FullName }"
powershell -Command "Get-ChildItem -Path '%OUTPUT_BASE%\server\packet_*.go' | ForEach-Object { (Get-Content $_.FullName) -replace '(\s+)(temp, err = \(\*pk\.\w+\)\(&)p\.', '$1$2s.' -replace '(\s+)p\.(\w+) = ', '$1s.$2 = ' -replace '(\s+)(temp, err = )p\.', '$1$2s.' | Set-Content $_.FullName }"
echo ✅ 修复完成
echo.

REM 统计
echo ========================================
echo 📊 生成统计
echo ========================================
echo.
echo 📦 Client 封包:
for /f %%i in ('dir /b "%OUTPUT_BASE%\client\packet_*.go" 2^>nul ^| find /c /v ""') do echo   总封包数: %%i
for /f %%i in ('findstr /m "// TODO" "%OUTPUT_BASE%\client\packet_*.go" 2^>nul ^| find /c /v ""') do echo   有 TODO: %%i
echo.
echo 📦 Server 封包:
for /f %%i in ('dir /b "%OUTPUT_BASE%\server\packet_*.go" 2^>nul ^| find /c /v ""') do echo   总封包数: %%i
for /f %%i in ('findstr /m "// TODO" "%OUTPUT_BASE%\server\packet_*.go" 2^>nul ^| find /c /v ""') do echo   有 TODO: %%i
echo.

REM 编译验证
echo ========================================
echo 🔍 编译验证
echo ========================================
cd %OUTPUT_BASE%\client
go build .
if errorlevel 1 (
    echo ❌ Client 封包编译失败
    cd ..\..\..\..
    exit /b 1
)
echo ✅ Client 封包编译成功

cd ..\server
go build .
if errorlevel 1 (
    echo ❌ Server 封包编译失败
    cd ..\..\..\..
    exit /b 1
)
echo ✅ Server 封包编译成功
cd ..\..\..\..

echo.
echo ========================================
echo ✅ 生成完成！
echo ========================================
echo.
echo 💡 提示:
echo   • 大部分封包可以直接使用
echo   • Switch 类型需要手动实现条件逻辑
echo   • 查看 tools\enhanced-generator\MANUAL_FIXES.md 了解详情
echo.

pause
