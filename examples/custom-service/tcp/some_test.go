package tcp

import (
	"fmt"
	"testing"

	"github.com/glue/application"

	"github.com/gin-gonic/gin"
)

func TestMain(t *testing.M) {
	app := application.Default()
	r := app.HTTPService.Gin()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	fmt.Println("pong")
}
