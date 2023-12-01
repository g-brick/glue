package grpc

import (
	"os"
	"strconv"
)

const (
	DefaultGRPCAddr = "0.0.0.0"
	DefaultGRPCPort = 4321
	DefaultGRPCEnable = false

	DefaultGRPCPortEnvName = "PORT0"
)

type Config struct {
	Enable 			bool        `json:"grpc.enable" toml:"enable"`
	GRPCAddr 		string   	`json:"grpc.addr" toml:"addr"`
	GRPCPort 		int			`json:"grpc.port" toml:"port"`
}

func NewConfig() Config{
	c := Config{
		Enable: DefaultGRPCEnable,
		GRPCAddr: DefaultGRPCAddr,
		GRPCPort: DefaultGRPCPort,
	}

	if len(os.Getenv(DefaultGRPCPortEnvName)) != 0 {
		c.GRPCPort = GetIntValue(os.Getenv(DefaultGRPCPortEnvName), DefaultGRPCPort )
	}

	return c
}

func GetIntValue(value string, df int) int {
	v, err := strconv.Atoi(value);
	if err != nil {
		return df
	}

	return v
}