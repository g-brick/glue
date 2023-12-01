package tcp

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"

	"go.uber.org/zap"
)

func init() {
}

type Service struct {
	ctx    context.Context
	Logger *zap.Logger
	cfg    *Config
	isInit bool

	Listener net.Listener
}

func NewService(ctx context.Context, cfg *Config) *Service {
	return &Service{
		isInit: false,
		cfg:    cfg,
		Logger: zap.NewNop(),
		ctx:    ctx,
	}
}

func (s *Service) Init() {
	s.isInit = true
	s.Logger.Info("Init TCP Service.")
}

func (s *Service) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("service", "tcp-service"))
}

func (s *Service) Start() error {
	s.Logger.Info("Start TCP Service.")
	if !s.isInit {
		return fmt.Errorf("TCP Service is not init.")
	}
	addr := s.ServiceAddress()

	var err error
	s.Listener, err = net.Listen("tcp", addr)
	if err != nil && !s.IsStopping() {
		return fmt.Errorf("Start TCP Service failed. %v", err)
	}

	s.Logger.Info("TCP Server is listen.", zap.String("Listen", addr))

	s.Logger.Info("TCP Server is started.")

	go func() {
		for {
			c, err := s.Listener.Accept()
			if err != nil {
				s.Logger.Error("TCP Service Listener err.", zap.Error(err))
				return
			}
			go s.handleConnection(c)
		}
	}()

	return nil
}

func (s *Service) handleConnection(c net.Conn) {
	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		temp := strings.TrimSpace(string(netData))
		if temp == "STOP" {
			break
		}

		result := "hello, tcp service!"
		c.Write([]byte(string(result)))
	}
	c.Close()
}

func (s *Service) ServiceAddress() string {
	return fmt.Sprintf("%s:%d", s.cfg.TCPAddr, s.cfg.TCPPort)
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
	if s.cfg.Enable && !s.IsStopping() {
		s.Logger.Info("Stop TCP Service.")
		if s.Listener != nil {
			s.Listener.Close()
		}
	}
	return nil
}
