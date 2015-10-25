package mowa

import (
	"fmt"
)

type Error struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Cause string `json:"cause"`
}

func NewError(code int, format string, v ...interface{}) error {
	return &Error{
		Code: code,
		Msg:  fmt.Sprintf(format, v...),
	}
}

func (err *Error) Error() string {
	return err.Msg
}
