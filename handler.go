package mowa

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
)

var (
	notFoundResponse []byte
	textContentType  = "application/text; charset=utf-8"
	jsonContentType  = "application/json; charset=utf-8"
)

func init() {
	notFoundResponse, _ = json.Marshal(DataBody{Code: 404, Error: "page not found"})
}

// Handler function types supported
type (
	handleFuncRaw   = func(*fasthttp.RequestCtx)
	handleFunc      = func(*fasthttp.RequestCtx) interface{}
	handleFuncBreak = func(*fasthttp.RequestCtx) (interface{}, bool)
	handleFuncCode  = func(*fasthttp.RequestCtx) (int, interface{})
	handleFuncFull  = func(*fasthttp.RequestCtx) (int, interface{}, bool)
)

// Handler is the server handler type, switch interface{}.(type) is too slow
type Handler struct {
	f interface{}
}

// NewHandler create a new handler, the given argument must be a function
func NewHandler(f interface{}) (Handler, error) {
	switch f.(type) {
	case handleFuncRaw, handleFuncCode, handleFunc, handleFuncBreak, handleFuncFull:
		return Handler{f}, nil
	}
	return Handler{}, errors.New("unvalid function type for handler")
}

func (handler Handler) handle(ctx *fasthttp.RequestCtx, code *int, data *struct{ data interface{} }, continuous *bool) {
	*code, *continuous = 200, true
	switch f := handler.f.(type) {
	case handleFuncRaw:
		f(ctx)
	case handleFunc:
		*code = 200
		data.data = f(ctx)
	case handleFuncBreak:
		data.data, *continuous = f(ctx)
	case handleFuncCode:
		*code, data.data = f(ctx)
	case handleFuncFull:
		*code, data.data, *continuous = f(ctx)
	}
}

// Handlers  reprsents a list of handler, handler in it will be called in sort until one handler return false
type Handlers []Handler

func (handlers Handlers) handle(ctx *fasthttp.RequestCtx, code *int, data *struct{ data interface{} }, continuous *bool) {
	for _, handler := range handlers {
		handler.handle(ctx, code, data, continuous)
		if !*continuous {
			return
		}
	}
}

func notFoundHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(404)
	ctx.Write(notFoundResponse)
}

func httpRouterHandler(r *router, handlers Handlers) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var (
			code       int
			data       struct{ data interface{} }
			continuous = true
		)

		// run handler
		r.processHooks(ctx, headHook, &code, &data, &continuous)
		if continuous {
			handlers.handle(ctx, &code, &data, &continuous)
			if continuous {
				r.processHooks(ctx, tailHook, &code, &data, &continuous)
			}
		}

		if data.data != nil {
			var (
				content []byte
				err     error
			)
			switch d := data.data.(type) {
			case string:
				content = []byte(d)
				ctx.Response.Header.Set("Content-Type", textContentType)
			case []byte:
				content = d
				ctx.Response.Header.Set("Content-Type", textContentType)
			default:
				content, err = json.Marshal(data.data)
				if err != nil {
					content, _ = json.Marshal(Error("json format error, " + err.Error()))
				}
				ctx.Response.Header.Set("Content-Type", jsonContentType)
			}
			ctx.SetStatusCode(code)
			ctx.Write(content)
			return
		}
		ctx.SetStatusCode(204)
	}
}

// panicHandler 代表内置的 recover 函数, 它返回 panic 简单信息, 并打印 goroutine stack 信息到错误输出
func panicHandler(ctx *fasthttp.RequestCtx, err interface{}) {
	errs := ""
	switch rr := err.(type) {
	case string:
		errs = rr
	case error:
		errs = rr.Error()
	}
	b, _ := json.Marshal(Error(errs))
	ctx.Response.Header.Set("Content-Type", "application/json; charset=utf-8")
	ctx.Response.SetStatusCode(500)
	ctx.Write(b)

	buf := make([]byte, 1024*64)
	runtime.Stack(buf, false)
	log.Printf("----------------------------------------------------------------")
	log.Printf("%s\n%s\n", errs, buf)
	log.Printf("----------------------------------------------------------------")
}

func debugResponseError(ctx *fasthttp.RequestCtx, status int, txt string) {
	ctx.Response.Header.Set("Content-Type", "text/plain; charset=utf-8")
	ctx.Response.Header.Set("X-Go-Pprof", "1")
	ctx.Response.Header.Del("Content-Disposition")
	ctx.SetStatusCode(status)
	fmt.Fprintln(ctx, txt)
}

func sleep(ctx *fasthttp.RequestCtx, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}

// StringValue get a string argument from request by `name`, if not found, return `str`
func StringValue(ctx *fasthttp.RequestCtx, name, str string) string {
	v := ctx.UserValue(name)
	if v == nil {
		return str
	}
	return fmt.Sprintf("%s", v)
}

// IntValue get a integer argument from request by `name`, if not found, return `i`
func IntValue(ctx *fasthttp.RequestCtx, name string, i int) int {
	v := ctx.UserValue(name)
	if v == nil {
		return i
	}
	if j, err := strconv.Atoi(fmt.Sprintf("%s", v)); err == nil {
		return j
	}
	return i
}

// Int64Value get a integer argument from request by `name`, if not found, return `i`
func Int64Value(ctx *fasthttp.RequestCtx, name string, i int64) int64 {
	v := ctx.UserValue(name)
	if v == nil {
		return i
	}
	if j, err := strconv.ParseInt(fmt.Sprintf("%s", v), 10, 64); err == nil {
		return j
	}
	return i
}
