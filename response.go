package mowa

import "fmt"

// DataBody is a common response format
type DataBody struct {
	Code  int         `json:"code"`            // the error code, http code encouraged
	Error string      `json:"error,omitempty"` // the error message
	Data  interface{} `json:"data"`
}

// Data return the data body with given data
func Data(data interface{}) DataBody {
	return DataBody{
		Code: 0,
		Data: data,
	}
}

// Error return DataBody with given error message
func Error(err interface{}) DataBody {
	return ErrorWithCode(1, err)
}

// Errorf generate error data body with given message
func Errorf(format string, params ...interface{}) DataBody {
	return ErrorWithCode(1, fmt.Errorf(format, params...))
}

// ErrorfWithCode generate error data body with given message
func ErrorfWithCode(code int, format string, params ...interface{}) DataBody {
	return ErrorWithCode(code, fmt.Errorf(format, params...))
}

// ErrorWithCode return DataBody with given error message
func ErrorWithCode(code int, err interface{}) DataBody {
	d := DataBody{
		Code: code,
	}
	switch e := err.(type) {
	case error:
		d.Error = e.Error()
	case string:
		d.Error = e
	case []byte:
		d.Error = string(e)
	default:
		d.Error = fmt.Sprintf("%v", e)
	}
	return d
}
