package tcp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

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

	// About connection
	maxConnectionId uint64
	connections     map[uint64]*TcpConnection
	connectionMutex sync.Mutex

	// About server start and stop
	stopFlag int32
	stopWait sync.WaitGroup
}

func NewService(ctx context.Context, cfg Config) *Service {
	return &Service{
		isInit:      false,
		cfg:         &cfg,
		Logger:      zap.NewNop(),
		connections: make(map[uint64]*TcpConnection),
		ctx:         ctx,
	}
}

func (s *Service) Init() {

	if !s.cfg.Enable {
		return
	}

	s.isInit = true
	s.Logger.Info("Init TCP Service.")
}

func (s *Service) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("service", "tcp-service"))
}

func (s *Service) Start() error {

	if !s.cfg.Enable {
		return nil
	}

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

			connection, err := s.newConnection(c)
			s.Logger.Debug("new server connection", zap.String("RemoteAddr", c.RemoteAddr().String()), zap.String("LocalAddr", c.LocalAddr().String()))

			// 钩子函数，用于业务server嵌入连接建立时的逻辑
			OnConnect(connection)
			// 每个连接启动一个goroutine处理
			go s.handleConnection(connection)
		}
	}()

	return nil
}

func (s *Service) newConnection(conn net.Conn) (c *TcpConnection, err error) {
	c = newTcpConnection(conn)
	if err = c.SetKeepAlive(defaultKeepAlivePeriod); err != nil {
		return
	}
	s.addConnection(c)
	return
}

func (s *Service) addConnection(c *TcpConnection) {
	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()
	s.connections[c.id] = c
	s.stopWait.Add(1)
}

func (s *Service) handleConnection(c *TcpConnection) {
	for {
		req, err := c.Receive()
		if err != nil {
			s.Logger.Debug("connection is closed", zap.String("RemoteAddr", c.conn.RemoteAddr().String()), zap.String("LocalAddr", c.conn.LocalAddr().String()))
			s.delConnection(c)
			return
		}
		// 每个请求启动一个goroutine处理
		// 钩子函数，用于业务server嵌入接收数据时的逻辑
		go OnDataIn(c, req)
	}
}

func (s *Service) delConnection(c *TcpConnection) {
	s.connectionMutex.Lock()
	defer s.connectionMutex.Unlock()
	delete(s.connections, c.id)
	s.stopWait.Done()
}

func (s *Service) lenConnection() int {
	return len(s.connections)
}

func (s *Service) closeConnections() {
	for _, connection := range s.connections {
		connection.Close()
	}
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

		if atomic.CompareAndSwapInt32(&s.stopFlag, 0, 1) {
			if s.Listener != nil {
				s.Listener.Close()
			}
			s.closeConnections()
			s.stopWait.Wait()
			return nil
		}
		return nil
	}
	return nil
}
