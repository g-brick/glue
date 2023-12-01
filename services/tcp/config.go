package tcp

const (
	DefaultTCPAddr = "0.0.0.0"
	DefaultTCPPort = 4323
	DefaultTCPEnable = false
)

type Config struct {
	Enable 			bool        `json:"tcp.enable"  toml:"enable"`
	TCPAddr 		string   	`json:"tcp.addr"   toml:"addr"`
	TCPPort 		int			`json:"tcp.port"   toml:"port"`
}

func NewConfig() Config{
	c := Config{
		Enable:   DefaultTCPEnable,
		TCPAddr: DefaultTCPAddr,
		TCPPort: DefaultTCPPort,
	}

	return c
}