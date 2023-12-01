package task

import (
	"fmt"
	"net/http"
	"runtime"

	"go.uber.org/zap"
)

var (
	panicProtection bool = false
)

func InitWrapPanic(flag bool) {
	panicProtection = flag
}

func CheckWrapPanic() bool {
	return panicProtection
}

func GoSafeTCP(ch chan []byte, req interface{}, fn func(chan []byte, interface{})) {
	defer func() {
		if err := recover(); err != nil {
			stack := make([]byte, 1024*8)
			stack = stack[:runtime.Stack(stack, false)]
			f := "[PANIC] %s\n%s"
			zap.Error(fmt.Errorf(f, err, stack))
		}
	}()
	fn(ch, req)
}

func GoSafeHTTP(rw http.ResponseWriter, r *http.Request, fn func(http.ResponseWriter, *http.Request)) {
	defer func() {
		if err := recover(); err != nil {
			stack := make([]byte, 1024*8)
			stack = stack[:runtime.Stack(stack, false)]
			f := "[PANIC] %s\n%s"
			zap.Error(fmt.Errorf(f, err, stack))
		}
	}()
	fn(rw, r)
}

func GoSafeTimer(fn func()) {
	defer func() {
		if err := recover(); err != nil {
			stack := make([]byte, 1024*8)
			stack = stack[:runtime.Stack(stack, false)]
			f := "[PANIC] %s\n%s"
			zap.Error(fmt.Errorf(f, err, stack))
		}
	}()
	fn()
}

func GoSafe(fn func(...interface{}), args ...interface{}) {
	defer func() {
		if err := recover(); err != nil {
			stack := make([]byte, 1024*8)
			stack = stack[:runtime.Stack(stack, false)]
			f := "[PANIC] %s\n%s"
			zap.Error(fmt.Errorf(f, err, stack))
		}
	}()
	fn(args...)
}
