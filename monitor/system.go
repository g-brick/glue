package monitor

var (
	statCpuUsage 			= "cpuUsage"
	statMemUsed 			= "memUsed"
	statMemUsedPercent 		= "memUsedPercent"
	statGoroutines 			= "goroutines"
	statUptimes				= "uptime"
)

type system struct{}

func (s *system) Statistics(tags map[string]string) []Statistic {
	cpuUsage, _ := GetCPUPercent()
	memUsed, _ := GetMemUsage()
	memUsedPercent, _ := GetMemUsagePercent()
	goroutins := GetGoroutines()
	uptime, _ := GetCreateTime()

	return []Statistic{{
		Name: "system",
		Tags: tags,
		Values: map[string]interface{}{
			statCpuUsage: 				cpuUsage,
			statMemUsed: 				memUsed,
			statMemUsedPercent: 		memUsedPercent,
			statGoroutines: 			goroutins,
			statUptimes:				uptime,
		},
		Types: map[string]string{
			statCpuUsage:				StatisticTypeGauge,
			statMemUsed:				StatisticTypeGauge,
			statMemUsedPercent:			StatisticTypeGauge,
			statGoroutines:				StatisticTypeGauge,
			statUptimes:				StatisticTypeGauge,
		},
	}}
}