package tcp

const (
	DefaultTCPAddr   = "0.0.0.0"
	DefaultTCPPort   = 4323
	DefaultTCPEnable = true
)

type Config struct {
	Enable  bool   `json:"example.tcp.enable"`
	TCPAddr string `json:"example.tcp.addr"`
	TCPPort int    `json:"example.tcp.port"`
}

func NewConfig() *Config {
	c := &Config{
		Enable:  DefaultTCPEnable,
		TCPAddr: DefaultTCPAddr,
		TCPPort: DefaultTCPPort,
	}

	return c
}
