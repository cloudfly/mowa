package mowa

import (
	"fmt"
	"strconv"

	"github.com/valyala/fasthttp"
)

// Context in every request
type Context struct {
	*fasthttp.RequestCtx
	Code int
	// the data to response, the data will be format to json and written into response body
	Data interface{}
}

// String get a string argument from request by `name`, if not found, return `str`
func (c *Context) String(name, str string) string {
	v := c.RequestCtx.UserValue(name)
	if v == nil {
		return str
	}
	return fmt.Sprintf("%s", v)
}

// Int get a integer argument from request by `name`, if not found, return `i`
func (c *Context) Int(name string, i int) int {
	v := c.RequestCtx.UserValue(name)
	if v == nil {
		return i
	}
	if j, err := strconv.Atoi(fmt.Sprintf("%s", v)); err == nil {
		return j
	}
	return i
}

// Int64 get a integer argument from request by `name`, if not found, return `i`
func (c *Context) Int64(name string, i int64) int64 {
	v := c.RequestCtx.UserValue(name)
	if v == nil {
		return i
	}
	if j, err := strconv.ParseInt(fmt.Sprintf("%s", v), 10, 64); err == nil {
		return j
	}
	return i
}

// ReadBody read the content from request body
func (c *Context) ReadBody() []byte {
	return c.RequestCtx.Request.Body()
}
