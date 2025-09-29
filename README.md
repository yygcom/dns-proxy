# DNS代理程序

一个使用Go 1.23编写的高性能DNS代理程序，支持IPv6记录过滤、多上游DNS、智能速度排序、服务管理等高级功能。

## 🚀 功能特点

- ✅ **IPv6记录过滤**: 自动过滤所有AAAA记录，只返回IPv4地址
- ✅ **多上游DNS支持**: 支持配置多个上游DNS服务器，自动故障转移
- ✅ **智能速度排序**: 自动测试DNS服务器速度，按响应时间排序
- ✅ **服务管理**: 支持Linux systemd服务安装和管理
- ✅ **调试模式**: 详细的日志输出，便于故障排查
- ✅ **性能优化**: 编译优化，体积小，运行速度快
- ✅ **跨平台支持**: 支持Windows、Linux等多个平台
- ✅ **IPv6上游支持**: 支持IPv6地址作为上游DNS服务器

## 📦 程序文件

- `dns-proxy` - Linux当前系统版本
- `dns-proxy-linux` - Linux通用版本（amd64）
- `dns-proxy.exe` - Windows版本（amd64）

## 🚀 快速开始

### Linux系统
```bash
# 给程序添加执行权限
chmod +x dns-proxy-linux

# 基本用法（需要root权限）
sudo ./dns-proxy-linux

# 自定义配置
sudo ./dns-proxy-linux -upstream "8.8.8.8:53,1.1.1.1:53" -port 5353 -debug
```

### Windows系统
```cmd
# 基本用法（需要管理员权限）
dns-proxy.exe

# 自定义配置
dns-proxy.exe -upstream "8.8.8.8:53,1.1.1.1:53" -port 5353 -debug
```

## 📋 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-port` | 53 | DNS代理服务器监听端口 |
| `-upstream` | 8.8.8.8:53 | 上游DNS服务器地址（多个用逗号分隔） |
| `-debug` | false | 启用调试模式，显示详细日志 |
| `-test-interval` | 300 | DNS服务器速度测试间隔（秒，0禁用） |
| `-service` | "" | 服务操作(install/uninstall/start/stop) |

## 🏃‍♂️ 高级用法

### 多上游DNS配置
```bash
# 配置多个上游DNS，支持IPv4和IPv6
sudo ./dns-proxy-linux -upstream "8.8.8.8:53,1.1.1.1:53,240e:1f:1::1" -port 53

# 启用调试模式查看详细信息
sudo ./dns-proxy-linux -upstream "8.8.8.8:53,1.1.1.1:53" -debug
```

### 智能速度排序
```bash
# 每5分钟自动测试DNS速度并重新排序（默认）
sudo ./dns-proxy-linux -upstream "8.8.8.8:53,1.1.1.1:53,9.9.9.9:53" -test-interval 300

# 每1分钟测试一次（更频繁）
sudo ./dns-proxy-linux -upstream "8.8.8.8:53,1.1.1.1:53" -test-interval 60

# 禁用自动速度测试
sudo ./dns-proxy-linux -upstream "8.8.8.8:53,1.1.1.1:53" -test-interval 0
```

## 🏢 服务管理（Linux）

### 安装为systemd服务
```bash
# 安装服务（需要root权限）
sudo ./dns-proxy-linux -service install -upstream "8.8.8.8:53,1.1.1.1:53" -port 53

# 启动服务
sudo ./dns-proxy-linux -service start

# 查看服务状态
sudo ./dns-proxy-linux -service status

# 停止服务
sudo ./dns-proxy-linux -service stop

# 卸载服务
sudo ./dns-proxy-linux -service uninstall
```

### 服务管理（Windows）
Windows服务需要手动安装，程序会提供详细的操作指南：
```cmd
# 查看安装指南
dns-proxy.exe -service install -upstream "8.8.8.8:53,1.1.1.1:53"

# 按提示手动执行（需要管理员权限）:
# 1. 以管理员身份运行CMD
# 2. sc create dns-proxy binPath= "C:\path\to\dns-proxy.exe -upstream \"8.8.8.8:53,1.1.1.1:53\" -port 53" start= auto displayname= "DNS Proxy Service"
# 3. sc start dns-proxy
```

## 🧪 测试验证

### 使用dig命令测试
```bash
# 测试A记录（IPv4） - 应该正常返回
dig @127.0.0.1 -p 53 example.com A

# 测试AAAA记录（IPv6） - 应该被过滤，返回空结果或NXDOMAIN
dig @127.0.0.1 -p 53 example.com AAAA

# 测试特定域名
dig @127.0.0.1 -p 53 ipv6.baidu.com A
```

### 使用nslookup测试
```bash
# 测试A记录
nslookup -port=53 example.com 127.0.0.1

# 测试AAAA记录
nslookup -port=53 -type=AAAA example.com 127.0.0.1
```

### 使用编译脚本
```bash
# Linux
chmod +x build-linux.sh
./build-linux.sh

# Windows
build-windows.bat
```

## 🔧 编译说明

### 优化特性
- **体积优化**: 使用`-ldflags="-s -w"`去除调试信息，减小文件体积约30%
- **路径优化**: 使用`-trimpath`去除构建路径信息
- **版本信息**: 嵌入版本号，便于追踪

### 跨平台编译
```bash
# Linux版本
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -trimpath" -o dns-proxy-linux main.go

# Windows版本
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -trimpath" -o dns-proxy.exe main.go
```

## ⚠️ 注意事项

1. **权限要求**: 监听53端口需要root/administrator权限
2. **端口冲突**: 如果53端口被占用，请使用`-port`参数指定其他端口
3. **防火墙**: 确保防火墙允许DNS流量（UDP 53端口）
4. **IPv6支持**: 程序支持IPv6上游DNS，但会过滤返回的IPv6记录
5. **服务管理**: Linux服务需要systemd支持，Windows服务需要管理员权限

## 🔍 调试技巧

### 查看实时日志
```bash
# 启用调试模式查看详细信息
sudo ./dns-proxy-linux -debug -upstream "8.8.8.8:53,1.1.1.1:53"

# 查看DNS查询和过滤过程
# 输出示例：
# [DEBUG] 收到DNS查询: ID=12345, 问题数=1
# [DEBUG]   问题[0]: example.com A
# [DEBUG] 开始过滤IPv6记录，原始记录数: 3
# [DEBUG] 过滤掉IPv6记录: example.com. 300 IN AAAA 2404:6800:4008:801::200e
# [DEBUG] 保留记录: example.com. 300 IN A 142.250.190.46
```

### 服务状态检查
```bash
# Linux systemd服务
sudo systemctl status dns-proxy
sudo journalctl -u dns-proxy -f

# 查看服务日志
sudo ./dns-proxy-linux -service status
```

## 🐛 常见问题

### Q: 程序无法监听53端口？
A: 需要root权限，使用`sudo`运行或改用高端口（如5353）

### Q: 上游DNS服务器无响应？
A: 检查网络连接，使用`-debug`参数查看详细错误信息

### Q: IPv6记录没有被过滤？
A: 确保查询的是AAAA记录类型，程序只过滤AAAA记录

### Q: 服务安装失败？
A: Linux需要systemd支持，确保系统有systemd；Windows需要管理员权限

### Q: 编译失败？
A: 确保已安装Go 1.23+，并正确设置了GOPATH和PATH

## 📄 许可证

本项目采用MIT许可证，详见LICENSE文件。

## 🤝 贡献

欢迎提交Issue和Pull Request！