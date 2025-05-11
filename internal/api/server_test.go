package api

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	pubsub "github.com/Ararat25/grpc-pubsub/internal/proto"
	"github.com/Ararat25/grpc-pubsub/subpub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Mocks

// MockSubPub реализует интерфейс subpub.PubSub для целей тестирования.
type MockSubPub struct {
	mock.Mock
}

func (m *MockSubPub) Publish(subject string, msg interface{}) error {
	args := m.Called(subject, msg)
	return args.Error(0)
}

func (m *MockSubPub) Subscribe(subject string, handler subpub.MessageHandler) (subpub.Subscription, error) {
	args := m.Called(subject, handler)
	return args.Get(0).(subpub.Subscription), args.Error(1)
}

func (m *MockSubPub) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockSubscription реализует интерфейс subpub.Subscription для целей тестирования.
type MockSubscription struct {
	mock.Mock
}

func (m *MockSubscription) Unsubscribe() {
	m.Called()
}

// Mock gRPC Stream

// mockStream реализует grpc.ServerStream для имитации серверного стрима в тестах подписки.
type mockStream struct {
	ctx      context.Context
	received []*pubsub.Event
	mu       sync.Mutex
}

func (m *mockStream) Context() context.Context {
	return m.ctx
}

func (m *mockStream) Send(res *pubsub.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.received = append(m.received, res)
	return nil
}

func (m *mockStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockStream) SetTrailer(md metadata.MD)       {}
func (m *mockStream) SendMsg(msg any) error           { return nil }
func (m *mockStream) RecvMsg(msg any) error           { return nil }

// Tests

// TestPublish_Success проверяет успешную публикацию сообщения через PubSub.
func TestPublish_Success(t *testing.T) {
	mockPub := new(MockSubPub)
	mockPub.On("Publish", "test", mock.Anything).Return(nil)

	s := NewServer(mockPub)

	_, err := s.Publish(context.Background(), &pubsub.PublishRequest{
		Key:  "test",
		Data: "data",
	})

	assert.NoError(t, err)
	mockPub.AssertExpectations(t)
}

// TestPublish_MissingKey проверяет, что при отсутствии ключа публикации возвращается ошибка codes.InvalidArgument.
func TestPublish_MissingKey(t *testing.T) {
	s := NewServer(nil)

	_, err := s.Publish(context.Background(), &pubsub.PublishRequest{
		Key: "",
	})

	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// TestPublish_ContextCanceled проверяет, что при отмене контекста до вызова Publish возвращается ошибка codes.Canceled и Publish не вызывается.
func TestPublish_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockPub := new(MockSubPub)
	s := NewServer(mockPub)

	_, err := s.Publish(ctx, &pubsub.PublishRequest{
		Key:  "k",
		Data: "d",
	})

	assert.Equal(t, codes.Canceled, status.Code(err))
	// Проверка, что Publish НЕ вызывался
	mockPub.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}

// TestSubscribe_Success проверяет успешную подписку и получение сообщения через стрим.
func TestSubscribe_Success(t *testing.T) {
	mockPub := new(MockSubPub)
	mockSub := new(MockSubscription)

	ctx, cancel := context.WithCancel(context.Background())
	stream := &mockStream{ctx: ctx}

	mockSub.On("Unsubscribe").Return()

	mockPub.On("Subscribe", "abc", mock.AnythingOfType("subpub.MessageHandler")).
		Run(func(args mock.Arguments) {
			handler := args.Get(1).(subpub.MessageHandler)
			go func() {
				time.Sleep(50 * time.Millisecond)
				handler(&pubsub.Event{Data: "hello"})
				cancel()
			}()
		}).Return(mockSub, nil)

	s := NewServer(mockPub)

	err := s.Subscribe(&pubsub.SubscribeRequest{Key: "abc"}, stream)
	assert.NoError(t, err)

	assert.Len(t, stream.received, 1)
	assert.Equal(t, "hello", stream.received[0].Data)

	mockPub.AssertExpectations(t)
	mockSub.AssertExpectations(t)
}

// TestSubscribe_EmptyKey проверяет, что при отсутствии ключа подписки возвращается ошибка codes.InvalidArgument.
func TestSubscribe_EmptyKey(t *testing.T) {
	s := NewServer(nil)
	stream := &mockStream{ctx: context.Background()}

	err := s.Subscribe(&pubsub.SubscribeRequest{Key: ""}, stream)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

// TestSubscribe_SubscribeFails проверяет поведение, когда PubSub возвращает ошибку при попытке подписки.
func TestSubscribe_SubscribeFails(t *testing.T) {
	mockPub := new(MockSubPub)
	stream := &mockStream{ctx: context.Background()}

	mockPub.On("Subscribe", "topic", mock.Anything).Return((*MockSubscription)(nil), errors.New("fail"))

	s := NewServer(mockPub)

	err := s.Subscribe(&pubsub.SubscribeRequest{Key: "topic"}, stream)
	assert.Equal(t, codes.Internal, status.Code(err))

	mockPub.AssertExpectations(t)
}
