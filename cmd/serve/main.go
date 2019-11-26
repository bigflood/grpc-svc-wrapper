package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bigflood/grpc_svc_wrapper/pb"
	"github.com/bigflood/grpc_svc_wrapper/svc"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
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

	//svr.GetServiceInfo()
	w := Wrapper{
		Desc: pb.GetHelloServiceDesc(),
		ServiceFactory: func(name string) *Svc {
			return &Svc{
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

type Wrapper struct {
	Desc *grpc.ServiceDesc

	ServiceFactory func(name string) *Svc

	mutex    sync.Mutex
	services map[string]*Svc
}

func (w *Wrapper) RegisterService(svr *grpc.Server) {
	desc2 := grpc.ServiceDesc{
		ServiceName: w.Desc.ServiceName,
		HandlerType: (*interface{})(nil),
		Metadata:    "grpc_svc_wrapper",
	}

	for _, sd := range w.Desc.Methods {
		handler := sd.Handler
		desc2.Methods = append(desc2.Methods, grpc.MethodDesc{
			MethodName: sd.MethodName,
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				svcname := metautils.ExtractIncoming(ctx).Get("svcname")
				s := w.GetServiceFor(svcname)
				defer w.ReturnService(s)

				// interceptor와 s.UnaryInterceptor를 둘다 사용하려면 grpc_middleware.ChainUnaryServer를 사용해야함.
				return handler(s.Srv, ctx, dec, s.UnaryInterceptor)
			},
		})
	}

	for _, sd := range w.Desc.Streams {
		handler := sd.Handler
		desc2.Streams = append(desc2.Streams, grpc.StreamDesc{
			StreamName:    sd.StreamName,
			ServerStreams: sd.ServerStreams,
			ClientStreams: sd.ClientStreams,
			Handler: func(srv interface{}, stream grpc.ServerStream) error {
				svcname := metautils.ExtractIncoming(stream.Context()).Get("svcname")
				s := w.GetServiceFor(svcname)
				defer w.ReturnService(s)

				if s.StreamInterceptor != nil {
					info := &grpc.StreamServerInfo{
						FullMethod:     fmt.Sprintf("/%s/%s", w.Desc.ServiceName, sd.StreamName),
						IsClientStream: sd.ClientStreams,
						IsServerStream: sd.ServerStreams,
					}
					return s.StreamInterceptor(s.Srv, stream, info, handler)
				}

				return handler(s.Srv, stream)
			},
		})
	}

	svr.RegisterService(&desc2, w)
}

func (w *Wrapper) GetServiceFor(name string) *Svc {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	s := w.services[name]
	if s == nil {
		log.Println("create service for:", name)
		s = w.ServiceFactory(name)

		if w.services == nil {
			w.services = make(map[string]*Svc)
		}

		w.services[name] = s
	}

	s.lastAccessTime = time.Now()
	atomic.AddInt32(&s.inHandlerCount, 1)

	return s
}

func (w *Wrapper) ReturnService(s *Svc) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	s.lastAccessTime = time.Now()
	atomic.AddInt32(&s.inHandlerCount, -1)
}

func (w *Wrapper) RemoveIdleSvc() {
	timeBound := time.Now().Add(-MaxSvcIdleTime)

	w.mutex.Lock()
	defer w.mutex.Unlock()

	for key, svc := range w.services {
		if svc.inHandlerCount == 0 && svc.lastAccessTime.Before(timeBound) {
			log.Println("release service:", key)
			delete(w.services, key)
		}
	}
}

type Svc struct {
	Srv               interface{}
	UnaryInterceptor  grpc.UnaryServerInterceptor
	StreamInterceptor grpc.StreamServerInterceptor

	inHandlerCount int32
	lastAccessTime time.Time
}
