package conf

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/imdario/mergo"
	"github.com/shima-park/agollo"

	"go.uber.org/zap"
)

const (
	DefaultConfigCheckInterval = 300 * time.Second
)

type ApolloService struct {
	ctx context.Context
	mu     sync.RWMutex
	configs map[string]interface{}
	isInit  bool
	Logger *zap.Logger
	lastUpdate int64
	config *Config
	watchers []WatchFunc
}

func NewApolloService(c *Config, ctx context.Context) *ApolloService {
	return &ApolloService{
		isInit: false,
		configs: make(map[string]interface{}),
		Logger:  zap.NewNop(),
		config: c,
		ctx: ctx,
		watchers: make([]WatchFunc, 0),
	}
}

// WithLogger sets the logger for the service.
func (s *ApolloService) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("sub-service", "apollo"))
}

func (cfg *ApolloService) Init() error {
	cfg.Logger.Info("Init Apollo Config Service.")
	err := agollo.InitWithDefaultConfigFile(
		agollo.WithLogger(agollo.NewLogger(agollo.LoggerWriter(os.Stdout))), // 打印日志信息
		agollo.AutoFetchOnCacheMiss(),                                       // 在配置未找到时，去apollo的带缓存的获取配置接口，获取配置
		agollo.FailTolerantOnBackupExists(),                                 // 在连接apollo失败时，如果在配置的目录下存在.agollo备份配置，会读取备份在服务器无法连接的情况下
	)

	if err != nil {
		return err
	}

	cfg.isInit = true

	if err := cfg.FlushCurrentConfig(); err !=nil {
		cfg.Logger.Error("Flush Current Config Err.", zap.Error(err))
		return err
	}

	return nil
}

func (cfg *ApolloService) Start() error {
	cfg.Logger.Info("Start Apollo Config Service.")

	if !cfg.isInit {
		return fmt.Errorf("apollo config service is not init.")
	}

	errorCh := agollo.Start()

	watchCh := agollo.Watch()

	go func() {
		ticker := time.NewTicker(DefaultConfigCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				time.Sleep(time.Duration(rand.Intn(60))*time.Second)
				if err := cfg.Check(); err != nil {
					cfg.Logger.Error("schedule apollo check err.", zap.Error(err))
				}
			case <-cfg.ctx.Done():
				return
			case err := <-errorCh:
				cfg.Logger.Error("apollo config service", zap.Error(err.Err))
				cfg.Start()
				return
			case <-watchCh:
				if err := cfg.FlushCurrentConfig(); err != nil {
					cfg.Logger.Error("apollo config service flush current config", zap.Error(err))
				}
				cfg.Printf(true)
			}
		}
	}()

	return nil
}

//TODO: 写完stop函数
func (cfg *ApolloService) Stop() error {
	cfg.Logger.Info("Stop Config Service.")
	return  nil
}

func (c *ApolloService) AddWatcher(w WatchFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.watchers = append(c.watchers, w)
}

func (cfg *ApolloService) FlushCurrentConfig() error {
	cfg.Logger.Info("Config Service Flush Current Config.")
	if cfg.isInit == false {
		return fmt.Errorf("config service is not init.")

	}

	publicConfig := agollo.GetNameSpace(cfg.config.PublicNamespace)
	applicationConfig := agollo.GetNameSpace(cfg.config.PrimayNamespace)

	if err := mergo.Merge(&applicationConfig, &publicConfig); err != nil {
		return fmt.Errorf("failed to merge public and application configs. %v", err)
	}

	cfg.configs = applicationConfig

	if len(cfg.configs) == 0 {
		return fmt.Errorf("failed to get configs from apollo. config is Empty.")
	}

	content, err := json.Marshal(cfg.configs)
	if err != nil {
		return fmt.Errorf("failed to marshal configs. %v", err)

	}
	cfg.Logger.Info("Current Config", zap.ByteString("config",   content))

	now := time.Now()
	cfg.lastUpdate = now.Unix()
	return nil
}

func (cfg *ApolloService) Check() error {
	if cfg.isInit == false {
		return fmt.Errorf("config service is not init.")

	}

	now := time.Now()
	if now.Unix() - cfg.lastUpdate > MaxLastUpdateInterval {
		if err := cfg.FlushCurrentConfig(); err != nil {
			return err
		}
	}
	return nil
}

func (cfg *ApolloService) Get(key string) (interface{}, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	value, _ := cfg.configs[key]
	return value, nil
}

func (cfg *ApolloService) GetAll() (map[string]interface{}, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return cfg.configs, nil
}

func (cfg *ApolloService) GetAllJson() ([]byte, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}

	conf, err := json.Marshal(cfg.configs)
	if err != nil {
		return nil, fmt.Errorf("Config Service Get All Json Failed. Marshal to json error:%v", err)
	}
	return conf, err
}

func (cfg *ApolloService) Printf(flag bool) {
	if flag {
		fmt.Println(cfg.lastUpdate, cfg.configs)
	}
}


func (cfg *ApolloService) Unmarshal(c interface{}) error {
	data, err := cfg.GetAll()
	if err != nil {
		return nil
	}
	if err := DealConfigData(data, c); err != nil {
		return fmt.Errorf("Unmarshal apollo data err: %v.", err)
	}
	return nil
}
