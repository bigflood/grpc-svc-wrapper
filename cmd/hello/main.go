package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/bigflood/grpc_svc_wrapper/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func main() {
	svcname := fmt.Sprintf("svc%d", rand.Intn(1000)+1000)

	ctx := context.Background()
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("svcname", svcname))

	conn, err := grpc.Dial(":9992", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	client := pb.NewHelloClient(conn)

	{
		reply, err := client.Say(ctx, &pb.SayRequest{Msg: "hello"})
		if err != nil {
			log.Fatal(err)
		}

		log.Println("reply:", reply.Msg)
	}

	{
		stream, err := client.Talk(ctx, &pb.TalkRequest{Msg: "talk"})
		if err != nil {
			log.Fatal(err)
		}

		for stream.Context().Err() == nil {
			reply, err := stream.Recv()
			if err != nil {
				log.Fatal(err)
			}

			log.Println("talk:", reply.Msg)
		}

		log.Println(stream.Context().Err())
	}
}
