package logger

import (
	"context"
	"time"

	"github.com/influxdata/influxdb/pkg/snowflake"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// OperationNameKey is the logging context key used for identifying name of an operation.
	OperationNameKey = "op_name"

	// OperationEventKey is the logging context key used for identifying a notable
	// event during the course of an operation.
	OperationEventKey = "op_event"

	// OperationElapsedKey is the logging context key used for identifying time elapsed to finish an operation.
	OperationElapsedKey = "op_elapsed"
)
const (
	eventStart = "start"
	eventEnd   = "end"
)

var (
	gen = snowflake.New(0)
)

func nextID() string {
	return gen.NextString()
}

// OperationName returns a field for tracking the name of an operation.
func OperationName(name string) zapcore.Field {
	return zap.String(OperationNameKey, name)
}

// OperationElapsed returns a field for tracking the duration of an operation.
func OperationElapsed(d time.Duration) zapcore.Field {
	return zap.Duration(OperationElapsedKey, d)
}

// OperationEventStart returns a field for tracking the start of an operation.
func OperationEventStart() zapcore.Field {
	return zap.String(OperationEventKey, eventStart)
}

// OperationEventFinish returns a field for tracking the end of an operation.
func OperationEventEnd() zapcore.Field {
	return zap.String(OperationEventKey, eventEnd)
}

// TraceID returns a field "trace_id", value pulled from the (Jaeger) trace ID found in the given context.
// Returns zap.Skip() if the context doesn't have a trace ID.
func TraceID(ctx context.Context) zap.Field {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		if spanContext, ok := span.Context().(jaeger.SpanContext); ok {
			return zap.String("trace_id", spanContext.TraceID().String())
		}
	}
	return zap.Skip()
}

// NewOperation uses the exiting log to create a new logger with context
// containing a trace id and the operation. Prior to returning, a standardized message
// is logged indicating the operation has started. The returned function should be
// called when the operation concludes in order to log a corresponding message which
// includes an elapsed time and that the operation has ended.
func NewOperation(ctx context.Context, log *zap.Logger, msg, name string, fields ...zapcore.Field) (*zap.Logger, func()) {
	f := []zapcore.Field{OperationName(name), TraceID(ctx)}
	if len(fields) > 0 {
		f = append(f, fields...)
	}

	now := time.Now()
	log = log.With(f...)
	log.Info(msg+" (start)", OperationEventStart())

	return log, func() { log.Info(msg+" (end)", OperationEventEnd(), OperationElapsed(time.Since(now))) }
}