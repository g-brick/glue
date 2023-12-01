package conf

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type ConfService interface {
	Init() error
	WithLogger(log *zap.Logger)
	Start() error
	Stop() error
	AddWatcher(w WatchFunc)
	FlushCurrentConfig() error
	Check() error
	Get(key string) (interface{}, error)
	GetAll() (map[string]interface{}, error)
	Printf(flag bool)
	Unmarshal(c interface{}) error
}

const (
	MaxLastUpdateInterval = 300
)

type WatchFunc = func(interface{})

type Service struct {
	ctx context.Context
	isInit  bool
	Logger *zap.Logger

	Service ConfService
}

func NewService(c *Config, ctx context.Context) *Service {
	service := &Service{
		isInit: false,
		Logger:  zap.NewNop(),
		ctx: ctx,
	}


	if c.ConfPath == DefaultConfPath {
		service.Service = NewApolloService(c, ctx)
	} else if c.FileConfigFormat == FileConfigFormatJson {
		service.Service = NewJsonFileService(c, ctx)
	} else if c.FileConfigFormat == FileConfigFormatToml {
		service.Service = NewTomlFileService(c, ctx)
	}

	return service
}

// WithLogger sets the logger for the service.
func (s *Service) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("service", "config-service"))
	s.Service.WithLogger(s.Logger)
}

func (cfg *Service) Init() error {
	if err := cfg.Service.Init(); err != nil {
		return err
	}

	cfg.isInit = true
	return nil
}

func (cfg *Service) Start() error {
	cfg.Logger.Info("Start Config Service.")

	if !cfg.isInit {
		return fmt.Errorf("config service is not init.")
	}

	if err := cfg.Service.Start(); err != nil {
		return err
	}

	cfg.Printf(true)

	return nil
}

//TODO: 写完stop函数
func (cfg *Service) Stop() error {
	cfg.Logger.Info("Stop Config Service.")

	if err := cfg.Service.Stop(); err != nil {
		return err
	}
	return  nil
}

func (c *Service) AddWatcher(w WatchFunc) {
	c.Service.AddWatcher(w)
}

func (cfg *Service) Get(key string) (interface{}, error) {
	return cfg.Service.Get(key)
}

func (cfg *Service) GetAll() (map[string]interface{}, error) {
	return cfg.Service.GetAll()
}

func (cfg *Service) Printf(flag bool) {
	cfg.Service.Printf(flag)
	return
}


func (cfg *Service) Unmarshal(c interface{}) error {
	return cfg.Service.Unmarshal(c)
}
