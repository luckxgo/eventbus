package eventbus

import (
	"context"
)

// EventListener 定义事件监听器接口
// T 是事件参数类型
// 所有事件监听器都应实现此接口
type EventListener[T any] interface {
	// OnEvent 处理事件的方法
	// ctx 用于传递上下文，可用于超时控制和取消操作
	// event 是要处理的事件实例
	OnEvent(ctx context.Context, event Event[T]) error

	// SupportsEvent 判断监听器是否支持处理指定事件
	// eventType 是事件类型
	SupportsEvent(eventType EventType) bool

	// SupportedEvents 返回监听器支持的所有事件类型
	SupportedEvents() []EventType

	// Order 返回监听器的执行顺序
	// 数值越小，优先级越高
	Order() int

	// IsAsync 是否异步执行事件处理
	IsAsync() bool
}

// BaseListener 提供基础监听器实现
// 可作为自定义监听器的嵌入类型
// BaseListener 提供基础监听器实现
// T 是事件参数类型
// 可作为自定义监听器的嵌入类型
type BaseListener[T any] struct {
	order      int
	async      bool
	eventTypes []EventType
}

// SupportedEvents 实现EventListener接口
func (l *BaseListener[T]) SupportedEvents() []EventType {
	return l.eventTypes
}

// NewBaseListener 创建新的基础监听器实例
// order 执行顺序，数值越小优先级越高
// async 是否异步执行
// eventTypes 支持的事件类型列表
// NewBaseListener 创建新的基础监听器实例
// T 是事件参数类型
// order 执行顺序，数值越小优先级越高
// async 是否异步执行
// eventTypes 支持的事件类型列表
func NewBaseListener[T any](order int, async bool, eventTypes ...EventType) *BaseListener[T] {
	return &BaseListener[T]{
		order:      order,
		async:      async,
		eventTypes: eventTypes,
	}
}

// SupportsEvent 实现EventListener接口
func (l *BaseListener[T]) SupportsEvent(eventType EventType) bool {
	for _, t := range l.eventTypes {
		if t == eventType {
			return true
		}
	}
	return false
}

// Order 实现EventListener接口
func (l *BaseListener[T]) Order() int {
	return l.order
}

// IsAsync 实现EventListener接口
func (l *BaseListener[T]) IsAsync() bool {
	return l.async
}

// OnEvent 基础实现为空，需要子类重写
func (l *BaseListener[T]) OnEvent(ctx context.Context, event Event[T]) error {
	return nil
}

// EventListenerFunc 函数式监听器
// 允许使用函数作为监听器
// EventListenerFunc 函数式监听器
// T 是事件参数类型
// 允许使用函数作为监听器
// EventListenerFunc 函数式监听器
// T 是事件参数类型
// 允许使用函数作为监听器
type EventListenerFunc[T any] struct {
	BaseListener[T]
	handler func(ctx context.Context, event Event[T]) error
}

// NewEventListenerFunc 创建新的函数式监听器
// T 是事件参数类型
func NewEventListenerFunc[T any](order int, async bool, handler func(ctx context.Context, event Event[T]) error, eventTypes ...EventType) *EventListenerFunc[T] {
	return &EventListenerFunc[T]{
		BaseListener: *NewBaseListener[T](order, async, eventTypes...),
		handler:      handler,
	}
}

// OnEvent 实现EventListener接口
func (f *EventListenerFunc[T]) OnEvent(ctx context.Context, event Event[T]) error {
	return f.handler(ctx, event)
}
