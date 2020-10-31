package log

import (
	//"github.com/tylerb/gls"
	tls "github.com/huandu/go-tls"
)

var (
	request_id_key = "rid"
)

func GetContextRequestID() string {

	data, ok := tls.Get(request_id_key)
	if !ok {
		return ""
	}
	value, ok := data.Value().(string)
	if !ok {
		return ""
	}
	return value

}

func SetContextRequestID(requestID string) error {
	tls.Set(request_id_key, tls.MakeData(requestID))
	return nil
}

func CleanUpContext() {
	tls.Reset()
	return
}
