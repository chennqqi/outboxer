# Outboxer 使用指南

这是一个独立的 Outboxer 包，用于实现事务性发件箱模式（Transactional Outbox Pattern）。

## 概述

Outboxer 包提供了以下功能：
- 事务性消息发送，确保数据一致性
- 支持多种数据存储：MongoDB、MySQL、PostgreSQL、SQL Server
- 支持多种消息队列：AMQP (RabbitMQ)、AWS SQS、AWS Kinesis、Google Pub/Sub
- 自动重试和错误处理
- 消息清理和批处理

## 架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │───▶│   Data Store    │───▶│  Event Stream   │
│                 │    │   (MongoDB)     │    │    (AMQP)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 支持的组件

### 数据存储 (Data Stores)
- **MongoDB** - 推荐用于高性能场景
- **MySQL** - 关系型数据库支持
- **PostgreSQL** - 高级关系型数据库功能
- **SQL Server** - 企业级数据库支持

### 事件流 (Event Streams)
- **AMQP** - RabbitMQ 支持（推荐）
- **AWS SQS** - Amazon Simple Queue Service
- **AWS Kinesis** - 实时数据流处理
- **Google Pub/Sub** - Google Cloud 消息服务

## 快速开始

### 1. 安装依赖

```bash
go mod init your-project
go get github.com/italolelis/outboxer
```

### 2. 基本使用示例

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/italolelis/outboxer"
    "github.com/italolelis/outboxer/es/amqp"
    "github.com/italolelis/outboxer/storage/mongodb"
    amqplib "github.com/rabbitmq/amqp091-go"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
    ctx := context.Background()

    // 1. 设置 MongoDB 连接
    mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Fatal(err)
    }
    defer mongoClient.Disconnect(ctx)

    // 2. 创建数据存储
    dataStore, err := mongodb.WithInstance(ctx, mongoClient, "myapp")
    if err != nil {
        log.Fatal(err)
    }
    defer dataStore.Close(ctx)

    // 3. 创建 AMQP 连接和事件流
    amqpConn, err := amqplib.Dial("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Fatal(err)
    }
    defer amqpConn.Close()

    eventStream := amqp.NewAMQP(amqpConn)

    // 4. 创建 Outboxer 实例
    ob, err := outboxer.New(
        outboxer.WithDataStore(dataStore),
        outboxer.WithEventStream(eventStream),
        outboxer.WithCheckInterval(2*time.Second),
        outboxer.WithCleanupInterval(10*time.Minute),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 5. 启动 Outboxer
    go ob.Start(ctx)

    // 6. 发送消息
    msg := &outboxer.OutboxMessage{
        Payload: []byte(`{"event": "user_created", "user_id": "123"}`),
        Options: outboxer.DynamicValues{
            amqp.ExchangeNameOption: "events",
            amqp.ExchangeTypeOption: "topic",
            amqp.RoutingKeyOption:   "user.created",
        },
    }

    if err := ob.Send(ctx, msg); err != nil {
        log.Fatal(err)
    }

    // 等待消息处理
    time.Sleep(5 * time.Second)
    ob.Stop()
}
```

### 3. 在事务中使用

```go
// 在业务事务中发送消息
err = ob.SendWithinTx(ctx, msg, func(execer outboxer.ExecerContext) error {
    // 执行业务逻辑
    // 如果这里出错，消息也不会发送
    return performBusinessLogic()
})
```

## 配置选项

### Outboxer 配置

```go
ob, err := outboxer.New(
    outboxer.WithDataStore(dataStore),
    outboxer.WithEventStream(eventStream),
    outboxer.WithCheckInterval(2*time.Second),        // 检查新消息的频率
    outboxer.WithCleanupInterval(10*time.Minute),     // 清理旧消息的频率
    outboxer.WithMessageBatchSize(50),                // 批处理消息数量
    outboxer.WithCleanUpBatchSize(100),               // 清理批处理数量
    outboxer.WithCleanUpBefore(time.Now().Add(-24*time.Hour)), // 清理24小时前的消息
)
```

### AMQP 配置选项

```go
msg := &outboxer.OutboxMessage{
    Payload: payload,
    Options: outboxer.DynamicValues{
        amqp.ExchangeNameOption:   "my-exchange",
        amqp.ExchangeTypeOption:   "topic",        // direct, fanout, topic, headers
        amqp.RoutingKeyOption:     "my.routing.key",
        amqp.ExchangeDurable:      true,           // 持久化交换机
        amqp.ExchangeAutoDelete:   false,          // 自动删除
        amqp.ExchangeInternal:     false,          // 内部交换机
        amqp.ExchangeNoWait:       false,          // 不等待确认
    },
    Headers: outboxer.DynamicValues{
        "content-type": "application/json",
        "priority":     5,
    },
}
```

## 监控和错误处理

```go
// 监控 Outboxer 事件
go func() {
    for {
        select {
        case <-ob.OkChan():
            log.Println("✅ 消息发送成功")
        case err := <-ob.ErrChan():
            log.Printf("❌ 消息发送失败: %v", err)
        case <-ctx.Done():
            return
        }
    }
}()
```

## 最佳实践

1. **使用事务**: 总是在业务事务中发送消息，确保数据一致性
2. **合理配置批处理**: 根据消息量调整批处理大小
3. **监控错误**: 实现适当的错误处理和重试机制
4. **定期清理**: 配置合适的清理策略，避免数据库膨胀
5. **连接管理**: 正确管理数据库和消息队列连接

## 故障排除

### 常见问题

1. **消息未发送**
   - 检查 Outboxer 是否已启动
   - 验证数据存储和事件流连接
   - 查看错误日志

2. **性能问题**
   - 调整批处理大小
   - 优化检查间隔
   - 考虑数据库索引

3. **连接问题**
   - 验证连接字符串
   - 检查网络连接
   - 确认认证信息

## 示例项目

查看 `examples/` 目录中的完整示例：
- `mongodb_amqp_example.go` - MongoDB + AMQP 完整示例

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License