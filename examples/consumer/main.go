package main

import (
	"fmt"
	"time"

	"github.com/glue/services/consumer"

	"go.uber.org/zap"

	"github.com/glue/application"
)

type Server struct {
	application.App
}

func (s *Server) WriteToStdout(messages []*consumer.Message) error {
	for _, message := range messages {
		fmt.Println("message=", message.Timestamp, message.Topic, message.Partition, message.Offset, string(message.Key), string(message.Value))
		time.Sleep(1 * time.Second)
	}

	return nil
}

func main() {
	server := &Server{application.App{}}
	server.Init()

	server.ConsumerService.RegisterCallBackFunc(server.WriteToStdout)

	if err := server.Start(); err != nil {
		server.Logger.Panic("Start Server Failed.", zap.Error(err))
	}
}
