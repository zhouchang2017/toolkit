package log

import (
	"fmt"
	"github.com/zhouchang2017/toolkit/log/zapformatter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"sync"
)

var (
	zapFormatters    = map[string]zapFormatFactory{}
	_debugHiddenTime bool
)

type zapFormatFactory func(caller bool) zapcore.Encoder

func RegisterZapFormat(name string, factory zapFormatFactory) {
	mu.Lock()
	zapFormatters[name] = factory
	mu.Unlock()
}

func init() {
	RegisterZapFormat("json", func(caller bool) zapcore.Encoder {
		config := zapformatter.NewJsonEncoderConfig()
		if !caller {
			config.CallerKey = ""
			config.EncodeCaller = nil
		}
		if _debugHiddenTime {
			config.TimeKey = ""
			config.EncodeTime = nil
		}
		return zapformatter.NewJSONEncoder(config)
	})
	RegisterZapFormat("plain", func(caller bool) zapcore.Encoder {
		config := zapformatter.NewPlainEncoderConfig()
		if !caller {
			config.CallerKey = ""
			config.EncodeCaller = nil
		}
		if _debugHiddenTime {
			config.TimeKey = ""
			config.EncodeTime = nil
		}
		return zapformatter.NewPLAINEncoder(config)
	})
}

func NewZapLogger(formatter string, caller bool, level string, writers ...io.Writer) FieldLogger {
	return newZapLogger(formatter, caller, level, writers...)
}

func newZapLogger(formatter string, caller bool, level string, writers ...io.Writer) *zapLogger {
	logger := &zapLogger{
		level: zap.NewAtomicLevel(),
	}
	logger.SetLevel(level)

	encoder := newZapEncoder(formatter, caller)

	if formatter == "plain" {
		if enc, ok := encoder.(WithRequestId); ok {
			enc.SetContextFetcher(GetContextRequestID)
		}
	}

	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	ws := make([]zapcore.WriteSyncer, 0, len(writers))
	for _, w := range writers {
		ws = append(ws, zapcore.AddSync(w))
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(ws...),
		logger.level,
	)

	logger.l = zap.New(core, zap.WithCaller(caller), zap.AddCallerSkip(2))
	return logger
}

func newZapEncoder(formatter string, caller bool) zapcore.Encoder {
	if factory, ok := zapFormatters[formatter]; ok {
		return factory(caller)
	}

	config := zap.NewDevelopmentEncoderConfig()
	if !caller {
		config.CallerKey = ""
		config.EncodeCaller = nil
	}
	if _debugHiddenTime {
		config.TimeKey = ""
		config.EncodeTime = nil
	}
	return zapcore.NewConsoleEncoder(config)
}

type zapLogger struct {
	level zap.AtomicLevel
	l     *zap.Logger
	mu    sync.Mutex
}

func (z *zapLogger) clone() *zapLogger {
	copy := *z
	return &copy
}

func (z *zapLogger) log(lvl zapcore.Level, template string, fmtArgs []interface{}) {
	// If logging at this level is completely disabled, skip the overhead of
	// string formatting.
	if !z.l.Core().Enabled(lvl) {
		return
	}

	// Format with Sprint, Sprintf, or neither.
	msg := template
	if msg == "" && len(fmtArgs) > 0 {
		msg = fmt.Sprint(fmtArgs...)
	} else if msg != "" && len(fmtArgs) > 0 {
		msg = fmt.Sprintf(template, fmtArgs...)
	}

	if ce := z.l.Check(lvl, msg); ce != nil {
		ce.Write()
	}
}

func (z *zapLogger) WithField(key string, value interface{}) FieldLogger {
	clone := z.clone()
	clone.l = clone.l.With(zap.Any(key, value))
	return clone
}

func (z *zapLogger) WithFields(fields map[string]interface{}) FieldLogger {
	if len(fields) == 0 {
		return z
	}
	fs := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		fs = append(fs, zap.Any(k, v))
	}
	clone := z.clone()
	clone.l = clone.l.With(fs...)
	return clone
}

func (z *zapLogger) WithError(err error) FieldLogger {
	clone := z.clone()
	clone.l = clone.l.With(zap.Error(err))
	return clone
}

func (z *zapLogger) Debugf(format string, args ...interface{}) {
	z.log(zap.DebugLevel, format, args)
}

func (z *zapLogger) Infof(format string, args ...interface{}) {
	z.log(zap.InfoLevel, format, args)
}

func (z *zapLogger) Warnf(format string, args ...interface{}) {
	z.log(zap.WarnLevel, format, args)
}

func (z *zapLogger) Errorf(format string, args ...interface{}) {
	z.log(zap.ErrorLevel, format, args)
}

func (z *zapLogger) Fatalf(format string, args ...interface{}) {
	z.log(zap.FatalLevel, format, args)
}

func (z *zapLogger) Panicf(format string, args ...interface{}) {
	z.log(zap.PanicLevel, format, args)
}

func (z *zapLogger) Debug(args ...interface{}) {
	z.log(zap.DebugLevel, "", args)
}

func (z *zapLogger) Info(args ...interface{}) {
	z.log(zap.InfoLevel, "", args)
}

func (z *zapLogger) Warn(args ...interface{}) {
	z.log(zap.WarnLevel, "", args)
}

func (z *zapLogger) Error(args ...interface{}) {
	z.log(zap.ErrorLevel, "", args)
}

func (z *zapLogger) Fatal(args ...interface{}) {
	z.log(zap.FatalLevel, "", args)
}

func (z *zapLogger) Panic(args ...interface{}) {
	z.log(zap.PanicLevel, "", args)
}

func (z *zapLogger) SetLevel(l string) error {
	level := new(zapcore.Level)
	err := level.Set(l)
	if err != nil {
		return err
	}
	z.level.SetLevel(*level)
	return nil
}
