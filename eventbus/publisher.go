package eventbus

import (
	"context"
	"errors"
)

// Publisher 定义事件发布者接口
// Publisher 定义事件发布者接口
// T 是事件参数类型
type Publisher[T any] interface {
	// PublishEvent 发布事件
	// ctx 上下文
	// event 要发布的事件
	// 返回事件处理过程中产生的错误
	PublishEvent(ctx context.Context, event Event[T]) []error
}

// EventPublisher 事件发布者实现
// EventPublisher 事件发布者实现
type EventPublisher[T any] struct {
	bus EventBus[T]
}

// NewEventPublisher 创建新的事件发布者
// bus 事件总线实例
// NewEventPublisher 创建新的事件发布者
// bus 事件总线实例
func NewEventPublisher[T any](bus EventBus[T]) *EventPublisher[T] {
	return &EventPublisher[T]{
		bus: bus,
	}
}

// PublishEvent 实现Publisher接口
// PublishEvent 实现Publisher接口
func (p *EventPublisher[T]) PublishEvent(ctx context.Context, event Event[T]) []error {
	if p.bus == nil {
		return []error{errors.New("event bus is not initialized")}
	}
	return p.bus.Publish(ctx, event)
}

// CancellableEventPublisher 可取消事件发布者
// CancellableEventPublisher 可取消事件发布者
type CancellableEventPublisher[T any] struct {
	EventPublisher[T]
}

// NewCancellableEventPublisher 创建新的可取消事件发布者
// NewCancellableEventPublisher 创建新的可取消事件发布者
func NewCancellableEventPublisher[T any](bus EventBus[T]) *CancellableEventPublisher[T] {
	return &CancellableEventPublisher[T]{
		EventPublisher: *NewEventPublisher[T](bus),
	}
}

// PublishCancellableEvent 发布可取消事件
// 返回事件是否被取消以及处理错误
// PublishCancellableEvent 发布可取消事件
// 返回事件是否被取消以及处理错误
func (p *CancellableEventPublisher[T]) PublishCancellableEvent(ctx context.Context, event CancellableEvent[T]) (bool, []error) {
	errors := p.PublishEvent(ctx, event)
	return event.IsCancelled(), errors
}
