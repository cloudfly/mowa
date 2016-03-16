Mowa
====

This is a very very simple mini web framework written by golang.

##Demo

```golang
package main

import (
	"fmt"
	"github.com/cloudfly/mowa"
)

func preLog(c *mowa.Context) {
	fmt.Printf("%s %s\n", c.Request.Method, c.Request.URL)
}

func postLog(c *mowa.Context) {
	fmt.Printf("Response %d, %s\n", c.Code, c.Data)
}

func main() {
	api := mowa.New()
	api.PreHook(preLog)
	api.PostHook(postLog)

	api.Get("/debug", func(c *mowa.Context) (int, interface{}, bool) {
		return 200, "debug", true
	})

	v1 := api.Group("/api/v1")
	v1.Get("/hello", func(c *mowa.Context) (int, interface{}) {
		return 200, "hello world!"
	})

	api.Run(":10000")
}
```
