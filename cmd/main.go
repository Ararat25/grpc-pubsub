package main

import (
	"github.com/Ararat25/grpc-pubsub/internal/api"
	"github.com/Ararat25/grpc-pubsub/internal/logServer"
	pubsub "github.com/Ararat25/grpc-pubsub/internal/proto"
	"github.com/Ararat25/grpc-pubsub/subpub"
	"google.golang.org/grpc"
	"log"
	"net"
)

const port = ":50051"

func main() {
	run()
}

func run() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logServer.LoggingUnaryInterceptor),
		grpc.ChainStreamInterceptor(logServer.LoggingStreamInterceptor),
	)

	subpubService := subpub.NewSubPub()

	srv := api.NewService(subpubService)

	pubsub.RegisterPubSubServer(grpcServer, srv)

	log.Printf("gRPC server started on %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
