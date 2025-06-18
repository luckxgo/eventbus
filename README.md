# 事件发布订阅系统

一个基于Go 1.22+实现的高性能、类型安全的事件发布订阅系统，支持同步/异步事件处理、事件取消和并发安全。

## 特性

- **类型安全**：使用Go泛型确保事件参数类型安全
- **枚举事件类型**：通过EventType枚举管理事件类型，避免字符串硬编码错误
- **灵活的事件处理**：支持同步和异步事件监听
- **可取消事件**：支持事件传播过程中的取消操作
- **并发安全**：内部实现使用读写锁和原子操作，确保高并发场景下的线程安全
- **监听器排序**：支持按优先级排序的事件处理
- **完整测试**：包含单元测试和并发测试，确保功能稳定性

## 安装

```bash
# 使用Go Modules
go get github.com/luckxgo/eventbus
```

## 快速开始

### 1. 定义事件类型和参数

```go
import (
  "time"
  "github.com/luckxgo/eventbus"
)

// 定义事件参数类型
type UserCreatedEvent struct {
  UserID    int64
  Username  string
  CreatedAt time.Time
}

// 定义自定义事件类型（扩展EventType枚举）
const (
  EventTypeUserCreated eventbus.EventType = "user.created"
  EventTypeOrderPaid   eventbus.EventType = "order.paid"
)
```

### 2. 创建事件总线和监听器

```go
// 创建事件总线实例
bus := eventbus.NewDefaultEventBus[UserCreatedEvent]()

defer bus.Shutdown(context.Background())

// 创建监听器
listener := eventbus.NewEventListenerFunc[UserCreatedEvent](
  10, // 优先级（值越小优先级越高）
  false, // 同步执行
  func(ctx context.Context, event eventbus.Event[UserCreatedEvent]) error {
    params := event.Params()
    fmt.Printf("用户创建: %s (ID: %d)\n", params.Username, params.UserID)
    return nil
  },
  EventTypeUserCreated, // 监听的事件类型
)

// 注册监听器
bus.RegisterListener(listener)
```

### 3. 发布事件

```go
// 创建事件
userEvent := eventbus.NewBaseEvent(
  EventTypeUserCreated,
  UserCreatedEvent{
    UserID:    123,
    Username:  "john_doe",
    CreatedAt: time.Now(),
  },
)

// 发布事件
errors := bus.Publish(context.Background(), userEvent)
if len(errors) > 0 {
  // 处理错误
  fmt.Printf("发布事件失败: %v\n", errors)
}
```

## 高级用法

### 异步事件处理

```go
// 创建异步监听器
asyncListener := eventbus.NewEventListenerFunc[UserCreatedEvent](
  20,
  true, // 异步执行
  func(ctx context.Context, event eventbus.Event[UserCreatedEvent]) error {
    // 异步处理逻辑
    return nil
  },
  EventTypeUserCreated,
)

bus.RegisterListener(asyncListener)
```

### 可取消事件

```go
// 创建可取消事件
cancellableEvent := eventbus.NewBaseCancellableEvent(
  EventTypeUserCreated,
  UserCreatedEvent{UserID: 456, Username: "jane_smith"},
)

// 在监听器中取消事件
listener := eventbus.NewEventListenerFunc[UserCreatedEvent](
  5,
  false,
  func(ctx context.Context, event eventbus.Event[UserCreatedEvent]) error {
    if cancellableEvent, ok := event.(eventbus.CancellableEvent[UserCreatedEvent]); ok {
      if event.Params().UserID == 456 {
        cancellableEvent.Cancel()
        return fmt.Errorf("用户ID 456 不允许创建")
      }
    }
    return nil
  },
  EventTypeUserCreated,
)
```

## API 文档

### 核心接口

#### Event[T]

事件接口，所有事件都应实现此接口：

```go
type Event[T any] interface {
  Type() EventType        // 返回事件类型
  Timestamp() time.Time   // 返回事件时间戳
  Params() T              // 返回事件参数
}
```

#### EventListener[T]

事件监听器接口：

```go
type EventListener[T any] interface {
  OnEvent(ctx context.Context, event Event[T]) error  // 事件处理方法
  SupportsEvent(eventType EventType) bool             // 是否支持指定事件类型
  SupportedEvents() []EventType                       // 返回支持的事件类型
  Order() int                                         // 返回优先级
  IsAsync() bool                                      // 是否异步执行
}
```

#### EventBus[T]

事件总线接口：

```go
type EventBus[T any] interface {
  RegisterListener(listener EventListener[T])         // 注册监听器
  UnregisterListener(listener EventListener[T])       // 注销监听器
  Publish(ctx context.Context, event Event[T]) []error // 发布事件
  Shutdown(ctx context.Context) error                 // 关闭事件总线
}
```

## 测试

运行测试用例：

```bash
# 基本测试
go test -v

# 并发测试（检测竞态条件）
go test -race -v
```

## 许可证

[MIT](LICENSE)
