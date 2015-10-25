package mowa

import (
//	"fmt"
)

func ParamChecker(c *Context) (int, interface{}) {
	// check params
	for name, rules := range c.ParamRules {
		value := c.Params.ByName(name)
		if len(rules) > 0 {
			if err := c.TestValue(name, value, rules); err != nil {
				c.JSON(403, err)
				c.Return = true
				break
			}
		}
	}
	return 0, nil
}
