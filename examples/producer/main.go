package main

import (
	"net/http"

	"github.com/glue/application"
	"github.com/glue/services/producer"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	application.App
}

func (s *Server) PushMessage(c *gin.Context) {
	msg := &producer.Message{
		Key:   []byte("shanks.zhao"),
		Value: []byte("hello"),
	}
	if err := s.ProducerService.SendMessage(msg); err != nil {
		s.Logger.Error("err", zap.Error(err))
		c.String(http.StatusBadRequest, err.Error())
	}
	c.String(http.StatusOK, string(msg.Value))
}

func main() {
	// grpc example
	server := &Server{application.App{}}
	server.Init()

	// , app.HTTPService.Gin == gin.New()
	r := server.HTTPService.Gin()

	// http gin example
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome Gin Server")
	})

	r.GET("/push", server.PushMessage)

	if err := server.Start(); err != nil {
		server.Logger.Panic("Start Server Failed.", zap.Error(err))
	}
}
