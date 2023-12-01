package main

import (
	"github.com/glue/application"
	"github.com/glue/examples/custom-service/tcp"

	"go.uber.org/zap"
)

type Server struct {
	application.App
	cfg *Config
}

type Config struct {
	*application.Config
	TCP *tcp.Config
}

func main() {
	// grpc example
	server := &Server{application.App{}, &Config{application.NewConfig(), tcp.NewConfig()}}

	server.InitWithConfig(server.cfg)

	tcpService := tcp.NewService(server.Context(), server.cfg.TCP)

	server.RegisterNewService("tcp", tcpService)

	if err := server.Start(); err != nil {
		server.Logger.Panic("Start Server Failed.", zap.Error(err))
	}
}
