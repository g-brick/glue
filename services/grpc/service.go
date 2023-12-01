package grpc

import (
	"context"
	"fmt"
	"net"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/grpc-ecosystem/go-grpc-middleware"
)

type Service struct {
	ctx context.Context
	Logger *zap.Logger
	cfg *Config
	isInit bool

	*grpc.Server

	unaryServerInterceptors []grpc.UnaryServerInterceptor
	streamServerInterceptors []grpc.StreamServerInterceptor

	serverOptions []grpc.ServerOption
}


func NewService(ctx context.Context, cfg Config, options []interface{}) *Service {
	service := &Service{
		isInit: false,
		cfg: &cfg,
		Logger: zap.NewNop(),
		ctx: ctx,
		unaryServerInterceptors: make([]grpc.UnaryServerInterceptor, 0),
		streamServerInterceptors: make([]grpc.StreamServerInterceptor, 0),
		serverOptions: make([]grpc.ServerOption, 0),
	}

	if options != nil {
		for _, option := range options {
			switch optionV := option.(type) {
			case grpc.UnaryServerInterceptor:
				service.unaryServerInterceptors = append(service.unaryServerInterceptors, optionV)
			case grpc.StreamServerInterceptor:
				service.streamServerInterceptors = append(service.streamServerInterceptors, optionV)
			case grpc.ServerOption:
				service.serverOptions = append(service.serverOptions, optionV)
			default:
			}
		}
	}

	service.initCommonInterceptors()

	return service
}

func (s *Service) initCommonInterceptors() {
	s.unaryServerInterceptors = append(s.unaryServerInterceptors,
		grpc_ctxtags.UnaryServerInterceptor(),
		grpc_prometheus.UnaryServerInterceptor,
		grpc_recovery.UnaryServerInterceptor(),
	)

	s.streamServerInterceptors = append(s.streamServerInterceptors,
		grpc_ctxtags.StreamServerInterceptor(),
		grpc_prometheus.StreamServerInterceptor,
		grpc_recovery.StreamServerInterceptor(),
	)
}

func (s *Service) Init() {
	grpcOptions := append(
		s.serverOptions,
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			s.streamServerInterceptors...,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			s.unaryServerInterceptors...,
		)))
	s.Server = grpc.NewServer(grpcOptions...)

	if !s.cfg.Enable {
		return
	}

	s.isInit = true
	s.Logger.Info("Init GRPC Service.")
}

func (s *Service) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("service", "grpc-service"))
}


func (s *Service) Start() error {
	if !s.cfg.Enable {
		return nil
	}
	s.Logger.Info("Start GRPC Service.")
	if !s.isInit {
		return fmt.Errorf("GRPC Service is not init.")
	}

	s.RegisterMiddleWare()

	addr := s.ServiceAddress()
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("Start GRPC Service at %s failed, %v", addr, err)
	}

	s.Logger.Info("GRPC Server is listen.", zap.String("Listen", addr))

	go func() {
		if err := s.Server.Serve(lis); err != nil && !s.IsStopping() {
			s.Logger.Panic("GRPC Server has error.", zap.Error(err))
		}
		s.Logger.Info("GRPC Server is exited.")
	}()

	s.Logger.Info("GRPC Server is started.")

	return nil
}

func (s *Service) RegisterMiddleWare() {
	// register grpc_prometheus
	grpc_prometheus.Register(s.Server)
}

func (s *Service) ServiceAddress() string {
	return fmt.Sprintf("%s:%d", s.cfg.GRPCAddr, s.cfg.GRPCPort)
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
	if s.cfg.Enable && !s.IsStopping(){
		s.Logger.Info("Stop GRPC Service.")
	}
	return nil
}