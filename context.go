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

type Context struct {
	Ctx     context.Context
	Request *http.Request
	Writer  http.ResponseWriter
	code    int
	data    interface{}
	Return  bool
}

func (c *Context) JSON(code int, data interface{}) {
	c.code = code
	c.data = data
ENCODE:
	content, err := json.Marshal(data)
	if err != nil {
		c.code = 500
		c.data = errors.New("unvali return data")
		goto ENCODE
	}
	c.Writer.WriteHeader(c.code)
	c.Writer.Write(content)
}

// request data fetch function

func (c *Context) String(name, str string) string {
	params := c.Ctx.Value("params").(httprouter.Params)
	if v := params.ByName(name); v != "" {
		return v
	}
	return str
}

func (c *Context) Int(name string, i int) int {
	params := c.Ctx.Value("params").(httprouter.Params)
	if v := params.ByName(name); v != "" {
		if j, err := strconv.Atoi(v); err != nil {
			return i
		} else {
			return j
		}
	}
	return i
}

func (c *Context) Query(name, str string) string {
	if ret := c.Request.URL.Query().Get(name); len(ret) > 0 {
		return ret
	}
	return str
}

func (c *Context) QuerySlice(name string, slice []string) []string {
	if ret, ok := c.Request.URL.Query()[name]; ok {
		return ret
	}
	return slice
}

func (c *Context) ReadBody() []byte {
	content, err := ioutil.ReadAll(c.Request.Body)
	c.Request.Body.Close()
	if err != nil {
		return nil
	}
	return content
}

// assert function

func (c *Context) AssertNil(v interface{}) {
	if v != nil {
		panic(fmt.Errorf("%v is not nil", v))
	}
}

func (c *Context) AssertNotNil(v interface{}) {
	if v == nil {
		panic(fmt.Errorf("%v is nil", v))
	}
}

func (c *Context) AssertEqual(v1, v2 interface{}) {
	if !reflect.DeepEqual(v1, v2) {
		panic(fmt.Errorf("%v and %v is not equal", v1, v2))
	}
}

func (c *Context) AssertNotEqual(v1, v2 interface{}) {
	if reflect.DeepEqual(v1, v2) {
		panic(fmt.Errorf("%v and %v is equal", v1, v2))
	}
}

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

func (c *Context) AssertEmpty(v interface{}) {
	if !c.IsEmpty(v) {
		panic(fmt.Errorf("%v is not a zero value", v))
	}
}
func (c *Context) AssertNotEmpty(v interface{}) {
	if c.IsEmpty(v) {
		panic(fmt.Errorf("%v is a zero value", v))
	}
}
