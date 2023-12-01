package monitor

import (
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

const (
	MaxMemEnvName = "MARATHON_APP_RESOURCE_MEM"
)

var (
	gProcess process.Process
)

func init() {
	pid := int32(os.Getpid())
	gProcess = process.Process{Pid: pid}
}

//获取cpu率
func GetCPUPercent() (float64, error) {
	cpuPercent, err := gProcess.Percent(0)
	if err != nil {
		return 0, err
	}
	return cpuPercent, nil
}

//获取内存使用量
func GetMemUsage() (int32, error) {
	memInfoStat, err := gProcess.MemoryInfo()
	if err != nil {
		return 0, err
	}
	rss := int32(memInfoStat.RSS / 1024 / 1024)
	return rss, nil

}

func GetMemUsagePercent() (float64, error) {
	memUsage, err := GetMemUsage()
	if err != nil {
		return 0, err
	}

	maxMemoryStr := os.Getenv(MaxMemEnvName)

	GlobalMaxMem, err := strconv.ParseFloat(maxMemoryStr, 10)
	if err != nil {
		v, _ := mem.VirtualMemory()
		return v.UsedPercent, nil
	}

	GlobalMemUsedPercent := (float64(memUsage) / float64(GlobalMaxMem)) * float64(100)
	return GlobalMemUsedPercent, nil
}

//获取goroutine数量
func GetGoroutines() int {
	return runtime.NumGoroutine()
}

//获取操作系统信息
func GetRunTime() (*host.InfoStat, error) {
	info, err := host.Info()
	if err != nil {
		return nil, err
	}
	return info, nil
}

//获取进程启动时间
func GetCreateTime() (int64, error) {
	createTime, err := gProcess.CreateTime()
	return time.Now().Unix() - createTime/1000, err
}
