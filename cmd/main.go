package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ararat25/grpc-pubsub/config"
	"github.com/Ararat25/grpc-pubsub/internal/api"
	"github.com/Ararat25/grpc-pubsub/internal/logServer"
	pubsub "github.com/Ararat25/grpc-pubsub/internal/proto"
	"github.com/Ararat25/grpc-pubsub/subpub"
	"google.golang.org/grpc"
)

func main() {
	conf, err := config.NewConfig(config.FetchConfigPath())
	if err != nil {
		log.Fatalf("failed to loading config file: %v", err)
	}

	runServer(conf)
}

// runServer запускает grpc-сервер и обрабатку для graceful shutdown
func runServer(conf *config.Config) {
	port := fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logServer.LoggingUnaryInterceptor),
		grpc.ChainStreamInterceptor(logServer.LoggingStreamInterceptor),
	)

	subpubService := subpub.NewSubPub()
	srv := api.NewServer(subpubService)

	pubsub.RegisterPubSubServer(grpcServer, srv)

	stop := make(chan os.Signal)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Запуск gRPC сервера в отдельной горутине
	go func() {
		log.Printf("gRPC server started on %s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-stop
	log.Println("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), conf.Server.Timeout)
	defer cancel()

	err = subpubService.Close(ctx)
	if err != nil {
		log.Printf("failed attempt to close the subpub bus: %v", err)
	}

	grpcServer.Stop()
	log.Println("server stopped gracefully")
}
