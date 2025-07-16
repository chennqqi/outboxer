# 修复 Ticker Panic 问题

## 问题描述

在运行基准测试时出现了以下 panic 错误：

```
panic: non-positive interval for NewTicker
goroutine 70 [running]:
time.NewTicker(0x20?)
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.2.linux-amd64/src/time/tick.go:38 +0xbb
github.com/italolelis/outboxer.(*Outboxer).StartCleanup(0xc000204600, {0xa4c148, 0xdc5ae0})
/home/runner/work/outboxer/outboxer/outboxer.go:146 +0x38
```

## 根本原因

问题出现在 `outboxer.go` 的 `New` 函数中，`checkInterval` 和 `cleanUpInterval` 字段没有设置默认值，导致它们的值为 0。当调用 `ob.Start(ctx)` 时：

1. `StartDispatcher` 方法调用 `time.NewTicker(o.checkInterval)`
2. `StartCleanup` 方法调用 `time.NewTicker(o.cleanUpInterval)`
3. 如果这些间隔值为 0，`time.NewTicker` 会 panic

## 修复方案

在 `New` 函数中为 `checkInterval` 和 `cleanUpInterval` 设置合理的默认值：

```go
// 修复前
func New(opts ...Option) (*Outboxer, error) {
	o := Outboxer{
		errChan:          make(chan error),
		okChan:           make(chan struct{}),
		messageBatchSize: messageBatchSize,
		cleanUpBatchSize: cleanUpBatchSize,
		// checkInterval 和 cleanUpInterval 没有默认值，为 0
	}
	// ...
}

// 修复后
func New(opts ...Option) (*Outboxer, error) {
	o := Outboxer{
		errChan:          make(chan error),
		okChan:           make(chan struct{}),
		messageBatchSize: messageBatchSize,
		cleanUpBatchSize: cleanUpBatchSize,
		checkInterval:    5 * time.Second,  // 默认检查间隔
		cleanUpInterval:  30 * time.Second, // 默认清理间隔
	}
	// ...
}
```

## 默认值选择理由

- **checkInterval: 5 秒**: 
  - 足够频繁以确保消息及时处理
  - 不会对系统造成过大负担
  - 可以通过 `WithCheckInterval` 选项覆盖

- **cleanUpInterval: 30 秒**:
  - 清理操作不需要太频繁
  - 给已发送消息足够的保留时间
  - 平衡存储空间和性能
  - 可以通过 `WithCleanupInterval` 选项覆盖

## 测试验证

修复后的基准测试结果：

```
BenchmarkOutboxer_Send-22                14218749        80.42 ns/op      46 B/op       0 allocs/op
BenchmarkOutboxer_SendWithinTx-22        13345922        87.46 ns/op      49 B/op       0 allocs/op
```

所有单元测试和集成测试都通过：

```
=== RUN   TestOutboxer_Send
started to listen for new messages
sending message...
waiting for successfully sent messages...
message received
--- PASS: TestOutboxer_Send (1.02s)
```

## 影响范围

这个修复：

1. **解决了 panic 问题**: 所有基准测试现在都能正常运行
2. **保持向后兼容**: 现有代码不需要修改
3. **提供合理默认值**: 新用户可以直接使用而无需配置间隔时间
4. **保持灵活性**: 用户仍可以通过选项函数自定义间隔时间

## 最佳实践建议

在使用 outboxer 时，建议：

1. **生产环境**: 根据业务需求调整间隔时间
   ```go
   ob, err := outboxer.New(
       outboxer.WithDataStore(dataStore),
       outboxer.WithEventStream(eventStream),
       outboxer.WithCheckInterval(1*time.Second),     // 高频检查
       outboxer.WithCleanupInterval(5*time.Minute),   // 定期清理
   )
   ```

2. **测试环境**: 使用较短的间隔以加快测试速度
   ```go
   ob, err := outboxer.New(
       outboxer.WithDataStore(dataStore),
       outboxer.WithEventStream(eventStream),
       outboxer.WithCheckInterval(100*time.Millisecond),
       outboxer.WithCleanupInterval(1*time.Second),
   )
   ```

3. **基准测试**: 根据测试目标调整间隔时间
   ```go
   // 快速基准测试
   outboxer.WithCheckInterval(10*time.Millisecond),
   
   // 内存使用基准测试
   outboxer.WithCleanupInterval(100*time.Millisecond),
   ```

## 相关文件

- `outboxer.go`: 主要修复文件
- `benchmark_test.go`: 触发问题的测试文件
- `options.go`: 选项配置文件
- `integration_test.go`: 集成测试验证