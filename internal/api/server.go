package api

import (
	"context"
	"fmt"
	"log"

	pubsub "github.com/Ararat25/grpc-pubsub/internal/proto"
	"github.com/Ararat25/grpc-pubsub/subpub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// server - структура для работы с grpc-сервером
type server struct {
	pubsub.UnimplementedPubSubServer
	subPub subpub.SubPub // объект структуры subpub.SubPub для работы с пакетом subpub
}

// NewServer создает и возвращает объект pubsub.PubSubServer
func NewServer(subPub subpub.SubPub) pubsub.PubSubServer {
	return &server{
		subPub: subPub,
	}
}

// Subscribe подписывает клиента на определенную тему
func (s *server) Subscribe(r *pubsub.SubscribeRequest, stream grpc.ServerStreamingServer[pubsub.Event]) error {
	subject := r.GetKey()
	if subject == "" {
		return status.Error(codes.InvalidArgument, "missing key")
	}

	if s.subPub == nil {
		return status.Error(codes.Internal, "subpub server not initialized")
	}

	ctx := stream.Context()
	log.Printf("client subscribed to subject: %s", subject)

	subscribe, err := s.subPub.Subscribe(subject, func(msg interface{}) {
		select {
		case <-ctx.Done():
			return
		default:
			event, ok := msg.(*pubsub.Event)
			if ok {
				err := stream.Send(event)
				if err != nil {
					log.Printf("failed to send event to client: %v", err)
				}
			}
		}
	})
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to subscribe client: %v", err))
	}

	// Отписываемся после завершения работы функции (при отключении клиента).
	defer func() {
		subscribe.Unsubscribe()
		log.Printf("client unsubscribed from subject: %s", subject)
	}()

	// Блокируем выполнение, пока клиент не отключится.
	<-ctx.Done()
	log.Printf("client context canceled for subject: %s", subject)
	return nil
}

// Publish публикует в теме данные
func (s *server) Publish(ctx context.Context, r *pubsub.PublishRequest) (*emptypb.Empty, error) {
	subject := r.GetKey()
	if subject == "" {
		return nil, status.Error(codes.InvalidArgument, "missing key")
	}

	if s.subPub == nil {
		return nil, status.Error(codes.Internal, "subpub server not initialized")
	}

	event := &pubsub.Event{
		Data: r.GetData(),
	}

	// Проверяем, не отменён ли контекст перед публикацией
	select {
	case <-ctx.Done():
		return nil, status.Error(codes.Canceled, "request canceled by client")
	default:
	}

	err := s.subPub.Publish(subject, event)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to publish event: %v", err))
	}

	return &emptypb.Empty{}, nil
}
