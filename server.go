package mowa

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"net/http"
	"path"
)

/************ API Server **************/
// represent a http server
type Mowa struct {
	// the router of server
	*Router
	// the address to listen on
	Addr   string
	server *http.Server
}

// Create a new http server
func New() *Mowa {
	s := &Mowa{
		Router: NewRouter(),
		server: new(http.Server),
	}
	s.server.Handler = s
	return s
}

// Run the server, and listen to given addr
func (api *Mowa) Run(addr string) error {
	api.server.Addr = addr
	println("Starting serve on", addr)
	return api.server.ListenAndServe()
}

/****************** Router *********************/
const (
	HANDLER0 = iota
	HANDLER1
	HANDLER2
	HANDLER3
)

type Handler struct {
	t  int
	h0 func(c *Context)
	h1 func(c *Context) interface{}
	h2 func(c *Context) (int, interface{})
	h3 func(c *Context) (int, interface{}, bool)
}

// Create a new handler, the given argument must be a function
func NewHandler(f interface{}) (Handler, error) {
	switch f.(type) {
	case func(c *Context):
		return Handler{t: HANDLER0, h0: f.(func(c *Context))}, nil
	case func(c *Context) interface{}:
		return Handler{t: HANDLER1, h1: f.(func(c *Context) interface{})}, nil
	case func(c *Context) (int, interface{}):
		return Handler{t: HANDLER2, h2: f.(func(c *Context) (int, interface{}))}, nil
	case func(c *Context) (int, interface{}, bool):
		return Handler{t: HANDLER3, h3: f.(func(c *Context) (int, interface{}, bool))}, nil
	}
	return Handler{}, errors.New("invalid function type for handler")
}

func httpRouterHandle(handlers []Handler) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		var (
			c *Context = &Context{
				Context: context.TODO(),
				Request: req,
				Writer:  rw,
				Code:    500,
				Data:    "",
			}
			b bool
		)
		c.Context = context.WithValue(c.Context, "params", ps)

		// defer to recover in case of some panic, assert in context use this
		defer func() {
			if r := recover(); r != nil {
				b, _ := json.Marshal(NewError(500, "handler panic: %s", r.(error).Error()))
				c.Writer.WriteHeader(500)
				c.Writer.Write(b)
			}
		}()

		c.Request.ParseForm()

		// run handler
		for _, handler := range handlers {
			switch handler.t {
			case HANDLER0:
				handler.h0(c)
			case HANDLER1:
				c.Code, c.Data = 200, handler.h1(c)
			case HANDLER2:
				c.Code, c.Data = handler.h2(c)
			case HANDLER3:
				c.Code, c.Data, b = handler.h3(c)
				if b {
					goto RETURN
				}
			}
		}
	RETURN:
		if c.Data != nil {
			content, err := json.Marshal(c.Data)
			if err != nil {
				content, _ = json.Marshal(NewError(500, "json format error", err.Error()))
			}
			c.Writer.WriteHeader(c.Code)
			c.Writer.Write(content)
		}
	}
}

type Router struct {
	basic  *httprouter.Router
	prefix string
	hooks  [2][]Handler // hooks[0] is pre run handler, hooks[1] is post run handler
}

func NewRouter(hooks ...[]Handler) *Router {
	r := &Router{
		basic:  httprouter.New(),
		prefix: "/",
	}
	r.basic.NotFound = new(notFoundHandler)

	// set hooks
	for i := 0; i < 2; i++ {
		if i < len(hooks) && hooks[i] != nil {
			r.hooks[i] = hooks[i]
		} else {
			r.hooks[i] = make([]Handler, 0)
		}
	}
	return r
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	r.basic.ServeHTTP(rw, req)
}

func (r *Router) setHook(i int, hooks ...interface{}) *Router {
	r.hooks[i] = make([]Handler, 0, len(hooks))
	for _, hook := range hooks {
		h, err := NewHandler(hook)
		if err != nil {
			panic(err)
		}
		r.hooks[i] = append(r.hooks[i], h)
	}
	return r
}
func (r *Router) PreHook(hooks ...interface{}) *Router  { return r.setHook(0, hooks...) }
func (r *Router) PostHook(hooks ...interface{}) *Router { return r.setHook(1, hooks...) }

func (r *Router) Group(prefix string, hooks ...[]Handler) *Router {
	gr := &Router{
		basic:  r.basic,
		prefix: path.Join(r.prefix, prefix),
	}
	// combine parent hooks and given hooks
	for i := 0; i < 2; i++ {
		if i < len(hooks) && hooks[i] != nil { // having hook setting
			gr.hooks[i] = append(r.hooks[i], hooks[i]...)
		} else { // no hook setting, carry the parent's hook
			gr.hooks[i] = make([]Handler, len(r.hooks[i]))
			copy(gr.hooks[i], r.hooks[i])
		}
	}
	return gr
}

// request for /people/:name:string/:age:int will changed to:
// httprouter uri is: `/people/:name/:age`
// `:string` and `:int` setting will be stored into paramRules
func (r *Router) Method(method, uri string, handler ...interface{}) {
	var handlers []Handler = make([]Handler, 0, len(r.hooks[0])+len(handler)+len(r.hooks[1]))
	handlers = append(handlers, r.hooks[0]...)
	for _, h := range handler {
		tmp, err := NewHandler(h)
		if err != nil {
			panic(err)
		}
		handlers = append(handlers, tmp)
	}
	handlers = append(handlers, r.hooks[1]...)
	r.basic.Handle(method, path.Join(r.prefix, uri), httpRouterHandle(handlers))
}

func (r *Router) Get(uri string, handler ...interface{})     { r.Method("GET", uri, handler...) }
func (r *Router) Post(uri string, handler ...interface{})    { r.Method("POST", uri, handler...) }
func (r *Router) Put(uri string, handler ...interface{})     { r.Method("PUT", uri, handler...) }
func (r *Router) Patch(uri string, handler ...interface{})   { r.Method("PATCH", uri, handler...) }
func (r *Router) Delete(uri string, handler ...interface{})  { r.Method("DELETE", uri, handler...) }
func (r *Router) Head(uri string, handler ...interface{})    { r.Method("HEAD", uri, handler...) }
func (r *Router) Options(uri string, handler ...interface{}) { r.Method("OPTIONS", uri, handler...) }

type notFoundHandler struct{}

func (h *notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	content, _ := json.Marshal(map[string]string{"code": "404", "msg": "page not found"})
	w.WriteHeader(404)
	w.Write(content)
}

/*********** Context *****************/
type Context struct {
	context.Context
	// the raw http request
	Request *http.Request
	// the http response writer
	Writer http.ResponseWriter
	// the http code to response
	Code int
	// the data to response, the data will be format to json and written into response body
	Data interface{}
}

/************* Error **************/
type Error struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func NewError(code int, format string, v ...interface{}) error {
	return &Error{Code: code, Msg: fmt.Sprintf(format, v...)}
}

func (err *Error) Error() string {
	return err.Msg
}
