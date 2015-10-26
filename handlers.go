package mowa

import (
	"github.com/julienschmidt/httprouter"
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
