package mowa

import (
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

var (
	notFoundResponse []byte
	varHandler       http.Handler
	textContentType  = "application/text; charset=utf-8"
	jsonContentType  = "application/json; charset=utf-8"
	debug            = false
)

func init() {
	notFoundResponse, _ = json.Marshal(DataBody{Code: 404, Error: "page not found"})
	varHandler = expvar.Handler()
	debug = os.Getenv("DEBUG") == "1"
}

// Handler function types supported
type (
	handleFuncRaw   = func(ctx *fasthttp.RequestCtx)
	handleFuncHook  = func(*Context)
	handleFunc      = func(*Context) interface{}
	handleFuncBreak = func(*Context) (interface{}, bool)
	handleFuncCode  = func(*Context) (int, interface{})
	handleFuncFull  = func(*Context) (int, interface{}, bool)
)

// Handler is the server handler type, switch interface{}.(type) is too slow
type Handler struct {
	f interface{}
}

// NewHandler create a new handler, the given argument must be a function
func NewHandler(f interface{}) (Handler, error) {
	switch f.(type) {
	case handleFuncRaw, handleFuncHook, handleFuncCode, handleFunc, handleFuncBreak, handleFuncFull:
		return Handler{f}, nil
	}
	return Handler{}, errors.New("unvalid function type for handler")
}

func (handler Handler) handle(ctx *Context) bool {
	continuous := true
	switch f := handler.f.(type) {
	case handleFuncRaw:
		f(ctx.RequestCtx)
	case handleFuncHook:
		f(ctx)
	case handleFunc:
		ctx.Data = f(ctx)
	case handleFuncBreak:
		ctx.Data, continuous = f(ctx)
	case handleFuncCode:
		ctx.Code, ctx.Data = f(ctx)
	case handleFuncFull:
		ctx.Code, ctx.Data, continuous = f(ctx)
	}
	return continuous
}

// Handlers  reprsents a list of handler, handler in it will be called in sort until one handler return false
type Handlers []Handler

func (handlers Handlers) handle(ctx *Context) (continuous bool) {
	for _, handler := range handlers {
		if continuous := handler.handle(ctx); !continuous {
			return false
		}
	}
	return true
}

func notFoundHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(404)
	ctx.Write(notFoundResponse)
}

func httpRouterHandler(r *router, handlers Handlers) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {

		c := &Context{
			RequestCtx: ctx,
			Code:       200,
			Data:       nil,
		}

		// run handler
		if continuous := r.processHooks(c, headHook); continuous {
			if continuous = handlers.handle(c); continuous {
				r.processHooks(c, tailHook)
			}
		}

		if c.Data != nil {
			var (
				content []byte
				err     error
			)
			switch d := c.Data.(type) {
			case string:
				content = []byte(d)
				c.Response.Header.Set("Content-Type", textContentType)
			case []byte:
				content = d
				c.Response.Header.Set("Content-Type", textContentType)
			default:
				content, err = json.Marshal(c.Data)
				if err != nil {
					content, _ = json.Marshal(Error("json format error, " + err.Error()))
				}
				c.Response.Header.Set("Content-Type", jsonContentType)
			}
			c.SetStatusCode(c.Code)
			c.Write(content)
		}
	}
}

// Recovery 代表内置的 recover 函数, 它返回 panic 简单信息, 并打印 goroutine stack 信息到错误输出
func Recovery(ctx *fasthttp.RequestCtx, err interface{}) {
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

func pprofHandler(ctx *fasthttp.RequestCtx) {
	name := fmt.Sprintf("%s", ctx.UserValue("name"))
	switch name {
	case "cmdline":
		ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
		ctx.Response.Header.Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(ctx, strings.Join(os.Args, "\x00"))
	case "profile":
		ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
		sec, err := strconv.ParseInt(string(ctx.FormValue("seconds")), 10, 64)
		if sec <= 0 || err != nil {
			sec = 30
		}

		// Set Content Type assuming StartCPUProfile will work,
		// because if it does it starts writing.
		ctx.Response.Header.Set("Content-Type", "application/octet-stream")
		ctx.Response.Header.Set("Content-Disposition", `attachment; filename="profile"`)
		if err := pprof.StartCPUProfile(ctx); err != nil {
			// StartCPUProfile failed, so no writes yet.
			debugResponseError(ctx, http.StatusInternalServerError,
				fmt.Sprintf("Could not enable CPU profiling: %s", err))
			return
		}
		sleep(ctx, time.Second*time.Duration(sec))
		pprof.StopCPUProfile()
	case "trace":
		ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
		sec, err := strconv.ParseInt(string(ctx.FormValue("seconds")), 10, 64)
		if sec <= 0 || err != nil {
			sec = 1
		}

		// Set Content Type assuming trace.Start will work,
		// because if it does it starts writing.
		ctx.Response.Header.Set("Content-Type", "application/octet-stream")
		ctx.Response.Header.Set("Content-Disposition", `attachment; filename="trace"`)
		if err := trace.Start(ctx); err != nil {
			// trace.Start failed, so no writes yet.
			debugResponseError(ctx, http.StatusInternalServerError,
				fmt.Sprintf("Could not enable tracing: %s", err))
			return
		}
		sleep(ctx, time.Second*time.Duration(sec))
		trace.Stop()
	default:
		ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
		p := pprof.Lookup(name)
		if p == nil {
			debugResponseError(ctx, http.StatusNotFound, "Unknown profile")
			return
		}
		gc, _ := strconv.Atoi(string(ctx.FormValue("gc")))
		if name == "heap" && gc > 0 {
			runtime.GC()
		}
		debug, _ := strconv.Atoi(string(ctx.FormValue("debug")))
		if debug != 0 {
			ctx.Response.Header.Set("Content-Type", "text/plain; charset=utf-8")
		} else {
			ctx.Response.Header.Set("Content-Type", "application/octet-stream")
			ctx.Response.Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
		}
		p.WriteTo(ctx, debug)
	}
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
