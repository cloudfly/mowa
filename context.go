package mowa

import (
	"context"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// Context in every request
type Context struct {
	context.Context
	// the raw http request
	Request *http.Request
	// the http response writer
	Writer http.ResponseWriter
	// the http code to response
	Code int
	// the data to response, the data will be format to json and written into response body
	Data   interface{}
	params httprouter.Params
}

// String get a string argument from request by `name`, if not found, return `str`
func (c *Context) String(name, str string) string {
	if v := c.params.ByName(name); v != "" {
		return v
	}
	return str
}

// Int get a integer argument from request by `name`, if not found, return `i`
func (c *Context) Int(name string, i int) int {
	if v := c.params.ByName(name); v != "" {
		if j, err := strconv.Atoi(v); err == nil {
			return j
		}
		return i
	}
	return i
}

// Int64 get a integer argument from request by `name`, if not found, return `i`
func (c *Context) Int64(name string, i int64) int64 {
	if v := c.params.ByName(name); v != "" {
		if j, err := strconv.ParseInt(v, 10, 64); err == nil {
			return j
		}
		return i
	}
	return i
}

// Query get a string argument from url-query by `name`, if not found, return `str`
func (c *Context) Query(name, str string) string {
	if ret := c.Request.URL.Query().Get(name); len(ret) > 0 {
		return ret
	}
	return str
}

// QueryInt get a int64 argument from url-query by `name`, if not found or not a integer, return `str`
func (c *Context) QueryInt(name string, i int64) int64 {
	if ret := c.Request.URL.Query().Get(name); len(ret) > 0 {
		if i64, err := strconv.ParseInt(ret, 10, 64); err == nil {
			return i64
		}
	}
	return i
}

// QueryFloat get a float64 argument from url-query by `name`, if not found or not a integer, return `str`
func (c *Context) QueryFloat(name string, f float64) float64 {
	if ret := c.Request.URL.Query().Get(name); len(ret) > 0 {
		if f64, err := strconv.ParseFloat(ret, 64); err == nil {
			return f64
		}
	}
	return f
}

// QuerySlice get a slice argument from url-query by `name`, if not found, return `slice`
func (c *Context) QuerySlice(name string, slice []string) []string {
	if ret, ok := c.Request.URL.Query()[name]; ok {
		return ret
	}
	return slice
}

// ReadBody read the content from request body
func (c *Context) ReadBody() []byte {
	content, err := ioutil.ReadAll(c.Request.Body)
	c.Request.Body.Close()
	if err != nil {
		return nil
	}
	return content
}

// IsEmpty check if v is empty, return true if it is
// empty means:
// 1. zero for number, int or float
// 2. zero length for string,chan,map and array.
// 3. false for bool
// 4. nil for interface
func (c *Context) IsEmpty(v interface{}) bool {
	rv := reflect.ValueOf(v)
	switch reflect.TypeOf(v).Kind() {
	case reflect.Bool:
		return rv.Bool() == false
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0.0
	case reflect.Slice, reflect.Array, reflect.Chan, reflect.Map, reflect.String:
		return rv.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v == nil
	default:
		return false
	}
}
