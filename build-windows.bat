@echo off
echo 正在编译Windows版本的DNS代理程序...
echo 使用优化参数进行编译...

REM 设置Go路径（如果需要的话，可以修改为你的Go安装路径）
REM set GO_PATH=C:\Go\bin

REM 检查Go是否可用
where go >nul 2>nul
if %errorlevel% neq 0 (
    echo 错误: 找不到Go编译器，请确保Go已安装并添加到PATH
    exit /b 1
)

REM 设置编译参数
set GOOS=windows
set GOARCH=amd64
set OUTPUT_FILE=dns-proxy.exe

REM 优化编译参数
set BUILD_FLAGS=-ldflags="-s -w -X main.version=1.0.0" -trimpath
echo.
echo 编译优化参数:
echo   - 去除符号表和调试信息 (-s -w)  减小文件体积
echo   - 去除文件路径信息 (-trimpath)   保护构建路径隐私
echo   - 设置版本信息 (-X main.version) 可追踪版本
echo.

echo 目标系统: Windows (amd64)
echo 输出文件: %OUTPUT_FILE%

REM 执行编译
go build %BUILD_FLAGS% -o %OUTPUT_FILE% main.go

if %errorlevel% equ 0 (
    echo.
    echo 编译成功!
    echo.
    echo 文件信息:
    dir %OUTPUT_FILE% | findstr /C:"%OUTPUT_FILE%"
    echo.
    echo 文件已优化，体积更小，运行更快!
) else (
    echo.
    echo 编译失败!
    exit /b 1
)

echo.
echo 使用方法:
echo   %OUTPUT_FILE% -port 53 -upstream 8.8.8.8:53
echo.
echo -e "${YELLOW}参数说明:${NC}"
echo -e "  -port          : 监听端口，默认53"
echo -e "  -upstream      : 上游DNS服务器，默认8.8.8.8:53"
echo -e "  -debug         : 启用调试模式"
echo -e "  -test-interval : 速度测试间隔（秒）"
echo -e "  -service       : 服务操作(install/uninstall/start/stop)"
echo.
echo 示例:
echo   %OUTPUT_FILE% -upstream "8.8.8.8:53,1.1.1.1:53" -test-interval 300
echo.
echo 服务管理:
echo   安装服务: %OUTPUT_FILE% -service install -upstream "8.8.8.8:53,1.1.1.1:53"
echo   启动服务: %OUTPUT_FILE% -service start
echo   停止服务: %OUTPUT_FILE% -service stop
echo   卸载服务: %OUTPUT_FILE% -service uninstall
echo.
pause