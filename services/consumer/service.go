package consumer

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"

	"github.com/glue/monitor"
	Producer "github.com/glue/services/producer"

	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
	"go.uber.org/zap"
)

const (
	statBrockers                    = "brokers"
	statConsumers                   = "consumers"
	statConsumerErrors              = "consumerErrors"
	statConsumerNotifications       = "consumerNotifications"
	statConsumerDelayDuration       = "consumerDelayDuration"
	statConsumerLastMessageDuration = "consumerLastMessageDuration"
	statConsumerTimeout             = "consumerTimeout"

	statMessages       = "messages"
	statEmptyMessages  = "emptyMessages"
	statCallbacks      = "callbacks"
	statCallbackOks    = "callbackOks"
	statCallbackErrors = "callbackErrors"
	statLoopConsumers  = "loopConsumers"

	statCallbackDurations = "callbackDurations"

	statCallbackInterval = "callbackInterval"

	DefaultRandomInterval = 50
)

type Statistics struct {
	Brokers                     int64
	Consumers                   int64
	ConsumerErrors              int64
	ConsumerNotifications       int64
	ConsumerDelayDuration       int64
	ConsumerLastMessageDuration int64
	ConsumerTimeout             int64

	Messages       int64
	EmptyMessage   int64
	Callbacks      int64
	CallbackOks    int64
	CallbackErrors int64
	LoopConsumers  int64

	CallbackDurations int64
	CallbackInterval  int64
}

func (s *Service) Statistics(tags map[string]string) []monitor.Statistic {
	if !s.cfg.Enable {
		return nil
	}

	return []monitor.Statistic{{
		Name: "consumer",
		Tags: tags,
		Values: map[string]interface{}{
			statBrockers:                    atomic.LoadInt64(&s.stats.Brokers),
			statConsumers:                   atomic.LoadInt64(&s.stats.Consumers),
			statConsumerErrors:              atomic.LoadInt64(&s.stats.ConsumerErrors),
			statConsumerNotifications:       atomic.LoadInt64(&s.stats.ConsumerNotifications),
			statConsumerDelayDuration:       atomic.LoadInt64(&s.stats.ConsumerDelayDuration),
			statConsumerLastMessageDuration: atomic.LoadInt64(&s.stats.ConsumerLastMessageDuration),
			statConsumerTimeout:             atomic.LoadInt64(&s.stats.ConsumerTimeout),
			statMessages:                    atomic.LoadInt64(&s.stats.Messages),
			statEmptyMessages:               atomic.LoadInt64(&s.stats.EmptyMessage),
			statCallbacks:                   atomic.LoadInt64(&s.stats.Callbacks),
			statCallbackOks:                 atomic.LoadInt64(&s.stats.CallbackOks),
			statCallbackErrors:              atomic.LoadInt64(&s.stats.CallbackErrors),
			statLoopConsumers:               atomic.LoadInt64(&s.stats.LoopConsumers),
			statCallbackDurations:           atomic.LoadInt64(&s.stats.CallbackDurations),
			statCallbackInterval:            atomic.LoadInt64(&s.stats.CallbackInterval),
		},
		Types: map[string]string{
			statBrockers:                    monitor.StatisticTypeGauge,
			statConsumers:                   monitor.StatisticTypeGauge,
			statLoopConsumers:               monitor.StatisticTypeGauge,
			statConsumerDelayDuration:       monitor.StatisticTypeGauge,
			statConsumerLastMessageDuration: monitor.StatisticTypeGauge,
			statCallbackInterval:            monitor.StatisticTypeGauge,
		},
	}}
}

type Message struct {
	Timestamp time.Time
	Brokers   []string
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
}

//type MessageFunc func(message Message) error
type MessagesFunc func(messages []*Message) error

type Service struct {
	ctx    context.Context
	cfg    *Config
	Logger *zap.Logger
	isInit bool

	buffer chan Message

	callbackFunc MessagesFunc

	isReisterCallback chan struct{}

	stats *Statistics

	lastMessageTime int64

	callbackInterval int
}

func NewService(ctx context.Context, cfg Config) *Service {
	return &Service{
		cfg:               &cfg,
		Logger:            zap.NewNop(),
		isInit:            false,
		ctx:               ctx,
		isReisterCallback: make(chan struct{}),
		stats:             &Statistics{},
		lastMessageTime:   time.Now().Unix(),
		callbackInterval:  cfg.ConsumerCallbackInterval,
	}
}

func (s *Service) Init() {
	if !s.cfg.Enable {
		return
	}
	s.isInit = true
	s.Logger.Info("Init Consumer Service.")
}

func (s *Service) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("service", "consumer-service"))
	sarama.Logger = Producer.NewSaramLoggerWithZap(s.Logger)
}

func (s *Service) Start() error {
	if !s.cfg.Enable {
		return nil
	}

	s.Logger.Info("Start Consumer Service.")
	if !s.isInit {
		return fmt.Errorf("consumer service is not init.")
	}

	go s.CheckLastMessageTimestamp()
	go s.CheckConsumerMemoryStatus()

	switch s.cfg.ConsumerService {
	case "kafka":
		return s.StartKafkaConsumers()
	default:
		return fmt.Errorf("Unsupport consumer service:%s.", s.cfg.ConsumerService)
	}
}

func (s *Service) CheckConsumerMemoryStatus() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			memPercent, err := monitor.GetMemUsagePercent()
			if err != nil {
				s.Logger.Error("Get Consumer MemUsagePercent Err.", zap.Error(err))
				continue
			}

			if memPercent > 80 && s.cfg.ConsumerCallbackInterval < 1000 {
				s.callbackInterval = 1000
			} else if memPercent > 50 && s.cfg.ConsumerCallbackInterval < 300 {
				s.callbackInterval = 300
			} else {
				s.callbackInterval = s.cfg.ConsumerCallbackInterval
			}
			atomic.StoreInt64(&s.stats.CallbackInterval, int64(s.callbackInterval))
		}
	}

}

func (s *Service) CheckLastMessageTimestamp() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			atomic.StoreInt64(&s.stats.ConsumerLastMessageDuration, time.Now().Unix()-s.lastMessageTime)
		}
	}
}

func (s *Service) StartKafkaConsumers() error {
	config := cluster.NewConfig()
	config.Version = sarama.V0_10_0_0
	config.Consumer.Offsets.CommitInterval = time.Second
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true
	config.Consumer.Group.Session.Timeout = 20 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 6 * time.Second
	config.Consumer.MaxProcessingTime = 500 * time.Millisecond
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin

	// init consumer
	topics := s.cfg.ConsumerTopics

	for _, brokers := range s.cfg.ConsumerEndpoints {
		atomic.AddInt64(&s.stats.Brokers, 1)
		s.StartKafkaConsumer(strings.Split(brokers, ","), s.cfg.ConsumerGroup, topics, config)
	}
	return nil
}

func (s *Service) StartKafkaConsumer(addrs []string, groupID string, topics []string, config *cluster.Config) {

	for i := 0; i < s.cfg.ConsumerForks; i++ {
		s.Logger.Info("Start Kafka Consumer", zap.Int("Fork", i), zap.Strings("brokers", addrs), zap.String("group", groupID), zap.Strings("topics", topics))

		go func(fork int) {
			consumer, err := cluster.NewConsumer(addrs, groupID, topics, config)
			if err != nil {
				s.Logger.Error("Create New Consumer Failed.", zap.Int("Fork", fork), zap.Error(err), zap.Strings("brokers", addrs), zap.String("group", groupID), zap.Strings("topics", topics))
			}

			atomic.AddInt64(&s.stats.Consumers, 1)

			// trap SIGINT to trigger a shutdown.
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, os.Interrupt)

			// consume errors
			go func() {
				for err := range consumer.Errors() {
					atomic.AddInt64(&s.stats.ConsumerErrors, 1)
					s.Logger.Error("consumer error.", zap.Int("Fork", fork), zap.Error(err))
				}
			}()

			go func() {
				for ntf := range consumer.Notifications() {
					atomic.AddInt64(&s.stats.ConsumerNotifications, 1)
					s.Logger.Debug("consumer rebalanced.", zap.String("ntf", ntf.Type.String()))
				}
			}()
			// consume messages, watch signals
			<-s.isReisterCallback

			messages := make([]*Message, 0)

			ticker := s.NewBatchTimeoutTicker()

			ctx := context.WithValue(context.Background(), "brockers", addrs)
			ctx = context.WithValue(ctx, "forks", fork)

			var currentMsg *sarama.ConsumerMessage

			for {
				select {
				case msg, ok := <-consumer.Messages():
					if ok {
						currentMsg = msg
						s.Logger.Debug("Get Message", zap.String("Topic", msg.Topic), zap.Int32("Partition", msg.Partition), zap.Int64("Offset", msg.Offset), zap.Int("Fork", fork), zap.Strings("brokers", addrs))
						message := &Message{Brokers: addrs, Topic: msg.Topic, Partition: msg.Partition, Offset: msg.Offset, Timestamp: msg.Timestamp, Key: msg.Key, Value: msg.Value}
						messages = append(messages, message)

						if len(messages) >= s.cfg.ConsumerBatchMessageSize {
							// 停止计时器
							ticker.Stop()
							s.SendMessagesToCallBackFunc(ctx, messages)
							consumer.MarkOffset(msg, "")
							messages = make([]*Message, 0)
							// sleep一段时间，防止消息堆积太多后打爆后端服务.
							time.Sleep(time.Duration(s.callbackInterval+rand.Intn(DefaultRandomInterval)) * time.Millisecond)
							// 重新启动计时器
							ticker = s.NewBatchTimeoutTicker()
						}
					} else {
						atomic.AddInt64(&s.stats.EmptyMessage, 1)
					}
				case <-ticker.C:
					atomic.AddInt64(&s.stats.ConsumerTimeout, 1)
					s.SendMessagesToCallBackFunc(ctx, messages)
					if currentMsg != nil {
						consumer.MarkOffset(currentMsg, "")
					}
					messages = make([]*Message, 0)
				case <-s.ctx.Done():
					ticker.Stop()
					consumer.Close()
					s.Logger.Info("Consumer Exit.", zap.Int("Fork", fork), zap.Strings("brokers", addrs))
					return
				}
			}
		}(i)
	}

	return
}

func (s *Service) NewBatchTimeoutTicker() *time.Ticker {
	return time.NewTicker(time.Duration(s.cfg.ConsumerBatchWaitTime) * time.Second)
}

func (s *Service) RegisterCallBackFunc(callback MessagesFunc) {
	s.Logger.Info("Register CallBack Func.")
	s.callbackFunc = callback
	close(s.isReisterCallback)
}

func (s *Service) SendMessagesToCallBackFunc(ctx context.Context, messages []*Message) {
	if len(messages) == 0 {
		atomic.AddInt64(&s.stats.EmptyMessage, 1)
		return
	}

	messageLen := int64(len(messages))

	atomic.AddInt64(&s.stats.Callbacks, 1)
	atomic.AddInt64(&s.stats.Messages, messageLen)
	atomic.AddInt64(&s.stats.LoopConsumers, 1)
	defer atomic.AddInt64(&s.stats.LoopConsumers, -1)

	atomic.StoreInt64(&s.stats.ConsumerDelayDuration, time.Now().Unix()-messages[messageLen-1].Timestamp.Unix())
	atomic.StoreInt64(&s.lastMessageTime, messages[messageLen-1].Timestamp.Unix())

	for {
		if err := s.sendMessagesToCallbackFunc(messages); err != nil {
			atomic.AddInt64(&s.stats.CallbackErrors, 1)
			s.Logger.Error("Send Message to Callback Failed. Sleep 1s, Retry...", zap.Error(err), zap.Any("brokers", ctx.Value("brokers")), zap.Any("forks", ctx.Value("forks")))
			time.Sleep(1 * time.Second)
		} else {
			atomic.AddInt64(&s.stats.CallbackOks, 1)
			s.Logger.Debug("Send Message To CallBackFunc Success.", zap.Any("brokers", ctx.Value("brokers")), zap.Any("forks", ctx.Value("forks")))
			break
		}
	}
}

func (s *Service) sendMessagesToCallbackFunc(messages []*Message) error {
	defer func(start time.Time) {
		atomic.AddInt64(&s.stats.CallbackDurations, time.Since(start).Nanoseconds())
	}(time.Now())

	if s.callbackFunc == nil {
		return fmt.Errorf("unregister callbackFun.")
	}

	if err := s.callbackFunc(messages); err != nil {
		return err
	}
	return nil
}

func (s *Service) IsStopping() bool {
	select {
	case _, ok := <-s.ctx.Done():
		return !ok
	default:
		return false
	}
}

func (s *Service) Stop() error {
	if s.cfg.Enable && !s.IsStopping() {
		s.Logger.Info("Stop Consumer Service.")
	}
	return nil
}
