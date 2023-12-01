package main

import (
	//"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/glue/application"
)

const (
	DefaultMySQLEndpoint = "localhost:3306"
	DefaultMySQLUsername = "root"
	DefaultMySQLPassowrd = ""
)

type MySQLConfig struct {
	MySQLEndpoint string `json:"mysql.endpoint"`
	MySQLUsername string `json:"mysql.username"`
	MySQLPassword string `json:"mysql.password"`
}

func NewMySQLConfig() *MySQLConfig {
	c := &MySQLConfig{
		MySQLEndpoint: DefaultMySQLEndpoint,
		MySQLUsername: DefaultMySQLUsername,
		MySQLPassword: DefaultMySQLPassowrd,
	}
	return c
}

type Server struct {
	application.App
	cfg *Config
}

type Config struct {
	*application.Config
	MySQL *MySQLConfig
}

func main() {
	server := &Server{application.App{}, &Config{application.NewConfig(), NewMySQLConfig()}}

	server.InitWithConfig(server.cfg)

	if err := server.Start(); err != nil {
		server.Logger.Panic("Start Server Failed.", zap.Error(err))
	}
}
