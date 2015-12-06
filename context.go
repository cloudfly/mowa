package mowa

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
)

// the context for each http request
type Context struct {
	context.Context
	// the raw http request
	Request *http.Request
	// the http response writer
	Writer http.ResponseWriter
	// the http code to response
	Code int
	// the data to response, the data will be format to json and written into response body
	Data interface{}
	// Return represent whether return directly, the hooks can set this value true, to interrupt the request, and response the current Code and Data directoly.
	Return bool
}

// json formatter
func (c *Context) JSON(code int, data interface{}) {
	var (
		content []byte
		err     error
	)
	if code >= 200 {
		c.Code = code
	}
	if data != nil {
		c.Data = data
	}
ENCODE:
	switch c.Data.(type) {
	case string, int, int8, int16, int32, int64, float32, float64, bool, byte:
		content, err = json.Marshal(map[string]interface{}{"data": c.Data})
	case error:
		content, err = json.Marshal(map[string]interface{}{"code": c.Code, "error": c.Data.(error).Error()})
	default:
		content, err = json.Marshal(c.Data)
	}
	if err != nil {
		c.Code = 500
		c.Data = errors.New("unvali return data")
		goto ENCODE
	}
	c.Writer.WriteHeader(c.Code)
	c.Writer.Write(content)
}

// get a string argument from request by `name`, if not found, return `str`
func (c *Context) String(name, str string) string {
	params := c.Value("params").(httprouter.Params)
	if v := params.ByName(name); v != "" {
		return v
	}
	return str
}

// get a integer argument from request by `name`, if not found, return `i`
func (c *Context) Int(name string, i int) int {
	params := c.Value("params").(httprouter.Params)
	if v := params.ByName(name); v != "" {
		if j, err := strconv.Atoi(v); err != nil {
			return i
		} else {
			return j
		}
	}
	return i
}

// get a string argument from url-query by `name`, if not found, return `str`
func (c *Context) Query(name, str string) string {
	if ret := c.Request.URL.Query().Get(name); len(ret) > 0 {
		return ret
	}
	return str
}

// get a slice argument from url-query by `name`, if not found, return `slice`
func (c *Context) QuerySlice(name string, slice []string) []string {
	if ret, ok := c.Request.URL.Query()[name]; ok {
		return ret
	}
	return slice
}

// get the request body
func (c *Context) ReadBody() []byte {
	content, err := ioutil.ReadAll(c.Request.Body)
	c.Request.Body.Close()
	if err != nil {
		return nil
	}
	return content
}

// assert function

// assert if v is a nil value, panic if not
func (c *Context) AssertNil(v interface{}) {
	if v != nil {
		panic(fmt.Errorf("%v is not nil", v))
	}
}

// assert if v is not a nil value, panic if it is
func (c *Context) AssertNotNil(v interface{}) {
	if v == nil {
		panic(fmt.Errorf("%v is nil", v))
	}
}

// assert if v1 == v2, panic if not. here using reflect.DeepEqual() to check equation
func (c *Context) AssertEqual(v1, v2 interface{}) {
	if !reflect.DeepEqual(v1, v2) {
		panic(fmt.Errorf("%v and %v is not equal", v1, v2))
	}
}

// assert if v1 != v2, panic if equal. here using reflect.DeepEqual() to check equal
func (c *Context) AssertNotEqual(v1, v2 interface{}) {
	if reflect.DeepEqual(v1, v2) {
		panic(fmt.Errorf("%v and %v is equal", v1, v2))
	}
}

// assert if v is empty, return true if it is
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

// assert if v is empty, panic if not
func (c *Context) AssertEmpty(v interface{}) {
	if !c.IsEmpty(v) {
		panic(fmt.Errorf("%v is not a zero value", v))
	}
}

// assert if v is not empty, panic if it is
func (c *Context) AssertNotEmpty(v interface{}) {
	if c.IsEmpty(v) {
		panic(fmt.Errorf("%v is a zero value", v))
	}
}
