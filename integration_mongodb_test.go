//go:build integration

package outboxer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/italolelis/outboxer"
	"github.com/italolelis/outboxer/es/amqp"
	"github.com/italolelis/outboxer/es/rabbitmq"
	"github.com/italolelis/outboxer/storage/mongodb"
	amqplib "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestEvent 表示测试事件结构
type TestEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// TestIntegration_MongoDB_RabbitMQ 测试 MongoDB + RabbitMQ 组合
func TestIntegration_MongoDB_RabbitMQ(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 设置 MongoDB
	mongoURI := getEnvOrDefault("MONGO_URI", "mongodb://localhost:27017")
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skip("MongoDB 不可用，跳过集成测试:", err)
	}
	defer mongoClient.Disconnect(ctx)

	// 创建 MongoDB 数据存储
	dataStore, err := mongodb.WithInstance(ctx, mongoClient, "outboxer_rabbitmq_test")
	if err != nil {
		t.Skip("创建 MongoDB 数据存储失败:", err)
	}
	defer dataStore.Close(ctx)

	// 设置 RabbitMQ 连接
	amqpURI := getEnvOrDefault("AMQP_URI", "amqp://guest:guest@localhost:5672/")
	amqpConn, err := amqplib.Dial(amqpURI)
	if err != nil {
		t.Skip("RabbitMQ 不可用，跳过集成测试:", err)
	}
	defer amqpConn.Close()

	// 创建 RabbitMQ 事件流
	eventStream := rabbitmq.NewRabbitMQ(amqpConn)

	// 创建 outboxer
	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(500*time.Millisecond),
		outboxer.WithCleanupInterval(5*time.Second),
		outboxer.WithMessageBatchSize(10),
	)
	if err != nil {
		t.Fatalf("创建 outboxer 失败: %v", err)
	}

	// 启动 outboxer
	go ob.Start(ctx)
	defer ob.Stop()

	// 监控结果
	var successCount, errorCount int
	var mu sync.Mutex

	go func() {
		for {
			select {
			case <-ob.OkChan():
				mu.Lock()
				successCount++
				mu.Unlock()
			case err := <-ob.ErrChan():
				mu.Lock()
				errorCount++
				mu.Unlock()
				t.Logf("处理消息时出错: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// 发送测试消息
	testMessages := 5
	for i := 0; i < testMessages; i++ {
		event := TestEvent{
			ID:        fmt.Sprintf("rabbitmq-test-%d", i),
			Type:      "mongodb-rabbitmq-integration",
			Payload:   fmt.Sprintf("RabbitMQ 测试消息 %d", i),
			Timestamp: time.Now(),
		}

		payload, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("序列化事件失败: %v", err)
		}

		msg := &outboxer.OutboxMessage{
			Payload: payload,
			Options: outboxer.DynamicValues{
				rabbitmq.ExchangeNameOption: "mongodb-rabbitmq-test",
				rabbitmq.ExchangeTypeOption: "topic",
				rabbitmq.RoutingKeyOption:   "test.mongodb.rabbitmq",
				rabbitmq.ContentTypeOption:  "application/json",
				rabbitmq.DeliveryModeOption: 2, // 持久化
				rabbitmq.PriorityOption:     uint8(i % 10),
			},
			Headers: outboxer.DynamicValues{
				"test-id":      event.ID,
				"test-type":    "mongodb-rabbitmq",
				"message-id":   i,
				"content-type": "application/json",
			},
		}

		if err := ob.Send(ctx, msg); err != nil {
			t.Fatalf("发送消息 %d 失败: %v", i, err)
		}
	}

	// 等待处理完成
	time.Sleep(3 * time.Second)

	mu.Lock()
	finalSuccessCount := successCount
	finalErrorCount := errorCount
	mu.Unlock()

	t.Logf("MongoDB + RabbitMQ 集成测试结果: %d 成功, %d 错误", finalSuccessCount, finalErrorCount)

	if finalSuccessCount == 0 {
		t.Error("期望至少有一些成功的消息投递")
	}

	if finalErrorCount > finalSuccessCount {
		t.Errorf("错误太多: %d 错误 vs %d 成功", finalErrorCount, finalSuccessCount)
	}
}

// TestIntegration_MongoDB_AMQP 测试 MongoDB + AMQP 组合
func TestIntegration_MongoDB_AMQP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 设置 MongoDB
	mongoURI := getEnvOrDefault("MONGO_URI", "mongodb://localhost:27017")
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skip("MongoDB 不可用，跳过集成测试:", err)
	}
	defer mongoClient.Disconnect(ctx)

	// 创建 MongoDB 数据存储
	dataStore, err := mongodb.WithInstance(ctx, mongoClient, "outboxer_amqp_test")
	if err != nil {
		t.Skip("创建 MongoDB 数据存储失败:", err)
	}
	defer dataStore.Close(ctx)

	// 设置 AMQP 连接
	amqpURI := getEnvOrDefault("AMQP_URI", "amqp://guest:guest@localhost:5672/")
	amqpConn, err := amqplib.Dial(amqpURI)
	if err != nil {
		t.Skip("AMQP 不可用，跳过集成测试:", err)
	}
	defer amqpConn.Close()

	// 创建 AMQP 事件流
	eventStream := amqp.NewAMQP(amqpConn)

	// 创建 outboxer
	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(500*time.Millisecond),
		outboxer.WithCleanupInterval(5*time.Second),
		outboxer.WithMessageBatchSize(8),
	)
	if err != nil {
		t.Fatalf("创建 outboxer 失败: %v", err)
	}

	// 启动 outboxer
	go ob.Start(ctx)
	defer ob.Stop()

	// 监控结果
	var successCount, errorCount int
	var mu sync.Mutex

	go func() {
		for {
			select {
			case <-ob.OkChan():
				mu.Lock()
				successCount++
				mu.Unlock()
			case err := <-ob.ErrChan():
				mu.Lock()
				errorCount++
				mu.Unlock()
				t.Logf("处理消息时出错: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// 发送测试消息
	testMessages := 6
	for i := 0; i < testMessages; i++ {
		event := TestEvent{
			ID:        fmt.Sprintf("amqp-test-%d", i),
			Type:      "mongodb-amqp-integration",
			Payload:   fmt.Sprintf("AMQP 测试消息 %d", i),
			Timestamp: time.Now(),
		}

		payload, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("序列化事件失败: %v", err)
		}

		msg := &outboxer.OutboxMessage{
			Payload: payload,
			Options: outboxer.DynamicValues{
				amqp.ExchangeNameOption: "mongodb-amqp-test",
				amqp.ExchangeTypeOption: "topic",
				amqp.RoutingKeyOption:   "test.mongodb.amqp",
				amqp.ExchangeDurable:    true,
				amqp.ExchangeAutoDelete: false,
				amqp.ExchangeInternal:   false,
				amqp.ExchangeNoWait:     false,
			},
			Headers: outboxer.DynamicValues{
				"test-id":    event.ID,
				"test-type":  "mongodb-amqp",
				"message-id": i,
				"timestamp":  event.Timestamp.Unix(),
			},
		}

		if err := ob.Send(ctx, msg); err != nil {
			t.Fatalf("发送消息 %d 失败: %v", i, err)
		}
	}

	// 等待处理完成
	time.Sleep(3 * time.Second)

	mu.Lock()
	finalSuccessCount := successCount
	finalErrorCount := errorCount
	mu.Unlock()

	t.Logf("MongoDB + AMQP 集成测试结果: %d 成功, %d 错误", finalSuccessCount, finalErrorCount)

	if finalSuccessCount == 0 {
		t.Error("期望至少有一些成功的消息投递")
	}

	if finalErrorCount > finalSuccessCount {
		t.Errorf("错误太多: %d 错误 vs %d 成功", finalErrorCount, finalSuccessCount)
	}
}

// TestIntegration_MongoDB_Comparison 比较 RabbitMQ 和 AMQP 的性能
func TestIntegration_MongoDB_Comparison(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// 设置 MongoDB
	mongoURI := getEnvOrDefault("MONGO_URI", "mongodb://localhost:27017")
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skip("MongoDB 不可用，跳过比较测试:", err)
	}
	defer mongoClient.Disconnect(ctx)

	// 设置 AMQP 连接
	amqpURI := getEnvOrDefault("AMQP_URI", "amqp://guest:guest@localhost:5672/")
	amqpConn, err := amqplib.Dial(amqpURI)
	if err != nil {
		t.Skip("AMQP 不可用，跳过比较测试:", err)
	}
	defer amqpConn.Close()

	// 测试 RabbitMQ 性能
	t.Run("RabbitMQ_Performance", func(t *testing.T) {
		dataStore, err := mongodb.WithInstance(ctx, mongoClient, "outboxer_rabbitmq_perf")
		if err != nil {
			t.Fatalf("创建 MongoDB 数据存储失败: %v", err)
		}
		defer dataStore.Close(ctx)

		eventStream := rabbitmq.NewRabbitMQ(amqpConn)
		testPerformance(t, ctx, dataStore, eventStream, "rabbitmq", 20)
	})

	// 测试 AMQP 性能
	t.Run("AMQP_Performance", func(t *testing.T) {
		dataStore, err := mongodb.WithInstance(ctx, mongoClient, "outboxer_amqp_perf")
		if err != nil {
			t.Fatalf("创建 MongoDB 数据存储失败: %v", err)
		}
		defer dataStore.Close(ctx)

		eventStream := amqp.NewAMQP(amqpConn)
		testPerformance(t, ctx, dataStore, eventStream, "amqp", 20)
	})
}

// testPerformance 执行性能测试
func testPerformance(t *testing.T, ctx context.Context, dataStore outboxer.DataStore, eventStream outboxer.EventStream, streamType string, messageCount int) {
	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(100*time.Millisecond),
		outboxer.WithMessageBatchSize(15),
	)
	if err != nil {
		t.Fatalf("创建 outboxer 失败: %v", err)
	}

	go ob.Start(ctx)
	defer ob.Stop()

	var successCount, errorCount int
	var mu sync.Mutex
	startTime := time.Now()

	go func() {
		for {
			select {
			case <-ob.OkChan():
				mu.Lock()
				successCount++
				mu.Unlock()
			case err := <-ob.ErrChan():
				mu.Lock()
				errorCount++
				mu.Unlock()
				t.Logf("%s 处理消息时出错: %v", streamType, err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// 发送测试消息
	for i := 0; i < messageCount; i++ {
		event := TestEvent{
			ID:        fmt.Sprintf("%s-perf-%d", streamType, i),
			Type:      fmt.Sprintf("mongodb-%s-performance", streamType),
			Payload:   fmt.Sprintf("%s 性能测试消息 %d", streamType, i),
			Timestamp: time.Now(),
		}

		payload, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("序列化事件失败: %v", err)
		}

		var msg *outboxer.OutboxMessage
		if streamType == "rabbitmq" {
			msg = &outboxer.OutboxMessage{
				Payload: payload,
				Options: outboxer.DynamicValues{
					rabbitmq.ExchangeNameOption: "perf-test-rabbitmq",
					rabbitmq.ExchangeTypeOption: "topic",
					rabbitmq.RoutingKeyOption:   "perf.rabbitmq",
					rabbitmq.ContentTypeOption:  "application/json",
				},
			}
		} else {
			msg = &outboxer.OutboxMessage{
				Payload: payload,
				Options: outboxer.DynamicValues{
					amqp.ExchangeNameOption: "perf-test-amqp",
					amqp.ExchangeTypeOption: "topic",
					amqp.RoutingKeyOption:   "perf.amqp",
					amqp.ExchangeDurable:    true,
				},
			}
		}

		if err := ob.Send(ctx, msg); err != nil {
			t.Fatalf("发送 %s 消息 %d 失败: %v", streamType, i, err)
		}
	}

	// 等待处理完成
	time.Sleep(5 * time.Second)
	duration := time.Since(startTime)

	mu.Lock()
	finalSuccessCount := successCount
	finalErrorCount := errorCount
	mu.Unlock()

	t.Logf("%s 性能测试结果:", streamType)
	t.Logf("  - 总时间: %v", duration)
	t.Logf("  - 成功: %d/%d", finalSuccessCount, messageCount)
	t.Logf("  - 错误: %d", finalErrorCount)
	t.Logf("  - 吞吐量: %.2f 消息/秒", float64(finalSuccessCount)/duration.Seconds())

	if finalSuccessCount == 0 {
		t.Errorf("%s: 期望至少有一些成功的消息投递", streamType)
	}
}

// TestIntegration_MongoDB_TransactionSupport 测试事务支持
func TestIntegration_MongoDB_TransactionSupport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 设置 MongoDB
	mongoURI := getEnvOrDefault("MONGO_URI", "mongodb://localhost:27017")
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skip("MongoDB 不可用，跳过事务测试:", err)
	}
	defer mongoClient.Disconnect(ctx)

	// 创建 MongoDB 数据存储
	dataStore, err := mongodb.WithInstance(ctx, mongoClient, "outboxer_transaction_test")
	if err != nil {
		t.Skip("创建 MongoDB 数据存储失败:", err)
	}
	defer dataStore.Close(ctx)

	// 设置 RabbitMQ
	amqpURI := getEnvOrDefault("AMQP_URI", "amqp://guest:guest@localhost:5672/")
	amqpConn, err := amqplib.Dial(amqpURI)
	if err != nil {
		t.Skip("RabbitMQ 不可用，跳过事务测试:", err)
	}
	defer amqpConn.Close()

	eventStream := rabbitmq.NewRabbitMQ(amqpConn)

	// 创建 outboxer
	ob, err := outboxer.New(
		outboxer.WithDataStore(dataStore),
		outboxer.WithEventStream(eventStream),
		outboxer.WithCheckInterval(300*time.Millisecond),
		outboxer.WithMessageBatchSize(5),
	)
	if err != nil {
		t.Fatalf("创建 outboxer 失败: %v", err)
	}

	go ob.Start(ctx)
	defer ob.Stop()

	var successCount int
	var mu sync.Mutex

	go func() {
		for {
			select {
			case <-ob.OkChan():
				mu.Lock()
				successCount++
				mu.Unlock()
			case err := <-ob.ErrChan():
				t.Logf("事务测试中处理消息时出错: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// 测试事务内发送消息
	testMessages := 3
	for i := 0; i < testMessages; i++ {
		event := TestEvent{
			ID:        fmt.Sprintf("tx-test-%d", i),
			Type:      "mongodb-transaction-test",
			Payload:   fmt.Sprintf("事务测试消息 %d", i),
			Timestamp: time.Now(),
		}

		payload, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("序列化事件失败: %v", err)
		}

		msg := &outboxer.OutboxMessage{
			Payload: payload,
			Options: outboxer.DynamicValues{
				rabbitmq.ExchangeNameOption: "transaction-test",
				rabbitmq.RoutingKeyOption:   "tx.test",
				rabbitmq.ContentTypeOption:  "application/json",
			},
		}

		// 在事务中发送消息
		err = ob.SendWithinTx(ctx, msg, func(execer outboxer.ExecerContext) error {
			// 模拟业务逻辑 - 这里可以执行其他数据库操作
			t.Logf("在事务中执行业务逻辑，消息 ID: %s", event.ID)
			return nil
		})

		if err != nil {
			t.Fatalf("在事务中发送消息 %d 失败: %v", i, err)
		}
	}

	// 等待处理完成
	time.Sleep(4 * time.Second)

	mu.Lock()
	finalSuccessCount := successCount
	mu.Unlock()

	t.Logf("MongoDB 事务测试结果: %d 成功", finalSuccessCount)

	if finalSuccessCount == 0 {
		t.Error("期望至少有一些成功的消息投递")
	}
}

// 辅助函数：获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
