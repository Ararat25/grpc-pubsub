package subpub

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnsubscribe проверяет, что после отписки обработчик больше не получает сообщения.
func TestUnsubscribe(t *testing.T) {
	sp := NewSubPub()

	var mu sync.Mutex
	received := false
	sub, err := sp.Subscribe("topic", func(msg interface{}) {
		mu.Lock()
		received = true
		mu.Unlock()
	})
	require.NoError(t, err)
	sub.Unsubscribe()

	err = sp.Publish("topic", "data")
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	assert.False(t, received, "message should not be received after unsubscribe")
	mu.Unlock()
}
