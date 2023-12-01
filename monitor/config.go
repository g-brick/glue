package monitor

const (
	DefaultMetricPrefix = "Glue_service"
)

type Config struct {
	MetricPrefix string `json:"monitor.metric_prefix" toml:"metric_prefix"`
}

func NewConfig() Config {
	return Config{
		MetricPrefix: DefaultMetricPrefix,
	}
}
