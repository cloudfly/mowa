package mowa

import (
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

// AccessLogConsoleMW print the accesslog for each request
func AccessLogConsoleMW(handler interface{}) interface{} {
	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()
		NewHandler(handler)(ctx)
		log.Printf("rip=%s method=%s path=%s code=%d cost=%s", ctx.RemoteIP(), ctx.Method(), ctx.Path(), ctx.Response.StatusCode(), time.Now().Sub(start))
	}
}
