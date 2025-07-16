# Windows 环境下的 Outboxer 集成测试指南

本指南专门针对 Windows 环境，介绍如何运行 RabbitMQ+MongoDB 和 AMQP+MongoDB 模式的 outboxer 集成测试。

## 前置条件

### 必需软件
1. **Docker Desktop for Windows** - [下载地址](https://www.docker.com/products/docker-desktop)
2. **Go 1.19+** - [下载地址](https://golang.org/dl/)
3. **PowerShell 5.0+** (Windows 10/11 自带)

### 验证安装
在 PowerShell 中运行以下命令验证安装：

```powershell
# 检查 Docker
docker --version
docker-compose --version

# 检查 Go
go version

# 检查 PowerShell 版本
$PSVersionTable.PSVersion
```

## 快速开始

### 方法一：使用 PowerShell 脚本（推荐）

```powershell
# 1. 运行所有集成测试
.\test_integration.ps1

# 2. 运行特定测试
.\test_integration.ps1 -TestType mongodb-rabbitmq
.\test_integration.ps1 -TestType mongodb-amqp
.\test_integration.ps1 -TestType mongodb-comparison

# 3. 仅启动服务（用于手动测试）
.\test_integration.ps1 -StartOnly

# 4. 清理环境
.\test_integration.ps1 -Cleanup

# 5. 查看帮助
.\test_integration.ps1 -Help
```

### 方法二：使用 Make 命令

```powershell
# 查看所有可用命令
make help-integration

# 运行 Windows 集成测试
make integration-test-windows
make integration-test-mongodb-windows
make integration-test-rabbitmq-windows
make integration-test-amqp-windows

# Docker 服务管理
make docker-up-windows
make docker-down-windows
```

### 方法三：手动执行

```powershell
# 1. 启动服务
docker-compose -f docker-compose.integration.yml up -d

# 2. 等待服务就绪
Start-Sleep -Seconds 20

# 3. 设置环境变量
$env:MONGO_URI = "mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin"
$env:AMQP_URI = "amqp://guest:guest@localhost:5672/"

# 4. 运行测试
go test --tags=integration -v -run "TestIntegration_MongoDB_RabbitMQ" ./...
go test --tags=integration -v -run "TestIntegration_MongoDB_AMQP" ./...

# 5. 停止服务
docker-compose -f docker-compose.integration.yml down
```

## 测试详情

### 1. MongoDB + RabbitMQ 集成测试

**命令**:
```powershell
.\test_integration.ps1 -TestType mongodb-rabbitmq
```

**测试内容**:
- 测试 RabbitMQ 特有功能：消息优先级、内容类型、投递模式
- 验证与 MongoDB 的集成
- 测试消息的可靠投递

**预期输出**:
```
[INFO] 运行 MongoDB + RabbitMQ 集成测试...
=== RUN   TestIntegration_MongoDB_RabbitMQ
    integration_mongodb_test.go:xxx: MongoDB + RabbitMQ 集成测试结果: 5 成功, 0 错误
--- PASS: TestIntegration_MongoDB_RabbitMQ (3.45s)
PASS
```

### 2. MongoDB + AMQP 集成测试

**命令**:
```powershell
.\test_integration.ps1 -TestType mongodb-amqp
```

**测试内容**:
- 测试 AMQP 底层协议功能
- 验证交换机声明和配置
- 测试消息路由

**预期输出**:
```
[INFO] 运行 MongoDB + AMQP 集成测试...
=== RUN   TestIntegration_MongoDB_AMQP
    integration_mongodb_test.go:xxx: MongoDB + AMQP 集成测试结果: 6 成功, 0 错误
--- PASS: TestIntegration_MongoDB_AMQP (3.12s)
PASS
```

### 3. 性能对比测试

**命令**:
```powershell
.\test_integration.ps1 -TestType mongodb-comparison
```

**测试内容**:
- 并行测试 RabbitMQ 和 AMQP 性能
- 比较吞吐量和延迟
- 生成性能报告

**预期输出**:
```
=== RUN   TestIntegration_MongoDB_Comparison/RabbitMQ_Performance
    integration_mongodb_test.go:xxx: rabbitmq 性能测试结果:
    integration_mongodb_test.go:xxx:   - 总时间: 2.345s
    integration_mongodb_test.go:xxx:   - 成功: 20/20
    integration_mongodb_test.go:xxx:   - 错误: 0
    integration_mongodb_test.go:xxx:   - 吞吐量: 8.53 消息/秒
=== RUN   TestIntegration_MongoDB_Comparison/AMQP_Performance
    integration_mongodb_test.go:xxx: amqp 性能测试结果:
    integration_mongodb_test.go:xxx:   - 总时间: 2.123s
    integration_mongodb_test.go:xxx:   - 成功: 20/20
    integration_mongodb_test.go:xxx:   - 错误: 0
    integration_mongodb_test.go:xxx:   - 吞吐量: 9.42 消息/秒
```

## 服务访问

测试运行时，可以通过以下地址访问服务：

### MongoDB
- **连接地址**: `localhost:27017`
- **用户名**: `admin`
- **密码**: `password`
- **数据库**: `outboxer_test`

**连接字符串**:
```
mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin
```

### RabbitMQ
- **AMQP 端口**: `localhost:5672`
- **管理界面**: `http://localhost:15672`
- **用户名**: `guest`
- **密码**: `guest`

**连接字符串**:
```
amqp://guest:guest@localhost:5672/
```

## 故障排除

### 1. PowerShell 执行策略问题

**错误信息**:
```
无法加载文件 test_integration.ps1，因为在此系统上禁止运行脚本
```

**解决方案**:
```powershell
# 临时允许脚本执行
Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process

# 或者直接运行
powershell -ExecutionPolicy Bypass -File test_integration.ps1
```

### 2. Docker Desktop 未启动

**错误信息**:
```
error during connect: Get "http://%2F%2F.%2Fpipe%2Fdocker_engine/v1.24/containers/json": open //./pipe/docker_engine: The system cannot find the file specified.
```

**解决方案**:
1. 启动 Docker Desktop
2. 等待 Docker 完全启动（系统托盘图标不再转动）
3. 重新运行测试

### 3. 端口占用问题

**检查端口占用**:
```powershell
# 检查 MongoDB 端口
netstat -an | findstr :27017

# 检查 RabbitMQ 端口
netstat -an | findstr :5672
netstat -an | findstr :15672
```

**解决方案**:
```powershell
# 停止占用端口的服务
.\test_integration.ps1 -Cleanup

# 或者手动清理
docker-compose -f docker-compose.integration.yml down -v
```

### 4. 网络连接问题

**检查 Docker 网络**:
```powershell
docker network ls
docker-compose -f docker-compose.integration.yml ps
```

**重置网络**:
```powershell
docker-compose -f docker-compose.integration.yml down
docker network prune -f
docker-compose -f docker-compose.integration.yml up -d
```

### 5. 测试超时问题

如果测试经常超时，可以调整超时时间：

```powershell
# 增加等待时间
.\test_integration.ps1 -StartOnly
Start-Sleep -Seconds 30  # 等待更长时间
# 然后手动运行测试
```

## 开发调试

### 查看服务日志

```powershell
# 查看所有服务日志
docker-compose -f docker-compose.integration.yml logs -f

# 查看特定服务日志
docker-compose -f docker-compose.integration.yml logs -f mongodb
docker-compose -f docker-compose.integration.yml logs -f rabbitmq
```

### 连接到服务进行调试

```powershell
# 连接到 MongoDB
docker exec -it outboxer-mongodb mongosh -u admin -p password --authenticationDatabase admin

# 连接到 RabbitMQ 容器
docker exec -it outboxer-rabbitmq bash
```

### 手动测试消息发送

```powershell
# 启动服务
.\test_integration.ps1 -StartOnly

# 在另一个 PowerShell 窗口中运行单个测试
$env:MONGO_URI = "mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin"
$env:AMQP_URI = "amqp://guest:guest@localhost:5672/"
go test --tags=integration -v -run "TestIntegration_MongoDB_RabbitMQ" ./... -count=1
```

## 性能优化建议

### Docker Desktop 设置
1. 分配足够的内存（建议 4GB+）
2. 分配足够的 CPU 核心（建议 2 核+）
3. 启用 WSL 2 后端（如果使用 WSL 2）

### Windows 防火墙
确保 Docker Desktop 被允许通过 Windows 防火墙。

### 磁盘空间
确保有足够的磁盘空间用于 Docker 镜像和容器数据。

## 持续集成

在 Windows CI 环境中运行：

```yaml
# GitHub Actions 示例
- name: Run Integration Tests on Windows
  shell: powershell
  run: |
    .\test_integration.ps1 -TestType mongodb-all
```

## 常用命令速查

```powershell
# 快速测试
.\test_integration.ps1 -TestType mongodb-rabbitmq

# 启动服务并保持运行
.\test_integration.ps1 -StartOnly

# 查看服务状态
docker-compose -f docker-compose.integration.yml ps

# 清理环境
.\test_integration.ps1 -Cleanup

# 查看帮助
.\test_integration.ps1 -Help
make help-integration
```