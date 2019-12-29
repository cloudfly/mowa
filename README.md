Mowa
====

[![GoDoc](http://godoc.org/github.com/cloudfly/mowa?status.svg)](http://godoc.org/github.com/cloudfly/mowa)

主要是给自己用的一个迷你 web 框架。当前 master 分之是最新的 `v3.x` 版本的代码。如果有老项目试用的老版本，请通过 tag 查找 `v1.x` 和 `v2.x` 版本
## Demo

```golang
package main

import (
	"fmt"
	"time"

	"github.com/cloudfly/mowa"
	"github.com/valyala/fasthttp"
)

func LogMW(handler interface{}) interface{} {
	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()
		mowa.NewHandler(handler)(ctx)
		fmt.Printf("%s %s cost %s\n", ctx.Method(), ctx.URI(), time.Now().Sub(start))
	}
}

func OtherMW(handler interface{}) interface{} {
	return func(ctx *fasthttp.RequestCtx) {
		mowa.NewHandler(handler)(ctx)
		fmt.Printf("other middleware\n")
	}
}

func main() {
	api := mowa.New(
		mowa.WithMiddleWare(LogMW),
	)

	// always return http code 200
	api.Get("/hello", func(c *fasthttp.RequestCtx) interface{} {
		return "hello world! /hello"
	})

	v1 := api.Group("/api/v1")
	v1.Get("/hello", OtherMW(func(c *fasthttp.RequestCtx) (int, interface{}) {
		return 202, "hello world! /api/v1/hello"
	}))

	api.Run(":8080")
}

```
