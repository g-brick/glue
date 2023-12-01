package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	ginzap "github.com/akath19/gin-zap"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

type Service struct {
	ctx context.Context
	Logger *zap.Logger
	cfg *Config
	isInit bool

	*http.Server

	*gin.Engine

	Monitor interface {
		GodEyeStatistics(tags map[string]string) (map[string]interface{}, error)
	}
}


func NewService(ctx context.Context, cfg Config) *Service {
	return &Service{
		isInit: false,
		cfg: &cfg,
		Logger: zap.NewNop(),
		ctx: ctx,
		Engine: gin.New(),
	}
}

func (s *Service) Gin() *gin.Engine {
	return s.Engine
}

func (s *Service) Init() {
	s.Server = &http.Server{}
	s.InitGin()
	s.isInit = true
	s.Logger.Info("Init HTTP Service.")
}

func (s *Service) InitGin() {
	s.Engine.Use(gin.Recovery())
	p := ginprometheus.NewPrometheus("gin")

	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		url := c.Request.URL.Path
		return url
	}

	p.Use(s.Engine)

	pprof.Register(s.Engine)

	s.InitHandlers()

	s.Handler = s.Engine
}

func (s *Service) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("service", "http-service"))
	s.Engine.Use(ginzap.Logger(3*time.Second, s.Logger))
}

func (s *Service) Start() error {
	s.Logger.Info("Start HTTP Service.")
	if !s.isInit {
		return fmt.Errorf("HTTP Service is not init.")
	}
	addr := s.ServiceAddress()

	s.Server.Addr = addr

	go func() {
		if err := s.ListenAndServe(); err != nil && !s.IsStopping() {
			s.Logger.Panic("HTTP Server has error.", zap.Error(err))
		}
		s.Logger.Info("HTTP Server is exited.")
	}()

	s.Logger.Info("HTTP Server is listen.", zap.String("Listen", s.Addr ))

	s.Logger.Info("HTTP Server is started.")

	return nil
}

func (s *Service) ServiceAddress() string {
	return fmt.Sprintf("%s:%d", s.cfg.HTTPAddr, s.cfg.HTTPPort)
}

func (s *Service) IsStopping() bool {
	select {
	case _, ok := <-s.ctx.Done():
		return !ok
	default:
		return false
	}
}

//TODO: finish stop func
func (s *Service) Stop() error {
	if s.cfg.Enable && ! s.IsStopping() {
		s.Logger.Info("Stop HTTP Service.")
	}
	return nil
}
