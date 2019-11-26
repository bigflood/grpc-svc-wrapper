package svc

import (
	"context"
	"fmt"
	"time"

	"github.com/bigflood/grpc_svc_wrapper/pb"
)

type Service struct {
	Name string
}

func (s *Service) Talk(req *pb.TalkRequest, stream pb.Hello_TalkServer) error {
	ctx := stream.Context()
	for i := 0; ctx.Err() == nil; i++ {
		err := stream.Send(&pb.TalkReply{
			Msg: fmt.Sprintf("svcname=%s: %s pong %v", s.Name, req.Msg, i),
		})
		if err != nil {
			return err
		}

		time.Sleep(time.Second)
	}

	return ctx.Err()
}

func (s *Service) Say(ctx context.Context, req *pb.SayRequest) (*pb.SayReply, error) {
	return &pb.SayReply{
		Msg: fmt.Sprintf("svcname=%s: %s pong", s.Name, req.Msg),
	}, nil
}

var _ pb.HelloServer = (*Service)(nil)
