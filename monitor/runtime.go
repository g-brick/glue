package monitor

import (
	"runtime"
)

type Runtime struct{}

func (r *Runtime) Statistics(tags map[string]string) []Statistic {
	var rt runtime.MemStats
	runtime.ReadMemStats(&rt)

	return []Statistic{{
		Name: "runtime",
		Tags: tags,
		Values: map[string]interface{}{
			"Alloc":        int64(rt.Alloc),
			"TotalAlloc":   int64(rt.TotalAlloc),
			"Sys":          int64(rt.Sys),
			"Lookups":      int64(rt.Lookups),
			"Mallocs":      int64(rt.Mallocs),
			"Frees":        int64(rt.Frees),
			"HeapAlloc":    int64(rt.HeapAlloc),
			"HeapSys":      int64(rt.HeapSys),
			"HeapIdle":     int64(rt.HeapIdle),
			"HeapInUse":    int64(rt.HeapInuse),
			"HeapReleased": int64(rt.HeapReleased),
			"HeapObjects":  int64(rt.HeapObjects),
			"PauseTotalNs": int64(rt.PauseTotalNs),
			"NumGC":        int64(rt.NumGC),
			"NumGoroutine": int64(runtime.NumGoroutine()),
		},
		Types: map[string]string{
			"Alloc":        StatisticTypeGauge,
			"Sys":          StatisticTypeGauge,
			"Lookups":      StatisticTypeGauge,
			"Mallocs":      StatisticTypeGauge,
			"Frees":        StatisticTypeGauge,
			"HeapAlloc":    StatisticTypeGauge,
			"HeapSys":      StatisticTypeGauge,
			"HeapIdle":     StatisticTypeGauge,
			"HeapInUse":    StatisticTypeGauge,
			"HeapReleased": StatisticTypeGauge,
			"HeapObjects":  StatisticTypeGauge,
			"NumGC":        StatisticTypeGauge,
			"NumGoroutine": StatisticTypeGauge,
		},
	}}
}
