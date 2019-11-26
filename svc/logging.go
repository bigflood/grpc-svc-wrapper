package svc

import (
	"context"
	"log"

	"google.golang.org/grpc"
)

func LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		reply, err := handler(ctx, req)

		log.Println("unary req:", req, "info:", info, "reply:", reply, "err:", err)

		return reply, err
	}
}

func LoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, &ServerStreamLogger{ServerStream: stream})
		log.Println("stream srv:", srv, "stream:", stream, "info:", info, "err:", err)
		return err
	}
}

type ServerStreamLogger struct {
	grpc.ServerStream
}

func (s *ServerStreamLogger) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	log.Println("SendMsg:", m, "err:", err)
	return err
}

func (s *ServerStreamLogger) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	log.Println("RecvMsg:", m, "err:", err)
	return err
}
