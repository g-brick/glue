package main

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/glue/application"
)

func main() {
	app := application.Default()
	r := app.HTTPService.Gin()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	if err := app.Start(); err != nil {
		app.Logger.Panic("Start Server Failed.", zap.Error(err))
	}
}
