package log

import (
	"sync"
)

const RequestID = "rid"

var mu sync.Mutex

var Logger FieldLogger = NewZapLogger("plain", true, "debug")

type WithRequestId interface {
	SetContextFetcher(func() string)
}

type FieldLogger interface {
	WithField(key string, value interface{}) FieldLogger
	WithFields(fields map[string]interface{}) FieldLogger
	WithError(err error) FieldLogger

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})

	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Panic(args ...interface{})

	SetLevel(l string) error
}
