package eventbus

import (
	"context"
	"errors"
	"sort"
	"sync"
	"sync/atomic"
)

// EventBus 定义事件总线接口
// T 是事件参数类型
// 负责管理监听器注册和事件发布
type EventBus[T any] interface {
	// RegisterListener 注册事件监听器
	// listener 要注册的监听器实例
	RegisterListener(listener EventListener[T])

	// UnregisterListener 移除事件监听器
	// listener 要移除的监听器实例
	UnregisterListener(listener EventListener[T])

	// Publish 发布事件
	// ctx 上下文，用于传递超时和取消信号
	// event 要发布的事件
	// 返回所有监听器处理过程中产生的错误
	Publish(ctx context.Context, event Event[T]) []error

	// Shutdown 关闭事件总线
	// 等待所有异步事件处理完成
	Shutdown(ctx context.Context) error

	// RegisteredEvents 返回所有已注册的事件类型
	RegisteredEvents() []EventType
}

// DefaultEventBus 事件总线默认实现
// T 是事件参数类型
type DefaultEventBus[T any] struct {
	// 监听器映射: eventType -> []EventListener[T]
	listeners map[EventType][]EventListener[T]
	// 保护listeners的读写锁
	mu sync.RWMutex
	// 异步任务等待组
	asyncWG sync.WaitGroup
	// 关闭标志
	closed int32
}

// NewDefaultEventBus 创建新的事件总线实例
// T 是事件参数类型
func NewDefaultEventBus[T any]() *DefaultEventBus[T] {
	return &DefaultEventBus[T]{
		listeners: make(map[EventType][]EventListener[T]),
	}
}

// RegisterListener 实现EventBus接口
func (bus *DefaultEventBus[T]) RegisterListener(listener EventListener[T]) {
	if listener == nil {
		return
	}

	// 检查是否已关闭
	if atomic.LoadInt32(&bus.closed) == 1 {
		return
	}

	// 获取监听器支持的所有事件类型
	eventTypes := listener.SupportedEvents()
	if len(eventTypes) == 0 {
		return
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	for _, eventType := range eventTypes {
		// 添加监听器并排序
		bus.listeners[eventType] = append(bus.listeners[eventType], listener)
		bus.sortListeners(bus.listeners[eventType])
	}
}

// UnregisterListener 实现EventBus接口
func (bus *DefaultEventBus[T]) UnregisterListener(listener EventListener[T]) {
	if listener == nil {
		return
	}

	// 检查是否已关闭
	if atomic.LoadInt32(&bus.closed) == 1 {
		return
	}

	eventTypes := listener.SupportedEvents()
	if len(eventTypes) == 0 {
		return
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	for _, eventType := range eventTypes {
		listeners, ok := bus.listeners[eventType]
		if !ok {
			continue
		}

		// 查找并移除监听器
		for i, l := range listeners {
			if l == listener {
				// 保持切片连续性
				bus.listeners[eventType] = append(listeners[:i], listeners[i+1:]...)
				break
			}
		}

		// 如果该事件类型没有监听器了，删除映射
		if len(bus.listeners[eventType]) == 0 {
			delete(bus.listeners, eventType)
		}
	}
}

// Publish 实现EventBus接口
func (bus *DefaultEventBus[T]) Publish(ctx context.Context, event Event[T]) []error {
	if event == nil {
		return []error{errors.New("event cannot be nil")}
	}

	// 检查是否已关闭
	if atomic.LoadInt32(&bus.closed) == 1 {
		return []error{errors.New("event bus is closed")}
	}

	eventType := event.Type()

	// 读取模式获取监听器副本
	bus.mu.RLock()
	listeners := make([]EventListener[T], len(bus.listeners[eventType]))
	copy(listeners, bus.listeners[eventType])
	bus.mu.RUnlock()

	// 检查可取消事件
	if cancellableEvent, ok := event.(CancellableEvent[T]); ok && cancellableEvent.IsCancelled() {
		return nil
	}

	// 收集错误
	errors := make([]error, 0)
	// 异步错误通道
	asyncErrors := make(chan error, len(listeners))
	// 异步等待组
	var asyncWG sync.WaitGroup

	for _, listener := range listeners {
		// 检查监听器是否支持该事件（双重检查）
		if !listener.SupportsEvent(eventType) {
			continue
		}

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			errors = append(errors, ctx.Err())
			return errors
		default:
		}

		if listener.IsAsync() {
			// 异步执行
			asyncWG.Add(1)
			bus.asyncWG.Add(1)
			go func(l EventListener[T]) {
				defer asyncWG.Done()
				defer bus.asyncWG.Done()

				// 为每个异步任务创建子上下文
				childCtx, cancel := context.WithCancel(ctx)
				defer cancel()

				if err := l.OnEvent(childCtx, event); err != nil {
					asyncErrors <- err
				}
			}(listener)
		} else {
			// 同步执行
			if err := listener.OnEvent(ctx, event); err != nil {
				errors = append(errors, err)
			}

			// 检查事件是否已取消
			if cancellableEvent, ok := event.(CancellableEvent[T]); ok && cancellableEvent.IsCancelled() {
				break
			}
		}
	}

	// 等待所有异步任务完成并收集错误
	go func() {
		asyncWG.Wait()
		close(asyncErrors)
	}()

	for err := range asyncErrors {
		errors = append(errors, err)
	}

	return errors
}

// Shutdown 实现EventBus接口
func (bus *DefaultEventBus[T]) Shutdown(ctx context.Context) error {
	// 设置关闭标志
	if !atomic.CompareAndSwapInt32(&bus.closed, 0, 1) {
		return nil
	}

	// 创建一个通道来等待异步任务完成
	done := make(chan struct{})
	go func() {
		bus.asyncWG.Wait()
		close(done)
	}()

	// 等待异步任务完成或上下文取消
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RegisteredEvents 实现EventBus接口
func (bus *DefaultEventBus[T]) RegisteredEvents() []EventType {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// 收集所有已注册的事件类型
	events := make([]EventType, 0, len(bus.listeners))
	for eventType := range bus.listeners {
		events = append(events, eventType)
	}
	return events
}

// sortListeners 按Order对监听器进行排序
func (bus *DefaultEventBus[T]) sortListeners(listeners []EventListener[T]) {
	sort.Slice(listeners, func(i, j int) bool {
		return listeners[i].Order() < listeners[j].Order()
	})
}
