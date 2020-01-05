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
	textContentType  = "text/plain; charset=utf-8"
	jsonContentType  = "application/json; charset=utf-8"
)

func init() {
	notFoundResponse, _ = json.Marshal(DataBody{Code: 404, Error: "page not found"})
}

// Handler function types supported
type (
	handleFuncRaw  = func(*fasthttp.RequestCtx)
	handleFunc     = func(*fasthttp.RequestCtx) interface{}
	handleFuncCode = func(*fasthttp.RequestCtx) (int, interface{})
)

// Handler is a alias of fasthttp.RequestHandler
type Handler = fasthttp.RequestHandler

// NewHandler create a new handler, the given argument must be a valid function, otherwise it will panic
func NewHandler(f interface{}) Handler {
	h, err := NewHandler2(f)
	if err != nil {
		panic(err)
	}
	return h
}

// NewHandler2 create a new handler, if return error if the given argument is not a valid function
func NewHandler2(f interface{}) (Handler, error) {
	switch fn := f.(type) {
	case fasthttp.RequestHandler:
		return fn, nil
	case handleFuncRaw:
		return Handler(fn), nil
	case handleFuncCode:
		return func(ctx *fasthttp.RequestCtx) {
			code, data := fn(ctx)
			handleCodeData(ctx, code, data)
		}, nil
	case handleFunc:
		return func(ctx *fasthttp.RequestCtx) {
			data := fn(ctx)
			handleCodeData(ctx, 0, data)
		}, nil
	}
	return nil, errors.New("unvalid function type for handler")
}

func notFoundHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(404)
	ctx.Write(notFoundResponse)
}

func handleCodeData(ctx *fasthttp.RequestCtx, code int, data interface{}) {
	if code > 0 {
		ctx.SetStatusCode(code)
	}
	if data != nil {
		ctx.ResetBody()
		var (
			content []byte
			err     error
		)
		switch d := data.(type) {
		case string:
			content = []byte(d)
			ctx.Response.Header.Set("Content-Type", textContentType)
		case []byte:
			content = d
			ctx.Response.Header.Set("Content-Type", textContentType)
		default:
			content, err = json.Marshal(data)
			if err != nil {
				content, _ = json.Marshal(Error("json format error, " + err.Error()))
			}
			ctx.Response.Header.Set("Content-Type", jsonContentType)
		}
		ctx.Write(content)
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
