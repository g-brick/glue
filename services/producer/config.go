package producer

const (
	DefaultProducerEnable 	= false
	DefaultProducerService  = "kafka"
	DefaultProducerTopic 	= "umonitor3"
	DefaultProducerPoolSize = 5
	DefaultProducerTimeout  = 3
	DefaultProducerMaxMessageBytes = 1024 * 1024 * 5
)

var (
	DefaultProducerEndpoints = make([]string, 0)
)

type Config struct {
	Enable 					bool   			`json:"producer.enable" toml:"enable"`
	ProducerService 		string 			`json:"producer.service" toml:"service"`
	ProducerEndpoints		[]string		`json:"producer.endpoints" toml:"endpoints"`
	ProducerTopic			string			`json:"producer.topic" toml:"topic"`
	ProducerPoolSize		int 			`json:"producer.pool_size" toml:"pool_size"`
	ProducerTimeout			int 			`json:"producer.timeout" toml:"timeout"`
	ProducerMaxMessageBytes int 			`json:"producer.max_message_bytes" toml:"max_message_bytes"`
}

func NewConfig() Config{
	c := Config{
		Enable:    					DefaultProducerEnable,
		ProducerService:   			DefaultProducerService,
		ProducerEndpoints: 			DefaultProducerEndpoints,
		ProducerTopic:    			DefaultProducerTopic,
		ProducerPoolSize:   		DefaultProducerPoolSize,
		ProducerTimeout: 			DefaultProducerTimeout,
		ProducerMaxMessageBytes: 	DefaultProducerMaxMessageBytes,
	}
	return c
}