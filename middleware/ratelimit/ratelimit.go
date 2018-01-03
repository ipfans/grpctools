package ratelimit

import (
	"go.uber.org/ratelimit"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// UnaryServerInterceptor returns a new unary server interceptor for lucky-bucket ratelimit.
func UnaryServerInterceptor(rate int) grpc.UnaryServerInterceptor {
	token := ratelimit.New(rate)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		token.Take()
		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a new streaming server interceptor for lucky-bucket ratelimit.
func StreamServerInterceptor(rate int) grpc.StreamServerInterceptor {
	token := ratelimit.New(rate)
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		token.Take()
		return handler(srv, stream)
	}
}
