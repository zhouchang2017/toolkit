package log

import (
	"testing"
)

func TestNewZapLogger(t *testing.T) {
	_debugHiddenTime = true
	logger := NewZapLogger("json", true, "debug")

	logger.Info("info msg")

	logger.WithFields(map[string]interface{}{
		"cost":100,
		"app":"sync-iot",
	}).Info("with info msg")
}