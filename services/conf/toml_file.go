package conf

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/pelletier/go-toml"
	"go.uber.org/zap"

	toml2 "github.com/BurntSushi/toml"
)


type TomlFileService struct {
	ctx context.Context
	mu     sync.RWMutex
	tree   *toml.Tree

	content []byte
	isInit  bool
	Logger *zap.Logger
	lastUpdate int64
	config *Config
	watchers []WatchFunc
}

func NewTomlFileService(c *Config, ctx context.Context) *TomlFileService {
	return &TomlFileService{
		isInit: false,
		tree: &toml.Tree{},
		Logger:  zap.NewNop(),
		config: c,
		ctx: ctx,
		watchers: make([]WatchFunc, 0),
	}
}

// WithLogger sets the logger for the service.
func (s *TomlFileService) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("sub-service", "toml-file"))
}

func (cfg *TomlFileService) Init() error {
	cfg.Logger.Info("Init Toml File Config Service.")

	cfg.isInit = true

	if err := cfg.FlushCurrentConfig(); err !=nil {
		return err
	}

	return nil
}

func (cfg *TomlFileService) Start() error {
	cfg.Logger.Info("Start Toml File Config Service.")

	if !cfg.isInit {
		return fmt.Errorf("json file config service is not init.")
	}

	return nil
}

//TODO: 写完stop函数
func (cfg *TomlFileService) Stop() error {
	cfg.Logger.Info("Stop Toml File Config Service.")
	return  nil
}

func (c *TomlFileService) AddWatcher(w WatchFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.watchers = append(c.watchers, w)
}


func (cfg *TomlFileService) FlushCurrentConfig() error {
	cfg.Logger.Debug("File Config Service Flush Current Config.")
	if cfg.isInit == false {
		return fmt.Errorf("file config service is not init.")
	}

	if err := cfg.ReadFromToml(); err != nil {
		cfg.Logger.Error("Read Content From Toml Err.", zap.Error(err))
		return err
	}


	now := time.Now()
	cfg.lastUpdate = now.Unix()
	return nil
}

func (cfg *TomlFileService) Check() error {
	if cfg.isInit == false {
		return fmt.Errorf("file config service is not init.")

	}

	now := time.Now()
	if now.Unix() - cfg.lastUpdate > MaxLastUpdateInterval {
		if err := cfg.FlushCurrentConfig(); err != nil {
			return err
		}
	}
	return nil
}

func (cfg *TomlFileService) Get(key string) (interface{}, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}

	return cfg.tree.Get(key), nil
}

func (cfg *TomlFileService) GetAll() (map[string]interface{}, error) {
	return cfg.tree.ToMap(), nil
}

func (cfg *TomlFileService) Printf(flag bool) {
	if flag {
		fmt.Println(cfg.lastUpdate, cfg.tree.String())
	}
}


func (cfg *TomlFileService) ReadFromToml() error {

	contentBytes, err := ioutil.ReadFile(cfg.config.ConfPath)
	if err != nil {
		cfg.Logger.Error("read confPath error.", zap.Error(err))
		return err
	}

	cfg.content = contentBytes

	tree, err := toml.LoadBytes(contentBytes)
	if err != nil {
		cfg.Logger.Error("load toml format error.", zap.Error(err))
		return err
	}

	cfg.tree = tree

	return nil
}

func (cfg *TomlFileService) Unmarshal(c interface{}) error {
	return toml2.Unmarshal(cfg.content, c)
}

