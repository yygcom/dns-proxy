# DNS代理程序

一个使用Go 1.23编写的DNS代理程序，可以过滤掉IPv6记录（AAAA记录）。

## 功能特点

- 可指定监听端口
- 可指定上游DNS服务器
- 自动过滤所有IPv6记录（AAAA记录）
- 优雅关闭支持

## 使用方法

### 基本用法
```bash
./dns-proxy
```
默认监听53端口，上游DNS为8.8.8.8:53

### 自定义端口和上游DNS
```bash
./dns-proxy -port 5353 -upstream 1.1.1.1:53
```

### 参数说明
- `-port`: 监听端口，默认为53
- `-upstream`: 上游DNS服务器地址，默认为8.8.8.8:53

## 测试方法

### 使用dig命令测试
```bash
# 测试A记录（IPv4）
dig @localhost -p 53 example.com A

# 测试AAAA记录（IPv6），应该返回空结果
dig @localhost -p 53 example.com AAAA
```

### 使用nslookup测试
```bash
# 测试A记录
nslookup -port=53 example.com localhost

# 测试AAAA记录
nslookup -port=53 -type=AAAA example.com localhost
```

## 注意事项

- 需要root权限或使用sudo运行，因为默认监听53端口
- 如果端口被占用，请使用其他端口
- 程序支持优雅关闭，使用Ctrl+C即可安全退出