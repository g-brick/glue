package redis

const (
	DefaultSentinelAddrs 		= ""
	DefaultSentinelMasterName 	= ""
	DefaultRedisCluster 		= ""
)

type Config struct {
	SentinelAddrs		string 		`json:"redis.sentinel.addrs" toml:"sentinelAddr"`
	SentinelMasterName	string 		`json:"redis.sentinel.master.name" toml:"sentinelMasterName"`
	RedisCluster		string 		`json:"redis.cluster" toml:"cluster"`
}

func NewConfig() Config{
	c := Config{
		SentinelAddrs: 		DefaultSentinelAddrs,
		SentinelMasterName: DefaultSentinelMasterName,
		RedisCluster: 		DefaultRedisCluster,
	}
	return c
}