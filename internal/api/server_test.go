package api

import (
	"context"
	pubsub "github.com/Ararat25/grpc-pubsub/internal/proto"
	"github.com/Ararat25/grpc-pubsub/subpub"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

func startTestServer(t *testing.T) (pubsub.PubSubClient, func()) {
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	s := &server{subPub: subpub.NewSubPub()}
	pubsub.RegisterPubSubServer(grpcServer, s)

	go grpcServer.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)

	client := pubsub.NewPubSubClient(conn)

	cleanup := func() {
		grpcServer.GracefulStop()
		conn.Close()
	}

	return client, cleanup
}

func TestPubSub(t *testing.T) {
	client, cleanup := startTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := client.Subscribe(ctx, &pubsub.SubscribeRequest{Key: "topic"})
	require.NoError(t, err)

	_, err = client.Publish(ctx, &pubsub.PublishRequest{
		Key:  "topic",
		Data: "test message",
	})
	require.NoError(t, err)

	recv, err := stream.Recv()
	//require.NoError(t, err)
	require.Equal(t, "test message", recv.GetData())
}
