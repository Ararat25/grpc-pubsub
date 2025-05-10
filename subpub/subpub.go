package subpub

import (
	"context"
	"errors"
	"sync"
)

// MessageHandler is a callback function that processes messages delivered to subscribers.
type MessageHandler func(msg interface{})

type Subscription interface {
	// Unsubscribe will remove interest in the current subject subscription is for.
	Unsubscribe()
}

type SubPub interface {
	// Subscribe creates an asynchronous queue subscriber on the given subject.
	Subscribe(subject string, cb MessageHandler) (Subscription, error)

	// Publish publishes the msg argument to the given subject.
	Publish(subject string, msg interface{}) error

	// Close will shutdown sub-pub system.
	// May be blocked by data delivery until the context is canceled.
	Close(ctx context.Context) error
}

type subPub struct {
	subscribers map[string][]*subscription
	mu          sync.RWMutex
	closed      bool
}

func NewSubPub() SubPub {
	return &subPub{
		subscribers: make(map[string][]*subscription),
	}
}

func (s *subPub) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	if cb == nil {
		return nil, errors.New("callback function cannot be nil")
	}

	if s.closed {
		return nil, errors.New("subPub system is closed")
	}

	subsc := &subscription{
		handler: cb,
		msgCh:   make(chan any),
		sb:      s,
		subject: subject,
	}

	s.mu.Lock()
	s.subscribers[subject] = append(s.subscribers[subject], subsc)
	s.mu.Unlock()

	go func() {
		for msg := range subsc.msgCh {
			subsc.handler(msg)
		}
	}()

	return subsc, nil
}

func (s *subPub) Publish(subject string, msg interface{}) error {
	if s.closed {
		return errors.New("subPub system is closed")
	}

	s.mu.RLock()
	for _, sub := range s.subscribers[subject] { // рассылка сообщений по подпискам
		sub.msgCh <- msg
	}
	s.mu.RUnlock()

	return nil
}

func (s *subPub) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if s.closed {
		return nil
	}

	s.closed = true

	var wg sync.WaitGroup
	for _, subs := range s.subscribers {
		for _, sub := range subs {
			wg.Add(1)
			go func(s *subscription) {
				s.Unsubscribe()
				wg.Done()
			}(sub)
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *subPub) unsubscribe(subject string, sub *subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, subs := range s.subscribers[subject] {
		if subs == sub {
			s.subscribers[subject] = append(s.subscribers[subject][:i], s.subscribers[subject][i+1:]...)
			break
		}
	}
}
