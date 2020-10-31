package zapformatter

import (
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"strings"
	"time"
)

var (
	_pool = buffer.NewPool()
	// Get retrieves a buffer from the pool, creating one if necessary.
	getBuf = _pool.Get

	TimeFormat = "2006-01-02 15:04:05.000000"
)

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}

var encodeTime = func(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
	encoder.AppendString(t.Format(TimeFormat))
}

var encoderCaller = func(caller zapcore.EntryCaller, encoder zapcore.PrimitiveArrayEncoder) {
	buf := getBuf()
	idx := strings.LastIndexByte(caller.File, '/')
	if idx > -1 {
		buf.AppendString(caller.File[idx+1:])
	} else {
		buf.AppendString(caller.File)
	}
	buf.AppendByte(':')
	buf.AppendInt(int64(caller.Line))
	encoder.AppendString(buf.String())
	buf.Reset()
	buf.Free()
}

func NewJsonEncoderConfig() zapcore.EncoderConfig {
	ec := zap.NewProductionEncoderConfig()
	ec.EncodeTime = encodeTime
	ec.TimeKey = "time"
	ec.EncodeCaller = encoderCaller
	return ec
}

func NewPlainEncoderConfig() zapcore.EncoderConfig {
	ec := zap.NewProductionEncoderConfig()
	ec.ConsoleSeparator = " "
	ec.EncodeLevel = zapcore.CapitalLevelEncoder
	ec.EncodeTime = encodeTime
	ec.EncodeLevel = func(level zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(level.CapitalString()[:4])
	}
	ec.EncodeCaller = encoderCaller
	return ec
}
