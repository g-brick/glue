package task

import (
	"reflect"
	"runtime"
	"strings"
)

type taskState int

const (
	stateNew taskState = iota
	stateRun
	stateFinished
)

var (
	globalTaskId       int32
)

// 根据handler获取任务方法名字
func GetTaskFuncName(taskHandler interface{}) string {
	funcInfo := runtime.FuncForPC(reflect.ValueOf(taskHandler).Pointer()).Name()
	return strings.Split(funcInfo, ".")[1]
}

// 根据Goroutine Id 获取任务实例
func GetTaskByGid(gid uint64) (task interface{}) {
	task = GetTCPTaskByGid(gid)
	if task != nil {
		return
	}
	return
}
