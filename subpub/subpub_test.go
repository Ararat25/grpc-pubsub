package subpub

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubscribeAndPublish проверяет, что сообщение успешно доставляется подписчику после публикации
func TestSubscribeAndPublish(t *testing.T) {
	sp := NewSubPub()
	var wg sync.WaitGroup
	wg.Add(1)

	var receivedMsg interface{}
	sub, err := sp.Subscribe("topic1", func(msg interface{}) {
		receivedMsg = msg
		wg.Done()
	})
	require.NoError(t, err)
	require.NotNil(t, sub)

	err = sp.Publish("topic1", "hello")
	require.NoError(t, err)

	waitDone(t, &wg)
	assert.Equal(t, "hello", receivedMsg)
}

// TestMultipleSubscribersOnOneSubject проверяет, что несколько подписчиков получают одно и то же сообщение и могут отписаться
func TestMultipleSubscribersOnOneSubject(t *testing.T) {
	sp := NewSubPub()
	var wg sync.WaitGroup
	count := 3
	wg.Add(count)

	subs := make([]Subscription, 0, count)
	for i := 0; i < count; i++ {
		sub, err := sp.Subscribe("shared", func(msg interface{}) {
			wg.Done()
		})
		require.NoError(t, err)
		subs = append(subs, sub)
	}

	err := sp.Publish("shared", "broadcast")
	require.NoError(t, err)
	waitDone(t, &wg)

	for _, sub := range subs {
		sub.Unsubscribe()
	}
}

// TestSlowSubscriber проверяет, что медленный обработчик не задерживает вызов других подписчиков
func TestSlowSubscriber(t *testing.T) {
	sp := NewSubPub()
	var fastWg sync.WaitGroup
	fastWg.Add(1)

	_, err := sp.Subscribe("topic", func(msg interface{}) {
		fastWg.Done()
	})
	require.NoError(t, err)

	_, err = sp.Subscribe("topic", func(msg interface{}) {
		time.Sleep(100 * time.Millisecond) // медленный обработчик
	})
	require.NoError(t, err)

	err = sp.Publish("topic", "data")
	require.NoError(t, err)

	waitDone(t, &fastWg)
}

// TestMessageOrderFIFO проверяет, что сообщения доставляются подписчику в порядке отправки (FIFO)
func TestMessageOrderFIFO(t *testing.T) {
	sp := NewSubPub()
	var mu sync.Mutex
	var received []string
	expected := []string{"one", "two", "three"}

	var wg sync.WaitGroup
	wg.Add(len(expected))

	_, err := sp.Subscribe("seq", func(msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, msg.(string))
		wg.Done()
	})
	require.NoError(t, err)

	for _, msg := range expected {
		require.NoError(t, sp.Publish("seq", msg))
	}
	waitDone(t, &wg)

	mu.Lock()
	assert.Equal(t, expected, received)
	mu.Unlock()
}

// TestNoGoroutineLeaks проверяет, что после отписки и закрытия не остаются работающие горутины
func TestNoGoroutineLeaks(t *testing.T) {
	before := runtime.NumGoroutine()

	sp := NewSubPub()
	sub, _ := sp.Subscribe("topic", func(msg interface{}) {})
	sub.Unsubscribe()

	err := sp.Close(context.Background())
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // дать время завершиться горутинам

	after := runtime.NumGoroutine()
	assert.LessOrEqual(t, after, before, "should not leak goroutines")
}

// TestSubscribeWithNilHandler проверяет, что подписка с nil-обработчиком возвращает ошибку
func TestSubscribeWithNilHandler(t *testing.T) {
	sp := NewSubPub()
	sub, err := sp.Subscribe("topic", nil)
	assert.Error(t, err)
	assert.Nil(t, sub)
}

// TestPublishAfterClose проверяет, что публикация после закрытия системы возвращает ошибку
func TestPublishAfterClose(t *testing.T) {
	sp := NewSubPub()
	require.NoError(t, sp.Close(context.Background()))

	err := sp.Publish("topic", "msg")
	assert.Error(t, err)
	assert.EqualError(t, err, "subPub system is closed")
}

// TestSubscribeAfterClose проверяет, что подписка после закрытия системы возвращает ошибку
func TestSubscribeAfterClose(t *testing.T) {
	sp := NewSubPub()
	require.NoError(t, sp.Close(context.Background()))

	sub, err := sp.Subscribe("topic", func(msg interface{}) {})
	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.EqualError(t, err, "subPub system is closed")
}

// TestCloseWithTimeout проверяет, что метод Close корректно обрабатывает таймаут контекста
func TestCloseWithTimeout(t *testing.T) {
	sp := NewSubPub()

	// блокирующий обработчик
	_, err := sp.Subscribe("topic", func(msg interface{}) {
		time.Sleep(100 * time.Millisecond)
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err = sp.Close(ctx)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}

// waitDone — вспомогательная функция, ожидающая завершения WaitGroup или таймаута
func waitDone(t *testing.T, wg *sync.WaitGroup) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for WaitGroup")
	}
}
