package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Log struct {
	driver *logD
}

func New(parameter bool) *Log {
	logD := newD(parameter)
	return &Log{
		driver: logD,
	}
}

func (l *Log) Info(msg string, fields ...zap.Field) {
	l.driver.Info(msg, fields...)
}

func (l *Log) Fatal(msg string, fields ...zap.Field) {
	l.driver.Fatal(msg, fields...)
}

func (l *Log) Debug(msg string, fields ...zap.Field) {
	l.driver.Debug(msg, fields...)
}

func (l *Log) Error(msg string, fields ...zap.Field) {
	l.driver.Error(msg, fields...)
}

func (l *Log) Panic(msg string, fields ...zap.Field) {
	l.driver.Panic(msg, fields...)
}

func (l *Log) StringC(key, val string) zap.Field {
	return zap.String(key, val)
}

func (l *Log) StackC(key string) zap.Field {
	return zap.Stack(key)
}

func (l *Log) ErrorC(err error) zap.Field {
	return zap.Error(err)
}

func (l *Log) AnyC(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

func (l *Log) Int64C(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}

func (l *Log) Float64C(key string, val float64) zap.Field {
	return zap.Float64(key, val)
}

func (l *Log) IntC(key string, val int) zap.Field {
	return zap.Int(key, val)
}

type logD struct {
	*zap.Logger
}

func newD(parameter bool) *logD {
	log := logD{
		zap.New(zapcore.NewTee(), zap.AddCaller()),
	}
	log.setHook(parameter)
	return &log
}

func (l *logD) setHook(parameter bool) {
	if parameter {
		core := l.createFileCore()
		l.Logger = zap.New(core)
	} else {
		core := l.createStdoutCore()
		l.Logger = zap.New(core)
	}
}

func (l *logD) createFileCore() zapcore.Core {
	err := os.MkdirAll("logs", 0777)
	if err != nil {
		panic(err)
	}

	jsonEncoder := zapcore.NewJSONEncoder(
		zapcore.EncoderConfig{
			MessageKey:     "message",
			LevelKey:       "level",
			TimeKey:        "time",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    "function",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeName:     zapcore.FullNameEncoder,
		},
	)

	fileWriter := zapcore.AddSync(
		&lumberjack.Logger{
			Filename:   "logs/all.log",
			MaxSize:    50,
			MaxBackups: 1,
			Compress:   true,
			LocalTime:  true,
		},
	)

	stdoutWriter := zapcore.AddSync(os.Stdout)

	return zapcore.NewTee(
		zapcore.NewCore(jsonEncoder, fileWriter, zapcore.DebugLevel),
		zapcore.NewCore(jsonEncoder, stdoutWriter, zapcore.DebugLevel),
	)
}

func (l *logD) createStdoutCore() zapcore.Core {
	jsonEncoder := zapcore.NewJSONEncoder(
		zapcore.EncoderConfig{
			MessageKey:     "message",
			LevelKey:       "level",
			TimeKey:        "time",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    "function",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeName:     zapcore.FullNameEncoder,
		},
	)

	stdoutWriter := zapcore.AddSync(os.Stdout)

	return zapcore.NewCore(jsonEncoder, stdoutWriter, zapcore.DebugLevel)
}
