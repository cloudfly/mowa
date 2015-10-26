package mowa

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"net/http"
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
		c.data = NewError(500, "unvalid return data")
		goto ENCODE
	}
	c.Writer.WriteHeader(c.code)
	c.Writer.Write(content)
}

func (c *Context) TestValue(name, value string, rules []string) error {
	for _, rule := range rules {
		switch rule {
		case "int":
			if _, err := strconv.Atoi(value); err != nil {
				return NewError(403, "given param %s = %s, not a integer", name, value)
			}
		}
	}
	return nil
}

func (c *Context) Assert(name, value string, rules []string) {
	if err := c.TestValue(name, value, rules); err != nil {
		c.JSON(403, err)
	}
}

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

func (c *Context) BodyData(name string) interface{} {
	data := c.Ctx.Value("body").(map[string]interface{})
	if data == nil {
		return nil
	}
	if value, ok := data[name]; ok {
		return value
	}
	return nil
}
