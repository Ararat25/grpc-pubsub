package main

import (
	"context"
	"fmt"
	"github.com/Ararat25/grpc-pubsub/config"
	"github.com/Ararat25/grpc-pubsub/internal/api"
	"github.com/Ararat25/grpc-pubsub/internal/logServer"
	pubsub "github.com/Ararat25/grpc-pubsub/internal/proto"
	"github.com/Ararat25/grpc-pubsub/subpub"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
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
	port := fmt.Sprintf(":%d", conf.Server.Port)

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

	// Завершение с тайм-аутом
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown timeout reached, forcing stop")
		grpcServer.Stop()
	case <-stopped:
		log.Println("server stopped gracefully")
	}
}
