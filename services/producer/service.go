package producer

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/glue/monitor"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

const (
	NormalMode = iota
	SafeMode
)

const (
	DefaultNormalMemPercent = 85
	DefaultSafeMemPercent   = 90
)

const (
	statProducerMode           = "producerMode"
	statProducers              = "producers"
	statProducerRejects        = "producerRejects"
	statMesaggeBytes           = "messageBytes"
	statMessageMaxBytes        = "messageMaxBytes"
	statMessages               = "messages"
	statMessageOks             = "messageOks"
	statMessageErrors          = "messageErrors"
	statMessageDurations       = "messageDurations"
	statMessageDurationOver10s = "messageDurationOver10s"
	statMessageDurationOver5s  = "messageDurationOver5s"
	statMessageDurationOver3s  = "messageDurationOver3s"
	statMessageDurationOver2s  = "messageDurationOver2s"
	statMessageDurationOver1s  = "messageDurationOver1s"
)

type Statistics struct {
	Producers              int64
	ProducerMode           int64
	ProducerRejects        int64
	Messages               int64
	MessageBytes           int64
	MessageOks             int64
	MessageErrors          int64
	MessageDurations       int64
	MessageMaxBytes        int64
	MessageDurationOver10s int64
	MessageDurationOver5s  int64
	MessageDurationOver3s  int64
	MessageDurationOver2s  int64
	MessageDurationOver1s  int64
}

func (s *Service) Statistics(tags map[string]string) []monitor.Statistic {
	if !s.cfg.Enable {
		return nil
	}

	return []monitor.Statistic{{
		Name: "producer",
		Tags: tags,
		Values: map[string]interface{}{
			statProducers:              atomic.LoadInt64(&s.stats.Producers),
			statProducerMode:           atomic.LoadInt64(&s.stats.ProducerMode),
			statProducerRejects:        atomic.LoadInt64(&s.stats.ProducerRejects),
			statMessages:               atomic.LoadInt64(&s.stats.Messages),
			statMessageErrors:          atomic.LoadInt64(&s.stats.MessageErrors),
			statMessageDurations:       atomic.LoadInt64(&s.stats.MessageDurations),
			statMessageOks:             atomic.LoadInt64(&s.stats.MessageOks),
			statMesaggeBytes:           atomic.LoadInt64(&s.stats.MessageBytes),
			statMessageMaxBytes:        atomic.LoadInt64(&s.stats.MessageMaxBytes),
			statMessageDurationOver10s: atomic.LoadInt64(&s.stats.MessageDurationOver10s),
			statMessageDurationOver5s:  atomic.LoadInt64(&s.stats.MessageDurationOver5s),
			statMessageDurationOver3s:  atomic.LoadInt64(&s.stats.MessageDurationOver3s),
			statMessageDurationOver2s:  atomic.LoadInt64(&s.stats.MessageDurationOver2s),
			statMessageDurationOver1s:  atomic.LoadInt64(&s.stats.MessageDurationOver1s),
		},
		Types: map[string]string{
			statProducers:       monitor.StatisticTypeGauge,
			statProducerMode:    monitor.StatisticTypeGauge,
			statMessageMaxBytes: monitor.StatisticTypeGauge,
		},
	}}
}

type Message struct {
	Timestamp time.Time
	Topic     string
	Partition int32
	Key       []byte
	Value     []byte
}

type MessageFunc func(message Message) error

type SaramaLogger struct {
	Logger *zap.Logger
}

func NewSaramLoggerWithZap(log *zap.Logger) sarama.StdLogger {
	logger := &SaramaLogger{
		Logger: log.With(zap.String("sub-service", "sarama")),
	}
	return logger
}

func (sl *SaramaLogger) Print(v ...interface{}) {
	value := fmt.Sprintf("%v", v)
	sl.Logger.Debug("Sarama Print", zap.String("msg", value))
}

func (sl *SaramaLogger) Printf(format string, v ...interface{}) {
	value := fmt.Sprintf(format, v)
	sl.Logger.Debug("Sarama Printf", zap.String("msg", value))
}

func (sl *SaramaLogger) Println(v ...interface{}) {
	value := fmt.Sprintf("%v\n", v)
	sl.Logger.Debug("Sarama Println", zap.String("msg", value))
}

type Service struct {
	ctx    context.Context
	cfg    *Config
	Logger *zap.Logger
	isInit bool

	mu sync.Mutex

	stats *Statistics

	Monitor interface {
		Statistics(tags map[string]string) ([]*monitor.MonitorStatistic, error)
	}

	producers map[string]chan sarama.SyncProducer

	mode int
}

func hash(s string) uint32 {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		return 0
	}

	return h.Sum32()
}

func NewService(ctx context.Context, cfg Config) *Service {
	if cfg.ProducerPoolSize < 2 {
		cfg.ProducerPoolSize = 2
	}
	return &Service{
		cfg:       &cfg,
		Logger:    zap.NewNop(),
		isInit:    false,
		ctx:       ctx,
		stats:     &Statistics{},
		producers: make(map[string]chan sarama.SyncProducer),
	}
}

func (s *Service) Init() {
	if !s.cfg.Enable {
		return
	}

	for _, broker := range s.cfg.ProducerEndpoints {
		s.producers[broker] = make(chan sarama.SyncProducer, s.cfg.ProducerPoolSize)
	}

	s.isInit = true
	s.Logger.Info("Init Producer Service.")
}

func (s *Service) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("service", "producer-service"))
	sarama.Logger = NewSaramLoggerWithZap(s.Logger)
}

func (s *Service) Start() error {
	if !s.cfg.Enable {
		return nil
	}

	s.Logger.Info("Start producer Service.")
	if !s.isInit {
		return fmt.Errorf("producer service is not init.")
	}
	go s.CheckProducerMemoryStatus()

	switch s.cfg.ProducerService {
	case "kafka":
		return s.StartKafkaProducer()
	default:
		return fmt.Errorf("Unsupport producer service:%s.", s.cfg.ProducerService)
	}
}

func (s *Service) CheckProducerMemoryStatus() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			memPrecent, err := monitor.GetMemUsagePercent()
			if err != nil {
				s.Logger.Error("Get Consumer MemUsagePercent Err.", zap.Error(err))
				continue
			}

			if memPrecent > DefaultSafeMemPercent && s.mode == NormalMode {
				s.mode = SafeMode
				atomic.StoreInt64(&s.stats.ProducerMode, int64(s.mode))
				s.Logger.Error("Current MemPrecent > DefaultSafeMemPercent, Turn to SafeMode.", zap.Float64("MemPercent", memPrecent), zap.Int("Mode", s.mode))
			} else if memPrecent <= DefaultNormalMemPercent && s.mode == SafeMode {
				s.mode = NormalMode
				atomic.StoreInt64(&s.stats.ProducerMode, int64(s.mode))
				s.Logger.Info("Current MemPrecent <= DefaultNormalMemPercent, Turn to NormalMode.", zap.Float64("MemPercent", memPrecent), zap.Int("Mode", s.mode))
			}
		}
	}
}

func (s *Service) StartKafkaProducer() error {
	config := sarama.NewConfig()
	config.Version = sarama.V0_10_0_0
	config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message
	config.Producer.Retry.Max = 3
	// Retry up to 10 times to produce the message
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.Timeout = time.Duration(s.cfg.ProducerTimeout) * time.Second
	config.Producer.MaxMessageBytes = s.cfg.ProducerMaxMessageBytes

	config.Metadata.Retry.Max = 1
	config.Net.DialTimeout = 3 * time.Second

	// init producer
	for i := 0; i < s.cfg.ProducerPoolSize; i++ {
		for _, brokers := range s.cfg.ProducerEndpoints {
			s.Logger.Info("Start Kafka Producer", zap.Int("Fork", i), zap.String("brokers", brokers))

			producer, err := sarama.NewSyncProducer(strings.Split(brokers, ","), config)
			if err != nil {
				s.Logger.Error("New sync producer faile.", zap.Error(err))
			}

			s.producers[brokers] <- producer

			atomic.AddInt64(&s.stats.Producers, 1)
		}
	}

	if atomic.LoadInt64(&s.stats.Producers) == 0 {
		return fmt.Errorf("Start Kafka Producer failed.", zap.Int("producers", len(s.producers)))
	}

	return nil
}

func (s *Service) PopProducer(hashkey string) (string, sarama.SyncProducer) {
	index := 0
	if len(hashkey) == 0 {
		index = rand.Intn(len(s.cfg.ProducerEndpoints))
	} else {
		hashInt := hash(hashkey)
		index = int(hashInt % uint32(len(s.cfg.ProducerEndpoints)))
	}

	if index >= len(s.cfg.ProducerEndpoints) {
		index = rand.Intn(len(s.cfg.ProducerEndpoints))
	}

	broker := s.cfg.ProducerEndpoints[index]

	producer := <-s.producers[broker]

	return broker, producer
}

func (s *Service) SendMessage(message *Message) error {
	defer func(start time.Time) {
		atomic.AddInt64(&s.stats.MessageDurations, time.Since(start).Nanoseconds())
	}(time.Now())

	if s.mode == SafeMode {
		atomic.AddInt64(&s.stats.ProducerRejects, 1)
		return fmt.Errorf("Current Service is SafeMode. Reject New Request....")
	}

	brokder, producer := s.PopProducer(string(message.Key))
	defer func() {
		s.producers[brokder] <- producer
	}()

	messageLen := int64(len(message.Value))
	atomic.AddInt64(&s.stats.Messages, 1)
	atomic.AddInt64(&s.stats.MessageBytes, messageLen)

	if atomic.LoadInt64(&s.stats.MessageMaxBytes) < messageLen {
		atomic.StoreInt64(&s.stats.MessageMaxBytes, messageLen)
	}

	topic := s.cfg.ProducerTopic

	if len(message.Topic) != 0 {
		topic = message.Topic
	}

	producerMessage := &sarama.ProducerMessage{
		Key:       sarama.ByteEncoder(message.Key),
		Value:     sarama.ByteEncoder(message.Value),
		Topic:     topic,
		Timestamp: time.Now(),
	}
	if producerMessage.Key.Length() == 0 {
		producerMessage.Key = nil
	}

	partition, offset, err := producer.SendMessage(producerMessage)
	if err != nil {
		atomic.AddInt64(&s.stats.MessageErrors, 1)
		s.Logger.Error("producer SendMessage Err.", zap.Error(err), zap.Int64("ValueLen", messageLen))
		return err
	} else {
		atomic.AddInt64(&s.stats.MessageOks, 1)
		s.Logger.Info("producer SendMessage Ok.", zap.Int32("partition", partition), zap.Int64("offset", offset))
	}

	return nil
}

func (s *Service) SendMessages(messages []*Message) error {

	if s.mode == SafeMode {
		atomic.AddInt64(&s.stats.ProducerRejects, 1)
		return fmt.Errorf("Current Service is SafeMode. Reject New Request....")
	}

	broker, producer := s.PopProducer("")
	defer func() {
		s.producers[broker] <- producer
	}()

	messagesCount := int64(len(messages))

	atomic.AddInt64(&s.stats.Messages, messagesCount)

	now := time.Now()

	producerMessages := make([]*sarama.ProducerMessage, 0)
	for _, message := range messages {
		topic := s.cfg.ProducerTopic

		if len(message.Topic) != 0 {
			topic = message.Topic
		}
		producerMessage := &sarama.ProducerMessage{
			Key:       sarama.ByteEncoder(message.Key),
			Value:     sarama.ByteEncoder(message.Value),
			Topic:     topic,
			Timestamp: now,
		}
		if producerMessage.Key.Length() == 0 {
			producerMessage.Key = nil
		}
		producerMessages = append(producerMessages, producerMessage)

		messageLen := int64(len(message.Value))
		atomic.AddInt64(&s.stats.MessageBytes, messageLen)

		if atomic.LoadInt64(&s.stats.MessageMaxBytes) < messageLen {
			atomic.StoreInt64(&s.stats.MessageMaxBytes, messageLen)
		}
	}

	defer func(start time.Time) {
		t := time.Since(start).Nanoseconds()
		if t > time.Second.Nanoseconds()*10 {
			atomic.AddInt64(&s.stats.MessageDurationOver10s, 1)
		} else if t > time.Second.Nanoseconds()*5 {
			atomic.AddInt64(&s.stats.MessageDurationOver5s, 1)
		} else if t > time.Second.Nanoseconds()*3 {
			atomic.AddInt64(&s.stats.MessageDurationOver3s, 1)
		} else if t > time.Second.Nanoseconds()*2 {
			atomic.AddInt64(&s.stats.MessageDurationOver2s, 1)
		} else if t > time.Second.Nanoseconds()*1 {
			atomic.AddInt64(&s.stats.MessageDurationOver1s, 1)
		}
		atomic.AddInt64(&s.stats.MessageDurations, t)
	}(time.Now())

	err := producer.SendMessages(producerMessages)
	if err != nil {
		atomic.AddInt64(&s.stats.MessageErrors, messagesCount)
		s.Logger.Error("producer SendMessages Err.", zap.Error(err))
		return err
	} else {
		atomic.AddInt64(&s.stats.MessageOks, messagesCount)
		s.Logger.Debug("producer SendMessages Ok.", zap.Int("message count", len(messages)))
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
		s.Logger.Info("Stop Producer Service.")
		go func() {
			for _, producers := range s.producers {
				for producer := range producers {
					producer.Close()
				}
				time.Sleep(100 * time.Millisecond)
				close(producers)
			}
		}()
	}
	return nil
}
