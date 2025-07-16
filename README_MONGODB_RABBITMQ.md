# MongoDB + RabbitMQ Outboxer Implementation

本文档介绍如何使用MongoDB作为数据存储和RabbitMQ作为事件流的事务性发件箱实现。

## 功能特性

### MongoDB存储
- ✅ 完整的DataStore接口实现
- ✅ 支持MongoDB事务
- ✅ 自动创建索引优化查询性能
- ✅ 分布式锁机制防止并发问题
- ✅ TTL索引自动清理过期锁
- ✅ 批量操作支持

### RabbitMQ事件流
- ✅ 基于项目现有rabbitmq包实现
- ✅ 支持消息确认机制
- ✅ 可配置的exchange、routing key、优先级等
- ✅ 支持持久化和非持久化消息
- ✅ 灵活的消息头配置

## 快速开始

### 1. 基本使用

```go
package main

import (
    "context"
    "log"
    "time"

    "git.threatbook-inc.cn/csb/odyssey/pkg/outboxer"
    "git.threatbook-inc.cn/csb/odyssey/pkg/outboxer/es/rabbitmq"
    "git.threatbook-inc.cn/csb/odyssey/pkg/outboxer/storage/mongodb"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
    ctx := context.Background()

    // 连接MongoDB
    client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // 创建MongoDB数据存储
    dataStore, err := mongodb.WithInstance(ctx, client, "myapp")
    if err != nil {
        log.Fatal(err)
    }
    defer dataStore.Close(ctx)

    // 创建RabbitMQ事件流
    eventStream, err := rabbitmq.NewRabbitMQ(
        "my-exchange",    // exchange名称
        "topic",          // exchange类型
        "my-queue",       // 队列名称
        "events.#",       // routing key
        "amqp://guest:guest@localhost:5672/", // 连接字符串
    )
    if err != nil {
        log.Fatal(err)
    }
    defer eventStream.Close()

    // 创建outboxer实例
    ob, err := outboxer.New(
        outboxer.WithDataStore(dataStore),
        outboxer.WithEventStream(eventStream),
        outboxer.WithCheckInterval(2*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 发送消息
    msg := &outboxer.OutboxMessage{
        Payload: []byte(`{"message": "Hello, World!"}`),
        Options: outboxer.DynamicValues{
            rabbitmq.RoutingKeyOption:   "events.test",
            rabbitmq.ContentTypeOption:  "application/json",
            rabbitmq.DeliveryModeOption: 2, // 持久化
        },
    }

    if err := ob.Send(ctx, msg); err != nil {
        log.Fatal(err)
    }

    // 启动outboxer
    go ob.Start(ctx)

    // 等待消息处理
    select {
    case <-ob.OkChan():
        log.Println("消息发送成功!")
    case err := <-ob.ErrChan():
        log.Printf("错误: %v", err)
    }
}
```

### 2. 事务中使用

```go
// 在事务中发送消息
err = ob.SendWithinTx(ctx, msg, func(execer outboxer.ExecerContext) error {
    // 在这里执行业务逻辑
    // 例如：更新数据库记录
    log.Println("执行业务逻辑...")
    return nil
})
```

## 配置选项

### MongoDB存储配置

MongoDB存储会自动创建以下集合：
- `event_store`: 存储outbox消息
- `outboxer_locks`: 存储分布式锁

### RabbitMQ事件流配置

支持的消息选项：

| 选项 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `exchange.name` | string | Exchange名称 | - |
| `exchange.type` | string | Exchange类型 | - |
| `queue.name` | string | 队列名称 | - |
| `routing_key` | string | 路由键 | - |
| `content_type` | string | 内容类型 | `application/json` |
| `delivery_mode` | uint8/int/string | 投递模式 (1=非持久化, 2=持久化) | `2` |
| `priority` | uint8/int/string | 消息优先级 (0-255) | `0` |

示例：

```go
msg := &outboxer.OutboxMessage{
    Payload: []byte(`{"order_id": "12345"}`),
    Options: outboxer.DynamicValues{
        rabbitmq.ExchangeNameOption: "orders",
        rabbitmq.RoutingKeyOption:   "orders.created",
        rabbitmq.ContentTypeOption:  "application/json",
        rabbitmq.DeliveryModeOption: 2,
        rabbitmq.PriorityOption:     5,
    },
    Headers: outboxer.DynamicValues{
        "source":      "order-service",
        "event-type":  "order-created",
        "timestamp":   time.Now().Unix(),
    },
}
```

## 性能优化

### MongoDB索引

自动创建的索引：
- `{dispatched: 1, created_at: 1}` - 用于查找未发送消息
- `{dispatched: 1, dispatched_at: 1}` - 用于清理已发送消息
- `{expires_at: 1}` - TTL索引，自动清理过期锁

### 批量处理

```go
ob, err := outboxer.New(
    outboxer.WithDataStore(dataStore),
    outboxer.WithEventStream(eventStream),
    outboxer.WithMessageBatchSize(100),    // 每次处理100条消息
    outboxer.WithCleanUpBatchSize(500),    // 每次清理500条消息
    outboxer.WithCheckInterval(1*time.Second),
    outboxer.WithCleanupInterval(5*time.Minute),
)
```

## 错误处理

```go
go func() {
    for {
        select {
        case <-ob.OkChan():
            log.Println("消息发送成功")
        case err := <-ob.ErrChan():
            log.Printf("处理消息时出错: %v", err)
            // 根据错误类型进行处理
        }
    }
}()
```

## 监控和日志

outboxer使用logrus进行日志记录，支持以下日志级别：
- `INFO`: 连接状态、消息处理状态
- `WARN`: 连接重试、临时错误
- `ERROR`: 严重错误、连接失败

## 部署注意事项

### MongoDB
- 确保MongoDB支持事务（副本集或分片集群）
- 建议设置合适的连接池大小
- 监控集合大小，定期清理旧消息

### RabbitMQ
- 确保RabbitMQ服务可用
- 配置合适的队列持久化策略
- 监控队列长度和消息积压

### 网络
- 确保应用与MongoDB、RabbitMQ之间网络稳定
- 配置合适的连接超时和重试策略

## 完整示例

查看 `pkg/outboxer/examples/mongodb_rabbitmq_example.go` 获取完整的使用示例，包括：
- 订单事件处理
- 事务中的消息发送
- 批量消息处理
- 错误处理和监控

## 测试

运行测试：

```bash
# 测试MongoDB存储
go test ./pkg/outboxer/storage/mongodb/...

# 测试RabbitMQ事件流
go test ./pkg/outboxer/es/rabbitmq/...

# 运行完整示例
go run pkg/outboxer/examples/mongodb_rabbitmq_example.go
```

注意：测试需要本地运行MongoDB和RabbitMQ服务。