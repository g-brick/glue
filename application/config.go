package application

import (
	"encoding/json"
	"fmt"

	"github.com/glue/clients/mysql"

	"github.com/glue/monitor"
	"github.com/glue/services/conf"
	"github.com/glue/services/consumer"
	"github.com/glue/services/grpc"
	"github.com/glue/services/http"
	"github.com/glue/services/producer"
	"github.com/glue/services/tcp"
)

const (
	DefaultGroup = "default"
)

type GlobalConfig struct {
	Module string `json:"global.module" toml:"module"`
	Group  string `json:"global.group" toml:"group"`
}

func NewGlobalConfig(module string) GlobalConfig {
	return GlobalConfig{
		Module: module,
		Group:  DefaultGroup,
	}
}

type Configurator interface {
	Configuration() *Config
}

type Config struct {
	cfg *conf.Service

	Global   GlobalConfig    `toml:"global"`
	GRPC     grpc.Config     `toml:"grpc"`
	HTTP     http.Config     `toml:"http"`
	TCP      tcp.Config      `toml:"tcp"`
	Consumer consumer.Config `toml:"consumer"`
	Monitor  monitor.Config  `toml:"monitor"`
	MySQL    mysql.Config    `toml:"mysql"`
	Producer producer.Config `toml:"producer"`
}

func NewConfig() *Config {
	c := &Config{}
	c.Init()
	return c
}

func (c *Config) Configuration() *Config { return c }

func (c *Config) Init() {
	c.GRPC = grpc.NewConfig()
	c.HTTP = http.NewConfig()
	c.TCP = tcp.NewConfig()
	c.Consumer = consumer.NewConfig()
	c.MySQL = mysql.NewConfig()
	c.Monitor = monitor.NewConfig()
	c.Producer = producer.NewConfig()
}

func (c *Config) FromJson(data []byte) error {

	if err := json.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("Config Parse From Json Failed. %v", err)
	}

	return nil
}

func (c *Config) Unmarshal(cfg interface{}) error {
	return c.cfg.Unmarshal(cfg)
}
