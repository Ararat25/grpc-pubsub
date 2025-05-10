package logServer

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"time"
)

func LoggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("[Unary] %s | Duration: %s | Error: %v", info.FullMethod, time.Since(start), err)
	return resp, err
}

func LoggingStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	err := handler(srv, ss)
	log.Printf("[Stream] %s | Duration: %s | Error: %v", info.FullMethod, time.Since(start), err)
	return err
}
