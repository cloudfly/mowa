package mowa

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"io/ioutil"
)

func ParamChecker(c *Context) (int, interface{}) {
	paramRules := c.Ctx.Value("param-rules").(map[string][]string)
	params := c.Ctx.Value("params").(httprouter.Params)
	// check params
	for name, rules := range paramRules {
		value := params.ByName(name)
		if len(rules) > 0 {
			if err := c.TestValue(name, value, rules); err != nil {
				c.Return = true
				return 403, err
			}
		}
	}
	return 0, nil
}

func ParseJSONBody(c *Context) (int, interface{}) {
	if c.Request.Header.Get("Content-Type") != "application/json" {
		return 200, nil
	}
	var data map[string]interface{}
	content, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.Return = true
		return 403, NewError(403, "fail to read body from request")
	}

	c.Ctx = context.WithValue(c.Ctx, "body-bytes", content)
	if err := json.Unmarshal(content, &data); err != nil {
		c.Return = true
		return 403, NewError(403, "request body is not json-format")
	}
	c.Ctx = context.WithValue(c.Ctx, "body", data)
	return 200, data
}
