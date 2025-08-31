package logger

import "go.uber.org/zap"

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Panic(msg string, fields ...zap.Field)
	StringC(key, val string) zap.Field
	StackC(key string) zap.Field
	ErrorC(err error) zap.Field
	AnyC(key string, value any) zap.Field
	Int64C(key string, val int64) zap.Field
	Float64C(key string, val float64) zap.Field
	IntC(key string, val int) zap.Field
}
