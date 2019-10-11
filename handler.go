package mowa

import (
	"encoding/json"
	"errors"
	"expvar"
	"log"
	"net/http"
	"runtime"

	"github.com/julienschmidt/httprouter"
)

var (
	notFoundResponse []byte
	varHandler       http.Handler
	textContentType  = []string{"application/text; charset=utf-8"}
	jsonContentType  = []string{"application/json; charset=utf-8"}
)

func init() {
	notFoundResponse, _ = json.Marshal(DataBody{Code: 404, Error: "page not found"})
	varHandler = expvar.Handler()
}

// handler types
const (
	raw = iota // raw http.Handler
	ht0
	ht1
	ht2
	ht3
	ht4
)

// Handler is the server handler type, switch interface{}.(type) is too slow
type Handler struct {
	t   rune
	raw func(http.ResponseWriter, *http.Request)
	h0  func(*Context)
	h1  func(*Context) interface{}
	h2  func(*Context) (int, interface{})
	h3  func(*Context) (int, interface{}, bool)
	h4  func(*Context) (interface{}, bool)
}

// NewHandler create a new handler, the given argument must be a function
func NewHandler(f interface{}) (Handler, error) {
	switch handler := f.(type) {
	case http.Handler:
		return Handler{t: raw, raw: handler.ServeHTTP}, nil
	case func(c *Context):
		return Handler{t: ht0, h0: handler}, nil
	case func(c *Context) interface{}:
		return Handler{t: ht1, h1: handler}, nil
	case func(c *Context) (int, interface{}):
		return Handler{t: ht2, h2: handler}, nil
	case func(c *Context) (int, interface{}, bool):
		return Handler{t: ht3, h3: handler}, nil
	case func(c *Context) (interface{}, bool):
		return Handler{t: ht4, h4: handler}, nil
	}
	return Handler{}, errors.New("unvalid function type for handler")
}

func (handler Handler) handle(ctx *Context) bool {
	continuous := true
	switch handler.t {
	case raw:
		handler.raw(ctx.Writer, ctx.Request)
	case ht0:
		handler.h0(ctx)
	case ht1:
		ctx.Data = handler.h1(ctx)
	case ht2:
		ctx.Code, ctx.Data = handler.h2(ctx)
	case ht3:
		ctx.Code, ctx.Data, continuous = handler.h3(ctx)
	case ht4:
		ctx.Data, continuous = handler.h4(ctx)
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
			if r := recover(); r != nil {
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
			}
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
