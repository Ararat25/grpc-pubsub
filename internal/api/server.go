package api

import (
	"context"
	"errors"
	"fmt"
	pubsub "github.com/Ararat25/grpc-pubsub/internal/proto"
	"github.com/Ararat25/grpc-pubsub/subpub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
)

type service struct {
	pubsub.UnimplementedPubSubServer
	subPub subpub.SubPub
}

func NewService(subPub subpub.SubPub) pubsub.PubSubServer {
	return &service{
		subPub: subPub,
	}
}

func (s *service) Subscribe(r *pubsub.SubscribeRequest, stream grpc.ServerStreamingServer[pubsub.Event]) error {
	subject := r.GetKey()
	if subject == "" {
		return errors.New("the request is missing a key")
	}

	if s.subPub == nil {
		return errors.New("subpub service is not initialized")
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
		return err
	}

	// Отписываемся после завершения работы функции (например, при отключении клиента).
	defer func() {
		subscribe.Unsubscribe()
		log.Printf("client unsubscribed from subject: %s", subject)
	}()

	// Блокируем выполнение, пока клиент не отключится.
	<-ctx.Done()
	log.Printf("client context canceled for subject: %s", subject)
	return ctx.Err()
}

func (s *service) Publish(ctx context.Context, r *pubsub.PublishRequest) (*emptypb.Empty, error) {
	subject := r.GetKey()
	if subject == "" {
		return nil, status.Error(codes.InvalidArgument, "missing key")
	}

	if s.subPub == nil {
		return nil, status.Error(codes.Internal, "subpub service not initialized")
	}

	event := &pubsub.Event{
		Data: r.GetData(),
	}

	err := s.subPub.Publish(subject, event)
	if err != nil {
		return nil, status.Error(codes.Unknown, fmt.Sprintf("failed to publish event: %v", err))
	}

	return &emptypb.Empty{}, nil
}
