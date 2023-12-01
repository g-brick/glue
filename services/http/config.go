package http

import (
	"os"
	"strconv"
)

const (
	DefaultHTTPAddr   = "0.0.0.0"
	DefaultHTTPPort   = 4322
	DefaultHTTPEnable = true

	DefaultHTTPPortEnvName = "PORT1"
)

type Config struct {
	Enable   bool   `json:"http.enable" toml:"enable"`
	HTTPAddr string `json:"http.addr" toml:"addr"`
	HTTPPort int    `json:"http.port" toml:"port"`
}

func NewConfig() Config {
	c := Config{
		Enable:   DefaultHTTPEnable,
		HTTPAddr: DefaultHTTPAddr,
		HTTPPort: DefaultHTTPPort,
	}

	if len(os.Getenv(DefaultHTTPPortEnvName)) != 0 {
		c.HTTPPort = GetIntValue(os.Getenv(DefaultHTTPPortEnvName), DefaultHTTPPort)
	}

	return c
}

func GetIntValue(value string, df int) int {
	v, err := strconv.Atoi(value)
	if err != nil {
		return df
	}

	return v
}
