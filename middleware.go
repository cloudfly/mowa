package mowa

import (
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

// Middleware is a alias of fasthttp.Requestmiddlewarer
type Middleware func(*fasthttp.RequestCtx, Handler)

// Middlewares combine multiple middlewares to one and return the result
func Middlewares(handler Handler, mws ...Middleware) Handler {
	if len(mws) == 0 {
		return handler
	}
	return Middlewares(
		func(ctx *fasthttp.RequestCtx) {
			mws[len(mws)-1](ctx, handler)
		},
		mws[:len(mws)-1]...,
	)
}

// AccessLogConsoleMW print the accesslog for each request
func AccessLogConsoleMW(handler interface{}) interface{} {
	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()
		NewHandler(handler)(ctx)
		log.Printf("rip=%s method=%s path=%s code=%d cost=%s", ctx.RemoteIP(), ctx.Method(), ctx.Path(), ctx.Response.StatusCode(), time.Now().Sub(start))
	}
}
