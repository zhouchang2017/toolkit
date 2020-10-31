// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package zapformatter

import (
	"encoding/base64"
	"encoding/json"
	"go.uber.org/zap/zapcore"
	"math"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap/buffer"
)

var _plainPool = sync.Pool{New: func() interface{} {
	return &plainEncoder{}
}}

func getPLAINEncoder() *plainEncoder {
	return _plainPool.Get().(*plainEncoder)
}

func putPLAINEncoder(enc *plainEncoder) {
	if enc.reflectBuf != nil {
		enc.reflectBuf.Free()
	}
	enc.EncoderConfig = nil
	enc.buf = nil
	enc.spaced = false
	enc.openNamespaces = 0
	enc.reflectBuf = nil
	enc.reflectEnc = nil
	_plainPool.Put(enc)
}

type plainEncoder struct {
	*zapcore.EncoderConfig
	buf            *buffer.Buffer
	spaced         bool // include spaces after colons and commas
	openNamespaces int

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc *json.Encoder

	// with request id
	contextFetcher func() string
}

// NewPLAINEncoder creates a fast, low-allocation PLAIN encoder. The encoder
// appropriately escapes all field keys and values.
//
// Note that the encoder doesn't deduplicate keys, so it's possible to produce
// a message like
//   {"foo":"bar","foo":"baz"}
// This is permitted by the JSON specification, but not encouraged. Many
// libraries will ignore duplicate key-value pairs (typically keeping the last
// pair) when unmarshaling, but users should attempt to avoid adding duplicate
// keys.
func NewPLAINEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	if len(cfg.ConsoleSeparator) == 0 {
		// Use a default delimiter of '\t' for backwards compatibility
		cfg.ConsoleSeparator = "\t"
	}
	return newPLAINEncoder(cfg, true)
}

func newPLAINEncoder(cfg zapcore.EncoderConfig, spaced bool) *plainEncoder {
	return &plainEncoder{
		EncoderConfig: &cfg,
		buf:           getBuf(),
		spaced:        spaced,
	}
}

func (enc *plainEncoder) SetContextFetcher(fn func() string) {
	enc.contextFetcher = fn
}

func (enc *plainEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	return enc.AppendArray(arr)
}

func (enc *plainEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	return enc.AppendObject(obj)
}

func (enc *plainEncoder) AddBinary(key string, val []byte) {
	enc.AddString(key, base64.StdEncoding.EncodeToString(val))
}

func (enc *plainEncoder) AddByteString(key string, val []byte) {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	enc.AppendByteString(val)
}

func (enc *plainEncoder) AddBool(key string, val bool) {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	enc.AppendBool(val)
}

func (enc *plainEncoder) AddComplex128(key string, val complex128) {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	enc.AppendComplex128(val)
}

func (enc *plainEncoder) AddDuration(key string, val time.Duration) {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	enc.AppendDuration(val)
}

func (enc *plainEncoder) AddFloat64(key string, val float64) {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	enc.AppendFloat64(val)
}

func (enc *plainEncoder) AddInt64(key string, val int64) {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	enc.AppendInt64(val)
}

func (enc *plainEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = getBuf()
		enc.reflectEnc = json.NewEncoder(enc.reflectBuf)

		// For consistency with our custom JSON encoder.
		enc.reflectEnc.SetEscapeHTML(false)
	} else {
		enc.reflectBuf.Reset()
	}
}

// Only invoke the standard JSON encoder if there is actually something to
// encode; otherwise write JSON null literal directly.
func (enc *plainEncoder) encodeReflected(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nullLiteralBytes, nil
	}
	enc.resetReflectBuf()
	if err := enc.reflectEnc.Encode(obj); err != nil {
		return nil, err
	}
	enc.reflectBuf.TrimNewline()
	return enc.reflectBuf.Bytes(), nil
}

func (enc *plainEncoder) AddReflected(key string, obj interface{}) error {
	valueBytes, err := enc.encodeReflected(obj)
	if err != nil {
		return err
	}
	enc.addKey(key)
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *plainEncoder) OpenNamespace(key string) {
	enc.addKey(key)
	enc.buf.AppendByte('{')
	enc.openNamespaces++
}

func (enc *plainEncoder) AddString(key, val string) {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	enc.AppendString(val)
}

func (enc *plainEncoder) AddTime(key string, val time.Time) {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	enc.AppendTime(val)
}

func (enc *plainEncoder) AddUint64(key string, val uint64) {
	defer enc.addSeparator(']')
	enc.addSeparator('[')
	enc.addKey(key)
	enc.AppendUint64(val)
}

func (enc *plainEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	enc.addElementSeparator()
	enc.buf.AppendByte('[')
	err := arr.MarshalLogArray(enc)
	enc.buf.AppendByte(']')
	return err
}

func (enc *plainEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	enc.addElementSeparator()
	enc.buf.AppendByte('{')
	err := obj.MarshalLogObject(enc)
	enc.buf.AppendByte('}')
	return err
}

func (enc *plainEncoder) AppendBool(val bool) {
	enc.addElementSeparator()
	enc.buf.AppendBool(val)
}

func (enc *plainEncoder) AppendByteString(val []byte) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.safeAddByteString(val)
	enc.buf.AppendByte('"')
}

func (enc *plainEncoder) AppendComplex128(val complex128) {
	enc.addElementSeparator()
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(val)), float64(imag(val))
	enc.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, 64)
	enc.buf.AppendByte('+')
	enc.buf.AppendFloat(i, 64)
	enc.buf.AppendByte('i')
	enc.buf.AppendByte('"')
}

func (enc *plainEncoder) AppendDuration(val time.Duration) {
	cur := enc.buf.Len()
	if e := enc.EncodeDuration; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is a no-op. Fall back to nanoseconds to keep
		// JSON valid.
		enc.AppendInt64(int64(val))
	}
}

func (enc *plainEncoder) AppendInt64(val int64) {
	enc.addElementSeparator()
	enc.buf.AppendInt(val)
}

func (enc *plainEncoder) AppendReflected(val interface{}) error {
	valueBytes, err := enc.encodeReflected(val)
	if err != nil {
		return err
	}
	enc.addElementSeparator()
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *plainEncoder) AppendString(val string) {
	enc.addElementSeparator()
	//enc.buf.AppendByte('"')
	enc.safeAddString(val)
	//enc.buf.AppendByte('"')
}

func (enc *plainEncoder) AppendTimeLayout(time time.Time, layout string) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.buf.AppendTime(time, layout)
	enc.buf.AppendByte('"')
}

func (enc *plainEncoder) AppendTime(val time.Time) {
	cur := enc.buf.Len()
	if e := enc.EncodeTime; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is a no-op. Fall back to nanos since epoch to keep
		// output JSON valid.
		enc.AppendInt64(val.UnixNano())
	}
}

func (enc *plainEncoder) AppendUint64(val uint64) {
	enc.addElementSeparator()
	enc.buf.AppendUint(val)
}

func (enc *plainEncoder) AddComplex64(k string, v complex64) { enc.AddComplex128(k, complex128(v)) }
func (enc *plainEncoder) AddFloat32(k string, v float32)     { enc.AddFloat64(k, float64(v)) }
func (enc *plainEncoder) AddInt(k string, v int)             { enc.AddInt64(k, int64(v)) }
func (enc *plainEncoder) AddInt32(k string, v int32)         { enc.AddInt64(k, int64(v)) }
func (enc *plainEncoder) AddInt16(k string, v int16)         { enc.AddInt64(k, int64(v)) }
func (enc *plainEncoder) AddInt8(k string, v int8)           { enc.AddInt64(k, int64(v)) }
func (enc *plainEncoder) AddUint(k string, v uint)           { enc.AddUint64(k, uint64(v)) }
func (enc *plainEncoder) AddUint32(k string, v uint32)       { enc.AddUint64(k, uint64(v)) }
func (enc *plainEncoder) AddUint16(k string, v uint16)       { enc.AddUint64(k, uint64(v)) }
func (enc *plainEncoder) AddUint8(k string, v uint8)         { enc.AddUint64(k, uint64(v)) }
func (enc *plainEncoder) AddUintptr(k string, v uintptr)     { enc.AddUint64(k, uint64(v)) }
func (enc *plainEncoder) AppendComplex64(v complex64)        { enc.AppendComplex128(complex128(v)) }
func (enc *plainEncoder) AppendFloat64(v float64)            { enc.appendFloat(v, 64) }
func (enc *plainEncoder) AppendFloat32(v float32)            { enc.appendFloat(float64(v), 32) }
func (enc *plainEncoder) AppendInt(v int)                    { enc.AppendInt64(int64(v)) }
func (enc *plainEncoder) AppendInt32(v int32)                { enc.AppendInt64(int64(v)) }
func (enc *plainEncoder) AppendInt16(v int16)                { enc.AppendInt64(int64(v)) }
func (enc *plainEncoder) AppendInt8(v int8)                  { enc.AppendInt64(int64(v)) }
func (enc *plainEncoder) AppendUint(v uint)                  { enc.AppendUint64(uint64(v)) }
func (enc *plainEncoder) AppendUint32(v uint32)              { enc.AppendUint64(uint64(v)) }
func (enc *plainEncoder) AppendUint16(v uint16)              { enc.AppendUint64(uint64(v)) }
func (enc *plainEncoder) AppendUint8(v uint8)                { enc.AppendUint64(uint64(v)) }
func (enc *plainEncoder) AppendUintptr(v uintptr)            { enc.AppendUint64(uint64(v)) }

func (enc *plainEncoder) Clone() zapcore.Encoder {
	clone := enc.clone()
	clone.buf.Write(enc.buf.Bytes())
	return clone
}

func (enc *plainEncoder) clone() *plainEncoder {
	clone := getPLAINEncoder()
	clone.EncoderConfig = enc.EncoderConfig
	clone.spaced = enc.spaced
	clone.openNamespaces = enc.openNamespaces
	clone.buf = getBuf()
	clone.contextFetcher = enc.contextFetcher
	return clone
}

func (enc *plainEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	final := enc.clone()

	if final.LevelKey != "" {
		final.addSeparator('[')
		//final.addKey(final.LevelKey)
		cur := final.buf.Len()
		final.EncodeLevel(ent.Level, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeLevel was a no-op. Fall back to strings to keep
			// output JSON valid.
			final.AppendString(ent.Level.String())
		}
		final.addSeparator(']')
	}
	if final.TimeKey != "" {
		//final.AddTime(final.TimeKey, ent.Time)
		final.addSeparator('[')
		enc.EncodeTime(ent.Time,final)
		final.addSeparator(']')
	}
	if ent.LoggerName != "" && final.NameKey != "" {
		final.addSeparator('[')
		final.addKey(final.NameKey)
		cur := final.buf.Len()
		nameEncoder := final.EncodeName

		// if no name encoder provided, fall back to FullNameEncoder for backwards
		// compatibility
		if nameEncoder == nil {
			nameEncoder = zapcore.FullNameEncoder
		}

		nameEncoder(ent.LoggerName, final)
		if cur == final.buf.Len() {
			// User-supplied EncodeName was a no-op. Fall back to strings to
			// keep output JSON valid.
			final.AppendString(ent.LoggerName)
		}
		final.addSeparator(']')
	}
	if ent.Caller.Defined {
		if final.CallerKey != "" {
			final.addSeparator('[')
			//final.addKey(final.CallerKey)
			cur := final.buf.Len()
			final.EncodeCaller(ent.Caller, final)
			if cur == final.buf.Len() {
				// User-supplied EncodeCaller was a no-op. Fall back to strings to
				// keep output JSON valid.
				final.AppendString(ent.Caller.String())
			}
			final.addSeparator(']')
		}
		if final.FunctionKey != "" {
			final.addSeparator('[')
			final.addKey(final.FunctionKey)
			final.AppendString(ent.Caller.Function)
			final.addSeparator(']')
		}
	}

	if enc.contextFetcher != nil {
		final.addSeparator('[')
		rid := enc.contextFetcher()
		if rid == "" {
			final.buf.AppendByte('-')
		}else {
			final.buf.AppendString(rid)
		}
		final.addSeparator(']')
	}

	if enc.buf.Len() > 0 {
		final.addElementSeparator()
		final.buf.Write(enc.buf.Bytes())
	}
	addFields(final, fields)

	if final.MessageKey != "" {
		//final.addKey(enc.MessageKey)
		final.AppendString(ent.Message)
	}

	final.closeOpenNamespaces()
	if ent.Stack != "" && final.StacktraceKey != "" {
		final.AddString(final.StacktraceKey, ent.Stack)
	}
	if final.LineEnding != "" {
		final.buf.AppendString(final.LineEnding)
	} else {
		final.buf.AppendString(zapcore.DefaultLineEnding)
	}

	ret := final.buf
	putPLAINEncoder(final)
	return ret, nil
}

func (enc *plainEncoder) truncate() {
	enc.buf.Reset()
}

func (enc *plainEncoder) closeOpenNamespaces() {
	for i := 0; i < enc.openNamespaces; i++ {
		enc.buf.AppendByte('}')
	}
}

func (enc *plainEncoder) addKey(key string) {
	enc.addElementSeparator()
	enc.safeAddString(key)
	enc.buf.AppendByte(':')
}

func (enc *plainEncoder) addElementSeparator() {
	last := enc.buf.Len() - 1
	if last < 0 {
		return
	}
	switch enc.buf.Bytes()[last] {
	case '{', '[', ':', ',', ' ':
		return
	default:
		//enc.buf.AppendByte(',')
		if enc.spaced {
			enc.buf.AppendByte(' ')
		}
	}
}

func (enc *plainEncoder) addSeparator(s byte)  {
	last := enc.buf.Len() - 1
	if last < 0 {
		enc.buf.AppendByte(s)
		return
	}
	switch enc.buf.Bytes()[last] {
	case ']':
		enc.buf.AppendByte(' ')
	}
	enc.buf.AppendByte(s)
}

func (enc *plainEncoder) appendFloat(val float64, bitSize int) {
	enc.addElementSeparator()
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		enc.buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		enc.buf.AppendString(`"-Inf"`)
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the
// user from browser vulnerabilities or JSONP-related problems.
func (enc *plainEncoder) safeAddString(s string) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (enc *plainEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.Write(s[i : i+size])
		i += size
	}
}

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a single byte.
func (enc *plainEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if 0x20 <= b && b != '\\' && b != '"' {
		enc.buf.AppendByte(b)
		return true
	}
	switch b {
	case '\\', '"':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte(b)
	case '\n':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('n')
	case '\r':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('r')
	case '\t':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		enc.buf.AppendString(`\u00`)
		enc.buf.AppendByte(_hex[b>>4])
		enc.buf.AppendByte(_hex[b&0xF])
	}
	return true
}

func (enc *plainEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}
