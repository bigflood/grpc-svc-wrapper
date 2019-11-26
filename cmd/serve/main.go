package main

import (
	"log"
	"net"
	"time"

	"github.com/bigflood/grpc_svc_wrapper/pb"
	"github.com/bigflood/grpc_svc_wrapper/svc"
	"github.com/bigflood/grpc_svc_wrapper/wrapper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	MaxSvcIdleTime = 10 * time.Second
)

func main() {
	const addr = ":9992"

	svr := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			//MaxConnectionIdle:     300 * time.Second,
			//MaxConnectionAge:      5 * time.Minute,
			//MaxConnectionAgeGrace: 30 * time.Second,
			MaxConnectionAge:      30 * time.Second,
			MaxConnectionAgeGrace: 10 * time.Second,
			// pings the client to see if the transport is still alive.
			Time:    time.Minute,
			Timeout: 20 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			// 클라이언트가 grpc ping을 30초 간격까지 보낼 수 있음:
			MinTime: 30 * time.Second,
			// 스트림이 없는 커넥션에도 grpc ping을 허용:
			PermitWithoutStream: true,
		}),
	)

	w := wrapper.Wrapper{
		Desc:           pb.GetHelloServiceDesc(),
		MaxSvcIdleTime: MaxSvcIdleTime,
		ServiceFactory: func(name string) *wrapper.Svc {
			return &wrapper.Svc{
				Srv:               &svc.Service{Name: name},
				UnaryInterceptor:  svc.LoggingUnaryInterceptor(),
				StreamInterceptor: svc.LoggingStreamInterceptor(),
			}
		},
	}
	w.RegisterService(svr)

	//hello := &svc.Service{}
	//pb.RegisterHelloServer(svr, hello)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("serve:", addr)

	go func() {
		for {
			time.Sleep(MaxSvcIdleTime / 10)
			w.RemoveIdleSvc()
		}
	}()

	if err := svr.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
