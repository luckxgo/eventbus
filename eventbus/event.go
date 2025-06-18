package eventbus

import "time"

// EventType 定义事件类型枚举
type EventType string

// 预定义事件类型常量
const (
	EventTypeUserCreated  EventType = "user.created"
	EventTypeOrderUpdated EventType = "order.updated"
	// 可根据实际需求添加更多事件类型
)

// Event 定义事件接口
// 所有自定义事件都应实现此接口
// Event 定义事件接口
// T 是事件参数类型，使用泛型实现类型安全
// 所有自定义事件都应实现此接口
type Event[T any] interface {
	// Type 返回事件类型
	Type() EventType
	// Timestamp 返回事件发生时间
	Timestamp() time.Time
	// Params 返回事件参数
	Params() T
}

// BaseEvent 提供基础事件实现
// 可作为自定义事件的嵌入类型或直接使用
// BaseEvent 提供基础事件实现
// T 是事件参数类型，使用泛型实现类型安全
// 可作为自定义事件的嵌入类型或直接使用
type BaseEvent[T any] struct {
	EventType EventType
	timestamp time.Time
	params    T
}

// NewBaseEvent 创建新的基础事件实例
// eventType 参数指定事件类型
// NewBaseEvent 创建新的基础事件实例
// params 事件参数，支持任意类型
func NewBaseEvent[T any](eventType EventType, params T) *BaseEvent[T] {
	return &BaseEvent[T]{
		EventType: eventType,
		timestamp: time.Now(),
		params:    params,
	}
}

// Type 实现Event接口
func (e *BaseEvent[T]) Type() EventType {
	return e.EventType
}

// Timestamp 实现Event接口
func (e *BaseEvent[T]) Timestamp() time.Time {
	return e.timestamp
}

// Params 返回事件参数
func (e *BaseEvent[T]) Params() T {
	return e.params
}

// CancellableEvent 可取消事件接口
// 实现此接口的事件支持在传播过程中被取消
// CancellableEvent 可取消事件接口
// T 是事件参数类型
// 实现此接口的事件支持在传播过程中被取消
type CancellableEvent[T any] interface {
	Event[T]
	// Cancel 取消事件
	Cancel()
	// IsCancelled 检查事件是否已取消
	IsCancelled() bool
}

// BaseCancellableEvent 基础可取消事件实现
// BaseCancellableEvent 基础可取消事件实现
// T 是事件参数类型
type BaseCancellableEvent[T any] struct {
	BaseEvent[T]
	cancelled bool
}

// NewBaseCancellableEvent 创建新的可取消事件
// NewBaseCancellableEvent 创建新的可取消事件
func NewBaseCancellableEvent[T any](eventType EventType, params T) *BaseCancellableEvent[T] {
	return &BaseCancellableEvent[T]{
		BaseEvent: *NewBaseEvent[T](eventType, params),
	}
}

// Cancel 实现CancellableEvent接口
func (e *BaseCancellableEvent[T]) Cancel() {
	e.cancelled = true
}

// IsCancelled 实现CancellableEvent接口
func (e *BaseCancellableEvent[T]) IsCancelled() bool {
	return e.cancelled
}
