package consumer

const (
	DefaultConsumerEnable           = false
	DefaultConsumerService          = "kafka"
	DefaultConsumerGroup            = "example"
	DefaultConsumerForks            = 10
	DefaultConsumerBatchMessageSize = 2000
	DefaultConsumerBatchWaitTime    = 10
	DefaultConsumerCallbackInterval = 100
)

var (
	DefaultConsumerEndpoints = make([]string, 0)
	DefaultConsumerTopics    = []string{"Glue_service"}
)

type Config struct {
	Enable                   bool     `json:"consumer.enable" toml:"enable"`
	ConsumerService          string   `json:"consumer.service" toml:"service"`
	ConsumerEndpoints        []string `json:"consumer.endpoints" toml:"endpoints"`
	ConsumerTopics           []string `json:"consumer.topics" toml:"topics"`
	ConsumerGroup            string   `json:"consumer.group" toml:"group"`
	ConsumerForks            int      `json:"consumer.forks" toml:"forks"`
	ConsumerBatchMessageSize int      `json:"consumer.batch_message_size" toml:"batch_message_size"`
	ConsumerBatchWaitTime    int      `json:"consumer.batch_wait_time" toml:"batch_wait_time"`
	ConsumerCallbackInterval int      `json:"consumer.callback_interval" toml:"callback_interval"`
}

func NewConfig() Config {
	c := Config{
		Enable:                   DefaultConsumerEnable,
		ConsumerService:          DefaultConsumerService,
		ConsumerEndpoints:        DefaultConsumerEndpoints,
		ConsumerTopics:           DefaultConsumerTopics,
		ConsumerGroup:            DefaultConsumerGroup,
		ConsumerForks:            DefaultConsumerForks,
		ConsumerBatchMessageSize: DefaultConsumerBatchMessageSize,
		ConsumerBatchWaitTime:    DefaultConsumerBatchWaitTime,
		ConsumerCallbackInterval: DefaultConsumerCallbackInterval,
	}
	return c
}
