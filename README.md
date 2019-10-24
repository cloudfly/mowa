Mowa
====

[![GoDoc](http://godoc.org/github.com/cloudfly/mowa?status.svg)](http://godoc.org/github.com/cloudfly/mowa)

This is a very very simple mini web framework written by golang, mainly used by myself.

## Demo

```golang
import (
	"fmt"

	"github.com/cloudfly/mowa"
	"github.com/valyala/fasthttp"
)

func preLog(c *fasthttp.RequestCtx) {
	fmt.Printf("%s %s\n", c.Method(), c.URI())
}

func postLog(c *fasthttp.RequestCtx) {
	fmt.Printf("Response %s\n", c.Response.String())
}

func main() {
	api := mowa.New()
	api.BeforeRequest(preLog).AfterRequest(postLog)

	// always return http code 200
	api.Get("/hello", func(c *fasthttp.RequestCtx) interface{} {
		return "hello world! /hello"
	})

	v1 := api.Group("/api/v1")
	v1.Get("/hello", func(c *fasthttp.RequestCtx) (int, interface{}) {
		return 202, "hello world! /api/v1/hello"
	})

	api.Run(":8080")
}
```
