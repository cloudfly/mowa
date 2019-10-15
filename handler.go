package mowa

import (
	"encoding/json"
	"errors"
	"expvar"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"

	"github.com/julienschmidt/httprouter"
)

var (
	notFoundResponse []byte
	varHandler       http.Handler
	textContentType  = []string{"application/text; charset=utf-8"}
	jsonContentType  = []string{"application/json; charset=utf-8"}
	debug            = false
)

func init() {
	notFoundResponse, _ = json.Marshal(DataBody{Code: 404, Error: "page not found"})
	varHandler = expvar.Handler()
	debug = os.Getenv("DEBUG") == "1"
}

// Handler function types supported
type (
	handleFuncRaw   = func(http.ResponseWriter, *http.Request)
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
		f(ctx.Writer, ctx.Request)
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

type notFoundHandler struct{}

func (h notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	w.Write(notFoundResponse)
}

func httpRouterHandler(r *router, handlers Handlers) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {

		if debug {
			info, err := httputil.DumpRequest(req, true)
			if err != nil {
				log.Printf("[ERROR] failed to dump http request: %s", err.Error())
			} else {
				log.Printf("Incoming HTTP Request:")
				log.Printf("%s", info)
			}
		}

		c := &Context{
			Context: r.ctx,
			Request: req,
			Writer:  rw,
			Code:    200,
			Data:    nil,
			params:  ps,
		}

		// defer to recover in case of some panic, assert in context use this
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			errs := ""
			switch rr := r.(type) {
			case string:
				errs = rr
			case error:
				errs = rr.Error()
			}
			b, _ := json.Marshal(Error(errs))
			c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
			c.Writer.WriteHeader(500)
			c.Writer.Write(b)

			buf := make([]byte, 1024*64)
			runtime.Stack(buf, false)
			log.Printf("%s\n%s\n", errs, buf)
		}()

		c.Request.ParseForm()
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
				c.Writer.Header()["Content-Type"] = textContentType
			case []byte:
				content = d
				c.Writer.Header()["Content-Type"] = textContentType
			default:
				content, err = json.Marshal(c.Data)
				if err != nil {
					content, _ = json.Marshal(Error("json format error, " + err.Error()))
				}
				c.Writer.Header()["Content-Type"] = jsonContentType
			}

			c.Writer.WriteHeader(c.Code)
			c.Writer.Write(content)
		}
	}
}
