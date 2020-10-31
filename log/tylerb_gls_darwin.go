package log

import (
	"github.com/tylerb/gls"
)

var (
	request_id_key = "rid"
)

func GetContextRequestID() string {
	if request_id, ok := gls.Get(request_id_key).(string); ok {
		return request_id
	} else {
		return ""
	}
}

func SetContextRequestID(requestID string) error {
	gls.Set(request_id_key, requestID)
	return nil
}

func CleanUpContext() {
	gls.Cleanup()
	return
}
