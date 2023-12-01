package monitor

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"go.uber.org/zap"
)

type Service struct {
	ctx        context.Context
	Logger     *zap.Logger
	mu         sync.RWMutex
	reporter   Reporter
	globalTags map[string]string
	cfg        *Config

	otherReports map[string]Reporter

	metrics map[string]interface{}
}

func NewService(ctx context.Context, r Reporter, cfg Config) *Service {
	return &Service{
		globalTags:   make(map[string]string),
		reporter:     r,
		ctx:          ctx,
		Logger:       zap.NewNop(),
		otherReports: make(map[string]Reporter),
		cfg:          &cfg,
		metrics:      make(map[string]interface{}),
	}
}

func (s *Service) Init() {
	s.Logger.Info("Init Monitor Service.")
	s.RegisterReport("system", &system{})
	s.RegisterReport("runtime", &Runtime{})

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.PromStatistics(nil)
			}
		}
	}()
}

func (s *Service) RegisterReport(name string, reporter Reporter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.otherReports[name] = reporter
	s.Logger.Info("Register Report.", zap.String("name", name))
}

func (s *Service) WithLogger(log *zap.Logger) {
	s.Logger = log.With(zap.String("service", "monitor-service"))
}

// SetGlobalTag can be used to set tags that will appear on all points
// written by the Monitor.
func (m *Service) SetGlobalTag(key string, value interface{}) {
	m.mu.Lock()
	m.globalTags[key] = fmt.Sprintf("%v", value)
	m.mu.Unlock()
}

// Statistic represents the information returned by a single monitor client.
type MonitorStatistic struct {
	Statistic
}

func (m *Service) GodEyeStatistics(tags map[string]string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	tags = StatisticTags(tags).Merge(m.globalTags)
	statistics, err := m.Statistics(tags)
	if err != nil {
		return nil, err
	}
	for _, statistic := range statistics {
		tagArray := make([]string, 0)
		for tagKey, tagValue := range statistic.Tags {
			tmp := fmt.Sprintf("%s=%s", tagKey, tagValue)
			tagArray = append(tagArray, tmp)
		}

		sort.Strings(tagArray)

		tagStr := strings.Join(tagArray, ",")

		if statistic.Types == nil {
			statistic.Types = make(map[string]string)
		}

		for key, value := range statistic.Values {
			tp, ok := statistic.Types[key]
			if !ok {
				tp = StatisticTypeCounter
			}
			metricWithTag := fmt.Sprintf("%s.%s.%s{%s,type=%s}", m.cfg.MetricPrefix, statistic.Name, key, tagStr, tp)
			result[metricWithTag] = value
		}
	}

	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		m.Logger.Error("Prometheus Gather Err.", zap.Error(err))
		return result, nil
	}

	for _, metric := range metrics {
		var metricName, service, metricType string
		var metricValue float64

		nameArray := strings.Split(metric.GetName(), "_")

		if len(nameArray) > 1 {
			service = nameArray[0]
			if service == m.cfg.MetricPrefix {
				continue
			}
			metricName = strings.Join(nameArray[1:], "_")
		} else {
			service = "unknown"
			metricName = *metric.Name
		}

		typ := metric.GetType()
		switch typ {
		case io_prometheus_client.MetricType_COUNTER:
			metricType = StatisticTypeCounter
		case io_prometheus_client.MetricType_HISTOGRAM:
			metricType = StatisticTypeCounter
		case io_prometheus_client.MetricType_SUMMARY:
			metricType = StatisticTypeGauge
		default:
			metricType = StatisticTypeGauge
		}

		for _, tag := range metric.Metric {
			labels := tag.GetLabel()
			labelArray := make([]string, 0)
			for _, label := range labels {
				tmp := fmt.Sprintf("%s=%s", label.GetName(), label.GetValue())
				labelArray = append(labelArray, tmp)
			}

			for tagKey, tagValue := range tags {
				tmp := fmt.Sprintf("%s=%s", tagKey, tagValue)
				labelArray = append(labelArray, tmp)
			}

			sort.Strings(labelArray)
			tagStr := strings.Join(labelArray, ",")

			switch typ {
			case io_prometheus_client.MetricType_COUNTER:
				metricValue = tag.GetCounter().GetValue()
			case io_prometheus_client.MetricType_GAUGE:
				metricValue = tag.GetGauge().GetValue()
			default:
				continue
			}

			metricWithTag := fmt.Sprintf("%s.%s.%s{%s,type=%s}", m.cfg.MetricPrefix, service, metricName, tagStr, metricType)
			result[metricWithTag] = metricValue
		}
	}

	return result, nil
}

func (m *Service) PromStatistics(tags map[string]string) {
	tags = StatisticTags(tags).Merge(m.globalTags)
	statistics, err := m.Statistics(tags)
	if err != nil {
		return
	}
	for _, statistic := range statistics {
		tagKeyArray := make([]string, 0)
		tagValueArray := make([]string, 0)

		for tagKey, _ := range statistic.Tags {
			tagKeyArray = append(tagKeyArray, tagKey)
		}

		sort.Strings(tagKeyArray)

		for _, tagKey := range tagKeyArray {
			tagValue, ok := statistic.Tags[tagKey]
			if !ok {
				continue
			}
			tagValueArray = append(tagValueArray, tagValue)
		}

		if statistic.Types == nil {
			statistic.Types = make(map[string]string)
		}

		for key, value := range statistic.Values {
			help, _ := statistic.Helps[key]

			metricName := fmt.Sprintf("%s_%s_%s", m.cfg.MetricPrefix, statistic.Name, key)

			var Vector interface{}

			Vector, ok := m.metrics[metricName];
			if !ok {

				newVector := prometheus.NewGaugeVec(prometheus.GaugeOpts{
					Name: metricName,
					Help: help,
				}, tagKeyArray)

				m.metrics[metricName] = newVector
				if err := prometheus.Register(newVector); err != nil {
					continue
				}

			}

			Vector, _ = m.metrics[metricName];

			// FIXME: 有没有更好的解决办法。。。。
			var Value float64
			switch v := value.(type) {
			case int:
				Value = float64(v)
			case int8:
				Value = float64(v)
			case int32:
				Value = float64(v)
			case int64:
				Value = float64(v)
			case uint:
				Value = float64(v)
			case uint8:
				Value = float64(v)
			case uint32:
				Value = float64(v)
			case uint64:
				Value = float64(v)
			case float32:
				Value = float64(v)
			case float64:
				Value = float64(v)
			default:
				m.Logger.Error("unsupport value type.", zap.String("Key", metricName))
				continue
			}

			switch vector := Vector.(type) {
			case *prometheus.GaugeVec:
				vector.WithLabelValues(tagValueArray...).Set(Value)
			default:
				continue
			}
		}
	}

	return
}

// Statistics returns the combined statistics for all expvar data. The given
// tags are added to each of the returned statistics.
func (m *Service) Statistics(tags map[string]string) ([]*MonitorStatistic, error) {
	var statistics []*MonitorStatistic
	statistics = m.gatherStatistics(statistics, tags)

	return statistics, nil
}

func (m *Service) gatherStatistics(statistics []*MonitorStatistic, tags map[string]string) []*MonitorStatistic {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.reporter != nil {
		for _, s := range m.reporter.Statistics(tags) {
			statistics = append(statistics, &MonitorStatistic{Statistic: s})
		}
	}

	for _, report := range m.otherReports {
		for _, s := range report.Statistics(tags) {
			statistics = append(statistics, &MonitorStatistic{Statistic: s})
		}
	}
	return statistics
}
