# CI/CD 测试指南

## 概述

本文档描述了如何在CI/CD环境中运行Outboxer的测试套件，包括单元测试、集成测试和性能基准测试。

## 测试分层

### 1. 单元测试 (Unit Tests)
- **无外部依赖**: 可以在任何环境中运行
- **快速执行**: 通常在几秒内完成
- **高覆盖率**: 覆盖核心业务逻辑

```bash
# 运行所有单元测试
go test ./...

# 运行单元测试并生成覆盖率报告
go test -cover -coverprofile=coverage.out ./...
```

### 2. 集成测试 (Integration Tests)
- **需要外部服务**: MongoDB, PostgreSQL, RabbitMQ, AMQP
- **标记为integration**: 使用build tags分离
- **真实环境测试**: 验证端到端功能

```bash
# 运行集成测试 (需要服务运行)
go test -tags=integration ./...

# 运行特定集成测试
go test -tags=integration -run TestIntegration_MongoDB_AMQP ./...
```

### 3. 性能基准测试 (Benchmark Tests)
- **性能验证**: 测试关键操作的性能
- **回归检测**: 确保性能不会退化

```bash
# 运行基准测试
go test -bench=. ./...

# 运行特定基准测试
go test -bench=BenchmarkOutboxer_Send ./...
```

## CI/CD 策略

### 快速反馈策略 (推荐)

对于大多数CI/CD场景，建议采用分层测试策略：

1. **Pull Request检查**: 只运行单元测试
2. **主分支合并**: 运行单元测试 + 轻量级集成测试
3. **定期构建**: 运行完整测试套件

### GitHub Actions 配置

#### 基础配置 (单元测试)
```yaml
name: Unit Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - uses: actions/checkout@v3
    
    - name: Run unit tests
      run: go test -v -cover ./...
    
    - name: Run benchmarks (quick)
      run: go test -bench=. -benchtime=100ms ./...
```

#### 完整配置 (包含集成测试)
```yaml
name: Full Test Suite
on:
  push:
    branches: [main, master]
  schedule:
    - cron: '0 2 * * *'  # 每天凌晨2点运行

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:13
        env:
          POSTGRES_PASSWORD: admin
          POSTGRES_USER: admin
          POSTGRES_DB: outboxer
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
      
      mongodb:
        image: mongo:5.0
        env:
          MONGO_INITDB_ROOT_USERNAME: admin
          MONGO_INITDB_ROOT_PASSWORD: password
        ports:
          - 27017:27017
      
      rabbitmq:
        image: rabbitmq:3.8-alpine
        ports:
          - 5672:5672

    steps:
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - uses: actions/checkout@v3
    
    - name: Run unit tests
      run: go test -v -cover -coverprofile=coverage.out ./...
    
    - name: Run integration tests
      env:
        MONGO_URI: "mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin"
        POSTGRES_URI: "postgres://admin:admin@localhost:5432/outboxer?sslmode=disable"
        AMQP_URI: "amqp://guest:guest@localhost:5672/"
      run: go test -tags=integration -v ./...
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        files: ./coverage.out
```

## 本地开发测试

### 使用Docker Compose

```bash
# 启动所有服务
docker-compose -f docker-compose.integration.yml up -d

# 等待服务就绪
sleep 15

# 运行完整测试套件
MONGO_URI="mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin" \
POSTGRES_URI="postgres://postgres:password@localhost:5432/outboxer_test?sslmode=disable" \
AMQP_URI="amqp://guest:guest@localhost:5672/" \
go test -tags=integration -v ./...

# 清理
docker-compose -f docker-compose.integration.yml down -v
```

### 使用Makefile

```bash
# 运行单元测试
make test

# 运行集成测试 (Linux/Mac)
make integration-test-full

# 运行集成测试 (Windows)
make integration-test-windows

# 查看所有可用命令
make help-integration
```

## 故障排除

### 常见问题

1. **docker-compose命令未找到**
   ```bash
   # 安装Docker Compose
   sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
   sudo chmod +x /usr/local/bin/docker-compose
   ```

2. **服务启动超时**
   ```bash
   # 增加等待时间
   sleep 30
   
   # 检查服务状态
   docker-compose ps
   docker-compose logs [service-name]
   ```

3. **MongoDB连接失败**
   ```bash
   # 检查MongoDB是否就绪
   docker-compose exec mongodb mongosh --eval "db.adminCommand('ping')"
   
   # 或使用旧版mongo命令
   docker-compose exec mongodb mongo --eval "db.adminCommand('ping')"
   ```

4. **PostgreSQL连接失败**
   ```bash
   # 检查PostgreSQL是否就绪
   docker-compose exec postgres pg_isready -U admin -d outboxer
   ```

### 调试技巧

1. **查看服务日志**
   ```bash
   docker-compose logs -f [service-name]
   ```

2. **检查网络连接**
   ```bash
   docker-compose exec [service-name] netstat -tlnp
   ```

3. **手动测试连接**
   ```bash
   # 测试MongoDB连接
   mongosh "mongodb://admin:password@localhost:27017/outboxer_test?authSource=admin"
   
   # 测试PostgreSQL连接
   psql "postgres://admin:admin@localhost:5432/outboxer?sslmode=disable"
   ```

## 性能考虑

### CI环境优化

1. **并行测试**: 使用`-parallel`标志
2. **缓存依赖**: 缓存Go模块和Docker镜像
3. **分层测试**: 快速失败策略
4. **资源限制**: 合理设置超时和重试

### 示例优化配置

```yaml
- name: Cache Go modules
  uses: actions/cache@v3
  with:
    path: ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

- name: Run tests with timeout
  timeout-minutes: 10
  run: go test -timeout=5m -v ./...
```

## 最佳实践

1. **测试隔离**: 每个测试使用独立的数据库/集合
2. **清理资源**: 测试后清理创建的资源
3. **幂等性**: 测试应该可以重复运行
4. **快速失败**: 尽早发现问题
5. **详细日志**: 提供足够的调试信息

## 总结

通过合理的测试分层和CI/CD策略，可以在保证代码质量的同时，提供快速的反馈循环。建议在开发过程中主要依赖单元测试，在关键节点运行完整的集成测试套件。