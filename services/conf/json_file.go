package conf

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"sync"
	"time"

	"github.com/fatih/structs"
	"go.uber.org/zap"
)


type JsonFileService struct {
	ctx context.Context
	mu     sync.RWMutex
	configs map[string]interface{}
	isInit  bool
	Logger *zap.Logger
	lastUpdate int64
	config *Config
	watchers []WatchFunc
}

func NewJsonFileService(c *Config, ctx context.Context) *JsonFileService {
	return &JsonFileService{
		isInit: false,
		configs: make(map[string]interface{}),
		Logger:  zap.NewNop(),
		config: c,
		ctx: ctx,
		watchers: make([]WatchFunc, 0),
	}
}

// WithLogger sets the logger for the service.
func (s *JsonFileService) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("sub-service", "json-file"))
}

func (cfg *JsonFileService) Init() error {
	cfg.Logger.Info("Init Json File Config Service.")

	cfg.isInit = true

	if err := cfg.FlushCurrentConfig(); err !=nil {
		return err
	}

	return nil
}

func (cfg *JsonFileService) Start() error {
	cfg.Logger.Info("Start Json File Config Service.")

	if !cfg.isInit {
		return fmt.Errorf("json file config service is not init.")
	}

	return nil
}

//TODO: 写完stop函数
func (cfg *JsonFileService) Stop() error {
	cfg.Logger.Info("Stop Json File Config Service.")
	return  nil
}

func (c *JsonFileService) AddWatcher(w WatchFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.watchers = append(c.watchers, w)
}


func (cfg *JsonFileService) FlushCurrentConfig() error {
	cfg.Logger.Debug("File Config Service Flush Current Config.")
	if cfg.isInit == false {
		return fmt.Errorf("file config service is not init.")
	}

	if err := cfg.ReadFromJson(); err != nil {
		cfg.Logger.Error("Read Content From Json Err.", zap.Error(err))
		return err
	}


	now := time.Now()
	cfg.lastUpdate = now.Unix()
	return nil
}

func (cfg *JsonFileService) Check() error {
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

func (cfg *JsonFileService) Get(key string) (interface{}, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	value, _ := cfg.configs[key]
	return value, nil
}

func (cfg *JsonFileService) GetAll() (map[string]interface{}, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return cfg.configs, nil
}

func (cfg *JsonFileService) GetAllJson() ([]byte, error) {
	if err := cfg.Check(); err != nil {
		return nil, err
	}

	conf, err := json.Marshal(cfg.configs)
	if err != nil {
		return nil, fmt.Errorf("Config Service Get All Json Failed. Marshal to json error:%v", err)
	}
	return conf, err
}

func (cfg *JsonFileService) Printf(flag bool) {
	if flag {
		fmt.Println(cfg.lastUpdate, cfg.configs)
	}
}


func (cfg *JsonFileService) ReadFromJson() error {
	contentBytes, err := ioutil.ReadFile(cfg.config.ConfPath)
	if err != nil {
		cfg.Logger.Error("read confPath error.", zap.Error(err))
		return err
	}

	if err := json.Unmarshal(contentBytes, &cfg.configs); err != nil {
		cfg.Logger.Error("unmarshal conf content error.", zap.Error(err), zap.String("conf_path", cfg.config.ConfPath))
		return err
	}

	return nil
}

func (cfg *JsonFileService) Unmarshal(c interface{}) error {
	data, err := cfg.GetAll()
	if err != nil {
		return nil
	}
	if err := cfg.DealConfigDataNew(data, c); err != nil {
		return fmt.Errorf("Unmarshal apollo data err: %v.", err)
	}
	return nil
}

func (cfg *JsonFileService) DealConfigDataNew(data map[string]interface{}, c interface{}) error {
	s := structs.New(c)
	finial := make(map[string]interface{})

	if err := cfg.DealFields(s.Fields(), data, finial); err != nil {
		return err
	}

	content, err := json.Marshal(finial)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(content, c); err != nil {
		return err
	}

	return nil
}

func (cfg *JsonFileService) DealFields(fields []*structs.Field, data map[string]interface{}, finial map[string]interface{}) error {
	for _, fd := range fields {
		if !fd.IsExported() {
			continue
		}

		if fd.Kind() != reflect.Ptr && fd.Kind() != reflect.Struct {
			tag := fd.Tag("json")
			if tag != "" {

				value, ok := data[tag]
				if !ok {
					continue
				}

				finial[tag] = value
			}
			continue
		}

		if len(fd.Fields()) > 0 {
			tmpFinial := make(map[string]interface{})
			if fd.Name() != "Config" {
				finial[fd.Name()] = tmpFinial
			} else {
				tmpFinial = finial
			}

			if err := cfg.DealFields(fd.Fields(), data, tmpFinial); err != nil {
				return err
			}
		}
	}
	return nil
}

