# Outboxer 集成测试指南

本指南介绍如何运行 RabbitMQ+MongoDB 和 AMQP+MongoDB 模式的 outboxer 集成测试。

## 前置条件

1. **Docker 和 Docker Compose**: 用于运行测试依赖的服务
2. **Go 1.19+**: 用于运行测试
3. **Make**: 用于执行测试命令

## 快速开始

### 1. 启动测试服务

```bash
# 启动 MongoDB, RabbitMQ, PostgreSQL 服务
make docker-up

# 查看服务状态
docker-compose -f docker-compose.integration.yml ps

# 查看服务日志
make docker-logs
```

### 2. 运行集成测试

#### 运行所有 MongoDB 相关测试
```bash
make integration-test-mongodb
```

#### 运行特定组合测试
```bash
# MongoDB + RabbitMQ 测试
make test-mongodb-rabbitmq

# MongoDB + AMQP 测试  
make test-mongodb-amqp

# 性能对比测试
make test-mongodb-comparison
```

#### 运行完整集成测试套件
```bash
make integration-test-full
```

### 3. 清理环境

```bash
# 停止服务
make docker-down

# 清理所有数据和容器
make docker-clean
```

## 测试详情

### MongoDB + RabbitMQ 集成测试

**测试文件**: `integration_mongodb_test.go`
**测试函数**: `TestIntegration_MongoDB_RabbitMQ`

**测试内容**:
- 创建 MongoDB 数据存储
- 创建 RabbitMQ 事件流
- 发送 5 条测试消息
- 验证消息成功投递
- 测试 RabbitMQ 特有选项（优先级、内容类型、投递模式）

**特性测试**:
```go
Options: outboxer.DynamicValues{
    rabbitmq.ExchangeNameOption: "mongodb-rabbitmq-test",
    rabbitmq.ExchangeTypeOption: "topic", 
    rabbitmq.RoutingKeyOption:   "test.mongodb.rabbitmq",
    rabbitmq.ContentTypeOption:  "application/json",
    rabbitmq.DeliveryModeOption: 2, // 持久化
    rabbitmq.PriorityOption:     uint8(i % 10),
}
```

### MongoDB + AMQP 集成测试

**测试文件**: `integration_mongodb_test.go`
**测试函数**: `TestIntegration_MongoDB_AMQP`

**测试内容**:
- 创建 MongoDB 数据存储
- 创建 AMQP 事件流
- 发送 6 条测试消息
- 验证消息成功投递
- 测试 AMQP 特有选项（交换机属性）

**特性测试**:
```go
Options: outboxer.DynamicValues{
    amqp.ExchangeNameOption: "mongodb-amqp-test",
    amqp.ExchangeTypeOption: "topic",
    amqp.RoutingKeyOption:   "test.mongodb.amqp", 
    amqp.ExchangeDurable:    true,
    amqp.ExchangeAutoDelete: false,
    amqp.ExchangeInternal:   false,
    amqp.ExchangeNoWait:     false,
}
```

### 性能对比测试

**测试函数**: `TestIntegration_MongoDB_Comparison`

**测试内容**:
- 并行运行 RabbitMQ 和 AMQP 性能测试
- 每个测试发送 20 条消息
- 测量处理时间和吞吐量
- 比较两种实现的性能差异

**输出示例**:
```
rabbitmq 性能测试结果:
  - 总时间: 2.345s
  - 成功: 20/20
  - 错误: 0
  - 吞吐量: 8.53 消息/秒

amqp 性能测试结果:
  - 总时间: 2.123s
  - 成功: 20/20  
  - 错误: 0
  - 吞吐量: 9.42 消息/秒
```

### 事务支持测试

**测试函数**: `TestIntegration_MongoDB_TransactionSupport`

**测试内容**:
- 测试在 MongoDB 事务中发送消息
- 验证事务一致性
- 确保消息和业务逻辑的原子性

## 环境变量配置

测试使用以下环境变量（带默认值）:

```bash
# MongoDB 连接
MONGO_URI="mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin"

# AMQP/RabbitMQ 连接  
AMQP_URI="amqp://guest:guest@localhost:5672/"

# PostgreSQL 连接（用于其他测试）
POSTGRES_URI="postgres://postgres:password@localhost:5432/outboxer_test?sslmode=disable"
```

## 服务访问

启动服务后，可以通过以下地址访问：

- **MongoDB**: `localhost:27017`
  - 用户名: `admin`
  - 密码: `password`
  
- **RabbitMQ Management**: `http://localhost:15672`
  - 用户名: `guest`
  - 密码: `guest`
  
- **RabbitMQ AMQP**: `localhost:5672`

- **PostgreSQL**: `localhost:5432`
  - 用户名: `postgres`
  - 密码: `password`
  - 数据库: `outboxer_test`

## 故障排除

### 1. 服务启动失败
```bash
# 检查端口占用
netstat -tulpn | grep -E "(5672|27017|5432)"

# 清理并重新启动
make docker-clean
make docker-up
```

### 2. 测试连接失败
```bash
# 检查服务健康状态
docker-compose -f docker-compose.integration.yml ps

# 查看服务日志
make docker-logs
```

### 3. 权限问题
```bash
# 确保 Docker 有足够权限
sudo usermod -aG docker $USER
# 重新登录或重启终端
```

## 自定义测试

### 添加新的集成测试

1. 在 `integration_mongodb_test.go` 中添加新的测试函数
2. 使用 `//go:build integration` 标签
3. 在 `Makefile` 中添加对应的测试命令

### 修改测试配置

可以通过修改以下参数来调整测试行为：

```go
// 在测试中调整这些参数
outboxer.WithCheckInterval(500*time.Millisecond),    // 检查间隔
outboxer.WithCleanupInterval(5*time.Second),         // 清理间隔  
outboxer.WithMessageBatchSize(10),                   // 批处理大小
```

## 持续集成

在 CI/CD 流水线中运行测试：

```yaml
# GitHub Actions 示例
- name: Run Integration Tests
  run: |
    make docker-up
    sleep 30  # 等待服务就绪
    make integration-test-mongodb
    make docker-down
```

## 注意事项

1. **测试隔离**: 每个测试使用不同的数据库名称确保隔离
2. **资源清理**: 测试完成后自动清理资源
3. **超时设置**: 所有测试都有 30-45 秒的超时限制
4. **并发安全**: 测试中使用互斥锁保护共享状态
5. **错误处理**: 服务不可用时会跳过测试而不是失败