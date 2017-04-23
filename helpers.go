package mowa

import (
	"io/ioutil"
	"reflect"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// String get a string argument from request by `name`, if not found, return `str`
func (c *Context) String(name, str string) string {
	params := c.Value("params").(httprouter.Params)
	if v := params.ByName(name); v != "" {
		return v
	}
	return str
}

// Int get a integer argument from request by `name`, if not found, return `i`
func (c *Context) Int(name string, i int) int {
	params := c.Value("params").(httprouter.Params)
	if v := params.ByName(name); v != "" {
		if j, err := strconv.Atoi(v); err == nil {
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
