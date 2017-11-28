Mowa
====

[![GoDoc](http://godoc.org/github.com/cloudfly/mowa?status.svg)](http://godoc.org/github.com/cloudfly/mowa)

This is a very very simple mini web framework written by golang, mainly used by myself.

## Demo

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
	api.BeforeRequest(preLog).AfterRequest(postLog)


    // always return http code 200
	api.Get("/hello", func(c *mowa.Context) interface{} {
		return "hello world! /hello"
	})

	v1 := api.Group("/api/v1")
	v1.Get("/hello", func(c *mowa.Context) (int, interface{}) {
		return 202, "hello world! /api/v1/hello"
	})

	api.Run(":8080")
}
```
