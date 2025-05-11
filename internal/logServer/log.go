package logServer

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

// LoggingUnaryInterceptor - interceptor для логирования одиночных gRPC-запросов на сервере
func LoggingUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("[Unary] %s | Duration: %s | Error: %v", info.FullMethod, time.Since(start), err)
	return resp, err
}

// LoggingStreamInterceptor - interceptor для логирования потоковых gRPC-запросов на сервере
func LoggingStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	err := handler(srv, ss)
	log.Printf("[Stream] %s | Duration: %s | Error: %v", info.FullMethod, time.Since(start), err)
	return err
}
