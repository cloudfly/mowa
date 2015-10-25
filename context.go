package myapi

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strconv"
)

type Context struct {
	Request    *http.Request
	Writer     http.ResponseWriter
	code       int
	data       interface{}
	Return     bool
	Params     httprouter.Params
	ParamRules map[string][]string
	// TODO post form
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

func (c *Context) String(name, defaultS string) string {
	if v := c.Params.ByName(name); v == "" {
		return defaultS
	} else {
		return v
	}
}

func (c *Context) Int(name string, defaultI int) int {

	if v := c.Params.ByName(name); v == "" {
		return defaultI
	} else {
		if i, err := strconv.Atoi(v); err != nil {
			return defaultI
		} else {
			return i
		}
	}
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

func (c *Context) Form(name, str string) string {
	if ret := c.Request.PostForm.Get(name); len(ret) > 0 {
		return ret
	}
	return str
}

func (c *Context) FormSlice(name string, slice []string) []string {
	if ret, ok := c.Request.PostForm[name]; ok {
		return ret
	}
	return slice
}
