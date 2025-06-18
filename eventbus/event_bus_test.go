package eventbus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// 测试用事件类型常量
const (
	EventTypeTestEvent            EventType = "test.event"
	EventTypeTestCancellableEvent EventType = "test.cancellable.event"
	EventTypeTestSyncEvent        EventType = "test.sync.event"
	EventTypeTestAsyncEvent       EventType = "test.async.event"
)

type OrderEvent struct {
	UserId int64
}

func TestOrderEventEvent(t *testing.T) {
	// 创建bus
	bus := NewDefaultEventBus[OrderEvent]()
	// 创建监听者
	createOrderListener := NewEventListenerFunc(10, true, func(ctx context.Context, event Event[OrderEvent]) error {
		fmt.Println(event.Params().UserId)
		return nil
	}, EventTypeTestEvent)
	bus.RegisterListener(createOrderListener)
	// 发布事件
	baseEvent := NewBaseEvent(EventTypeTestEvent, OrderEvent{
		UserId: 111,
	})
	bus.Publish(context.Background(), baseEvent)

}

// TestBaseEvent 测试基础事件功能
func TestBaseEvent(t *testing.T) {
	// 测试普通事件
	baseEvent := NewBaseEvent[map[string]interface{}](EventTypeTestEvent, nil)
	if baseEvent.Type() != EventTypeTestEvent {
		t.Errorf("BaseEvent.Type() = %q, want %q", baseEvent.Type(), "test.event")
	}
	if baseEvent.Timestamp().IsZero() {
		t.Error("BaseEvent.Timestamp() should not be zero")
	}

	// 测试可取消事件
	cancellableEvent := NewBaseCancellableEvent[map[string]interface{}](EventTypeTestCancellableEvent, nil)
	if cancellableEvent.IsCancelled() {
		t.Error("New cancellable event should not be cancelled")
	}
	cancellableEvent.Cancel()
	if !cancellableEvent.IsCancelled() {
		t.Error("Cancellable event should be cancelled after Cancel()")
	}
}

// TestEventListener 测试监听器功能
func TestEventListener(t *testing.T) {
	// 测试基础监听器
	baseListener := NewBaseListener[map[string]interface{}](10, false, EventTypeTestEvent)
	if !baseListener.SupportsEvent(EventTypeTestEvent) {
		t.Error("BaseListener should support 'test.event'")
	}
	if baseListener.SupportsEvent(EventType("other.event")) {
		t.Error("BaseListener should not support 'other.event'")
	}
	if baseListener.Order() != 10 {
		t.Errorf("BaseListener.Order() = %d, want 10", baseListener.Order())
	}
	if baseListener.IsAsync() {
		t.Error("BaseListener should not be async")
	}

	// 测试函数式监听器
	called := false
	funcListener := NewEventListenerFunc[map[string]interface{}](5, true, func(ctx context.Context, event Event[map[string]interface{}]) error {
		called = true
		return nil
	}, EventTypeTestEvent)

	if err := funcListener.OnEvent(context.Background(), NewBaseEvent[map[string]interface{}](EventTypeTestEvent, nil)); err != nil {
		t.Errorf("EventListenerFunc.OnEvent() error = %v", err)
	}
	if !called {
		t.Error("EventListenerFunc handler was not called")
	}
}

// TestEventBus_RegisterAndUnregister 测试监听器注册和注销
func TestEventBus_RegisterAndUnregister(t *testing.T) {
	bus := NewDefaultEventBus[map[string]interface{}]()
	defer bus.Shutdown(context.Background())

	listener := NewBaseListener[map[string]interface{}](1, false, EventTypeTestEvent)

	// 测试注册
	bus.RegisterListener(listener)

	// 验证注册成功
	bus.mu.RLock()
	listeners, ok := bus.listeners[EventTypeTestEvent]
	bus.mu.RUnlock()

	if !ok || len(listeners) != 1 || listeners[0] != listener {
		t.Error("Failed to register listener")
	}

	// 测试注销
	bus.UnregisterListener(listener)

	// 验证注销成功
	bus.mu.RLock()
	listeners, ok = bus.listeners[EventTypeTestEvent]
	bus.mu.RUnlock()

	if ok || listeners != nil {
		t.Error("Failed to unregister listener")
	}
}

// TestEventBus_Publish_Sync 测试同步事件发布
func TestEventBus_Publish_Sync(t *testing.T) {
	bus := NewDefaultEventBus[map[string]interface{}]()
	defer bus.Shutdown(context.Background())

	// 记录监听器调用顺序
	callOrder := make([]int, 0)
	var mu sync.Mutex

	// 创建3个不同优先级的监听器
	listener1 := NewEventListenerFunc(1, false, func(ctx context.Context, event Event[map[string]interface{}]) error {
		mu.Lock()
		callOrder = append(callOrder, 1)
		mu.Unlock()
		return nil
	}, EventTypeTestSyncEvent)

	listener2 := NewEventListenerFunc(2, false, func(ctx context.Context, event Event[map[string]interface{}]) error {
		mu.Lock()
		callOrder = append(callOrder, 2)
		mu.Unlock()
		return nil
	}, EventTypeTestSyncEvent)

	listener3 := NewEventListenerFunc[map[string]interface{}](0, false, func(ctx context.Context, event Event[map[string]interface{}]) error {
		mu.Lock()
		callOrder = append(callOrder, 3)
		baseEvent, ok := event.(*BaseEvent[map[string]interface{}])
		if !ok {
			t.Errorf("event is not of type *BaseEvent[map[string]interface{}]")
			return nil
		}
		params := baseEvent.Params()
		t.Logf("listener3 params: %v", params)
		mu.Unlock()
		return nil
	}, EventTypeTestSyncEvent)

	// 注册监听器
	bus.RegisterListener(listener1)
	bus.RegisterListener(listener2)
	bus.RegisterListener(listener3)

	// 发布事件
	errors := bus.Publish(context.Background(), NewBaseEvent[map[string]interface{}](EventTypeTestSyncEvent, map[string]interface{}{
		"key": "value",
		"123": "123123",
	}))

	// 检查错误
	if len(errors) > 0 {
		t.Errorf("Publish returned errors: %v", errors)
	}

	// 检查调用顺序 (应该按Order升序: 3(0), 1(1), 2(2))
	if len(callOrder) != 3 {
		t.Fatalf("Expected 3 listeners to be called, got %d", len(callOrder))
	}
	if callOrder[0] != 3 || callOrder[1] != 1 || callOrder[2] != 2 {
		t.Errorf("Unexpected call order: %v, want [3, 1, 2]", callOrder)
	}
}

// TestEventBus_Publish_Async 测试异步事件发布
func TestEventBus_RegisteredEvents(t *testing.T) {
	bus := NewDefaultEventBus[string]()
	defer bus.Shutdown(context.Background())

	// 注册不同类型的监听器
	listener1 := NewBaseListener[string](0, false, EventTypeTestEvent)
	listener2 := NewBaseListener[string](0, false, EventTypeTestCancellableEvent)
	bus.RegisterListener(listener1)
	bus.RegisterListener(listener2)

	// 验证已注册事件
	registeredEvents := bus.RegisteredEvents()
	fmt.Println(registeredEvents)

	// 移除一个监听器
	bus.UnregisterListener(listener1)

	// 验证剩余已注册事件
	registeredEvents = bus.RegisteredEvents()
	fmt.Println(registeredEvents)
}

func TestEventBus_Publish_Async(t *testing.T) {
	bus := NewDefaultEventBus[string]()
	defer bus.Shutdown(context.Background())

	// 注册异步监听器
	received := make(chan string, 1)
	listener := NewEventListenerFunc(0, true, func(ctx context.Context, event Event[string]) error {
		received <- event.Params()
		return nil
	}, EventTypeTestAsyncEvent)
	bus.RegisterListener(listener)

	// 发布事件
	event := NewBaseEvent(EventTypeTestAsyncEvent, "async test data")
	errors := bus.Publish(context.Background(), event)
	fmt.Println(errors)

	// 验证监听器接收到事件
	select {
	case data := <-received:
		fmt.Println(data)
	case <-time.After(time.Second * 2):
		t.Error("异步事件处理超时")
	}
}

// TestEventBus_Publish_Cancellable 测试可取消事件
func TestEventBus_Publish_Cancellable(t *testing.T) {
	bus := NewDefaultEventBus[map[string]interface{}]()
	defer bus.Shutdown(context.Background())

	callCount := 0

	// 第一个监听器取消事件
	listener1 := NewEventListenerFunc(1, false, func(ctx context.Context, event Event[map[string]interface{}]) error {
		callCount++
		if cancellableEvent, ok := event.(CancellableEvent[map[string]interface{}]); ok {
			cancellableEvent.Cancel()
		}
		return nil
	}, "test.cancellable.event")

	// 第二个监听器不应该被调用
	listener2 := NewEventListenerFunc(2, false, func(ctx context.Context, event Event[map[string]interface{}]) error {
		callCount++
		return nil
	}, "test.cancellable.event")

	bus.RegisterListener(listener1)
	bus.RegisterListener(listener2)

	// 发布可取消事件
	event := NewBaseCancellableEvent[map[string]interface{}]("test.cancellable.event", nil)
	errors := bus.Publish(context.Background(), event)

	if len(errors) > 0 {
		t.Errorf("Publish returned errors: %v", errors)
	}

	// 应该只调用第一个监听器
	if callCount != 1 {
		t.Errorf("Expected 1 listener call, got %d", callCount)
	}

	// 确认事件已取消
	if !event.IsCancelled() {
		t.Error("Event should be cancelled")
	}
}

// TestEventBus_ErrorHandling 测试错误处理
func TestEventBus_ErrorHandling(t *testing.T) {
	bus := NewDefaultEventBus[map[string]interface{}]()
	defer bus.Shutdown(context.Background())

	expectedErr := errors.New("listener error")

	// 创建返回错误的监听器
	listener := NewEventListenerFunc(1, false, func(ctx context.Context, event Event[map[string]interface{}]) error {
		return expectedErr
	}, "test.error.event")

	bus.RegisterListener(listener)

	// 发布事件
	errors := bus.Publish(context.Background(), NewBaseEvent[map[string]interface{}]("test.error.event", nil))

	// 检查错误是否被捕获
	if len(errors) != 1 || errors[0] != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, errors)
	}
}

// TestEventBus_Shutdown 测试事件总线关闭
func TestEventBus_Shutdown(t *testing.T) {
	bus := NewDefaultEventBus[map[string]interface{}]()

	// 创建长时间运行的异步监听器
	listener := NewEventListenerFunc(1, true, func(ctx context.Context, event Event[map[string]interface{}]) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}, "test.shutdown.event")

	bus.RegisterListener(listener)

	// 发布事件
	bus.Publish(context.Background(), NewBaseEvent[map[string]interface{}]("test.shutdown.event", nil))

	// 关闭事件总线并设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := bus.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}

	// 尝试发布到已关闭的总线
	errors := bus.Publish(context.Background(), NewBaseEvent[map[string]interface{}]("test.shutdown.event", nil))
	if len(errors) != 1 || errors[0].Error() != "event bus is closed" {
		t.Errorf("Expected 'event bus is closed' error, got %v", errors)
	}
}

// TestEventBus_Concurrent 测试并发安全性
func TestEventBus_Concurrent(t *testing.T) {
	bus := NewDefaultEventBus[map[string]interface{}]()
	defer bus.Shutdown(context.Background())

	var wg sync.WaitGroup
	listenerCount := 100
	publishCount := 1000

	// 并发注册监听器
	for i := 0; i < listenerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			listener := NewEventListenerFunc(id, false, func(ctx context.Context, event Event[map[string]interface{}]) error {
				return nil
			}, "test.concurrent.event")
			bus.RegisterListener(listener)
		}(i)
	}

	// 等待所有监听器注册完成
	wg.Wait()

	// 并发发布事件
	for i := 0; i < publishCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(context.Background(), NewBaseEvent[map[string]interface{}]("test.concurrent.event", nil))
		}()
	}

	// 等待所有发布完成
	wg.Wait()

	// 验证监听器数量
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	if len(bus.listeners["test.concurrent.event"]) != listenerCount {
		t.Errorf("Expected %d listeners, got %d", listenerCount, len(bus.listeners["test.concurrent.event"]))
	}
}

// 自定义监听器实现，用于处理字符串类型事件
type StringEventListener struct {
	BaseListener[string]
	handler func(ctx context.Context, event Event[string]) error
}

// NewStringEventListener 创建字符串事件监听器
func NewStringEventListener(order int, async bool, handler func(ctx context.Context, event Event[string]) error, eventTypes ...EventType) *StringEventListener {
	return &StringEventListener{
		BaseListener: *NewBaseListener[string](order, async, eventTypes...),
		handler:      handler,
	}
}

// OnEvent 重写事件处理方法
func (l *StringEventListener) OnEvent(ctx context.Context, event Event[string]) error {
	if l.handler == nil {
		return fmt.Errorf("handler is not set")
	}
	return l.handler(ctx, event)
}

func TestStringEventListener_OnEvent(t *testing.T) {
	// 创建测试事件类型
	testEventType := EventType("test.string.event")
	testData := "test payload"
	bus := NewDefaultEventBus[string]()
	defer bus.Shutdown(context.Background())

	// 创建监听器
	received := make(chan string, 1)
	listener := NewStringEventListener(0, false, func(ctx context.Context, event Event[string]) error {
		received <- event.Params()
		return nil
	}, testEventType)
	bus.RegisterListener(listener)
	// 创建事件
	event := NewBaseEvent(testEventType, testData)
	// 发布事件
	bus.Publish(context.Background(), event)

	// 验证结果
	select {
	case data := <-received:
		if data != testData {
			t.Errorf("expected %q, got %q", testData, data)
		}
	case <-time.After(time.Second):
		t.Error("did not receive event data")
	}
}
