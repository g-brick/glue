package main

import (
	"context"
	"log"
	//"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/glue/application"
	pb "github.com/glue/examples/grpc/helloworld"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	// grpc example
	app := &application.App{}
	app.Init()

	pb.RegisterGreeterServer(app.GRPCService.Server, &server{})

	if err := app.Start(); err != nil {
		app.Logger.Panic("Start Server Failed.", zap.Error(err))
	}
}
