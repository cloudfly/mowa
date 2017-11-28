package mowa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
)

/************ API Server **************/

// Mowa represent a http server
type Mowa struct {
	sync.Mutex
	Router          // the router of server
	Addr     string // the address to listen on
	server   *http.Server
	ctx      context.Context
	listener net.Listener
}

// New create a new http server
func New(ctx context.Context) *Mowa {
	if ctx == nil {
		ctx = context.Background()
	}
	s := &Mowa{
		Router: newRouter(ctx),
		server: new(http.Server),
		ctx:    ctx,
	}
	s.server.Handler = s
	return s
}

// Run the server, and listen to given addr
func (api *Mowa) Run(addr string) error {
	api.Lock() // lock the api in case of calling Shutdown() before Serve()
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		api.Unlock()
		return err
	}

	listener, err := net.ListenTCP("tcp4", tcpAddr)
	if err != nil {
		api.Unlock()
		return err
	}
	api.listener = listener
	api.Unlock()

	return api.server.Serve(api.listener)
}

// RunWithListener serve the http service using the given listener
func (api *Mowa) RunWithListener(listener net.Listener) error {
	api.Lock()
	api.listener = listener
	api.Unlock()
	return api.server.Serve(api.listener)
}

// Shutdown the server gracefully
func (api *Mowa) Shutdown(timeout time.Duration) error {
	api.Lock()
	defer api.Unlock()
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return api.server.Shutdown(c)
}

// Listener return the net.TCPListener http service serve on
func (api *Mowa) Listener() net.Listener {
	api.Lock()
	defer api.Unlock()
	return api.listener
}

// The Router used by server
type Router interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	ServeFiles(uri string, root http.FileSystem)
	BeforeRequest(hooks ...interface{}) Router
	AfterRequest(hooks ...interface{}) Router
	Group(prefix string) Router
	Get(uri string, handler ...interface{}) Router
	Post(uri string, handler ...interface{}) Router
	Put(uri string, handler ...interface{}) Router
	Patch(uri string, handler ...interface{}) Router
	Delete(uri string, handler ...interface{}) Router
	Head(uri string, handler ...interface{}) Router
	Options(uri string, handler ...interface{}) Router
	NotFound(handler http.Handler) Router
}

// Context in every request
type Context struct {
	context.Context
	// the raw http request
	Request *http.Request
	// the http response writer
	Writer http.ResponseWriter
	// the http code to response
	Code int
	// the data to response, the data will be format to json and written into response body
	Data   interface{}
	params httprouter.Params
}

/****************** Handler *********************/

// Handler is the server handler type
type Handler struct {
	t  rune
	h0 func(c *Context)
	h1 func(c *Context) interface{}
	h2 func(c *Context) (int, interface{})
	h3 func(c *Context) (int, interface{}, bool)
}

// handler types
const (
	ht0 = iota
	ht1
	ht2
	ht3
)

// NewHandler create a new handler, the given argument must be a function
func NewHandler(f interface{}) (Handler, error) {
	switch f.(type) {
	case func(c *Context):
		return Handler{t: ht0, h0: f.(func(c *Context))}, nil
	case func(c *Context) interface{}:
		return Handler{t: ht1, h1: f.(func(c *Context) interface{})}, nil
	case func(c *Context) (int, interface{}):
		return Handler{t: ht2, h2: f.(func(c *Context) (int, interface{}))}, nil
	case func(c *Context) (int, interface{}, bool):
		return Handler{t: ht3, h3: f.(func(c *Context) (int, interface{}, bool))}, nil
	}
	return Handler{}, errors.New("unvalid function type for handler")
}

func httpRouterHandle(ctx context.Context, handlers []Handler) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		var (
			c = &Context{
				Context: ctx,
				Request: req,
				Writer:  rw,
				Code:    500,
				Data:    "",
				params:  ps,
			}
			b bool
		)

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
				log.Printf("%s\n", buf)
			}
		}()

		c.Request.ParseForm()

		// run handler
	HANDLER:
		for _, handler := range handlers {
			switch handler.t {
			case ht0:
				handler.h0(c)
			case ht1:
				c.Code, c.Data = 200, handler.h1(c)
			case ht2:
				c.Code, c.Data = handler.h2(c)
			case ht3:
				c.Code, c.Data, b = handler.h3(c)
				if b {
					break HANDLER
				}
			}
		}

		if c.Data != nil {
			content, err := json.Marshal(c.Data)
			if err != nil {
				content, _ = json.Marshal(Error("json format error, " + err.Error()))
			}

			c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
			c.Writer.WriteHeader(c.Code)
			c.Writer.Write(content)
		}
	}
}

/****************** Router *********************/

// router is default router type, a realization of Router interface
type router struct {
	ctx    context.Context
	basic  *httprouter.Router
	prefix string
	hooks  [2][]Handler // hooks[0] is pre run handler, hooks[1] is post run handler
}

// newRouter create a default router
func newRouter(ctx context.Context) *router {
	r := &router{
		ctx:    ctx,
		basic:  httprouter.New(),
		prefix: "/",
		hooks:  [2][]Handler{nil, nil},
	}
	r.basic.NotFound = new(notFoundHandler)
	return r
}

func (r *router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/debug/pprof/cmdline":
		pprof.Cmdline(rw, req)
	case "/debug/pprof/symbol":
		pprof.Symbol(rw, req)
	case "/debug/pprof/profile":
		pprof.Profile(rw, req)
	case "/debug/pprof/trace":
		pprof.Trace(rw, req)
	case "/debug/pprof/goroutine":
		pprof.Handler("goroutine").ServeHTTP(rw, req)
	case "/debug/pprof/heap":
		pprof.Handler("heap").ServeHTTP(rw, req)
	case "/debug/pprof/block":
		pprof.Handler("block").ServeHTTP(rw, req)
	case "/debug/pprof/threadcreate":
		pprof.Handler("threadcreate").ServeHTTP(rw, req)
	default:
		r.basic.ServeHTTP(rw, req)
	}
}

// ServeFiles serve the static files
func (r *router) ServeFiles(uri string, root http.FileSystem) {
	r.basic.ServeFiles(uri, root)
}

func (r *router) setHook(i int, hooks ...interface{}) Router {
	for _, hook := range hooks {
		h, err := NewHandler(hook)
		if err != nil {
			panic(err)
		}
		r.hooks[i] = append(r.hooks[i], h)
	}
	return r
}

// Before set the pre hook for router, Before will run before handlers
func (r *router) BeforeRequest(hooks ...interface{}) Router { return r.setHook(0, hooks...) }

// After set the post hook for router, After will run after handlers
func (r *router) AfterRequest(hooks ...interface{}) Router { return r.setHook(1, hooks...) }

func (r *router) Get(uri string, handler ...interface{}) Router {
	return r.Method("GET", uri, handler...)
}
func (r *router) Post(uri string, handler ...interface{}) Router {
	return r.Method("POST", uri, handler...)
}
func (r *router) Put(uri string, handler ...interface{}) Router {
	return r.Method("PUT", uri, handler...)
}
func (r *router) Patch(uri string, handler ...interface{}) Router {
	return r.Method("PATCH", uri, handler...)
}
func (r *router) Delete(uri string, handler ...interface{}) Router {
	return r.Method("DELETE", uri, handler...)
}
func (r *router) Head(uri string, handler ...interface{}) Router {
	return r.Method("HEAD", uri, handler...)
}
func (r *router) Options(uri string, handler ...interface{}) Router {
	return r.Method("OPTIONS", uri, handler...)
}
func (r *router) NotFound(handler http.Handler) Router { r.basic.NotFound = handler; return r }

// Group create a router group with the uri prefix
func (r *router) Group(prefix string) Router {
	return &router{
		ctx:    r.ctx,
		basic:  r.basic,
		prefix: path.Join(r.prefix, prefix),
	}
}

// Method is a raw function route for handler, the method can be 'GET', 'POST'...
func (r *router) Method(method, uri string, handler ...interface{}) Router {
	handlers := make([]Handler, 0, len(r.hooks[0])+len(handler)+len(r.hooks[1]))
	handlers = append(handlers, r.hooks[0]...) // run before
	for _, h := range handler {
		tmp, err := NewHandler(h)
		if err != nil {
			panic(err)
		}
		handlers = append(handlers, tmp)
	}
	handlers = append(handlers, r.hooks[1]...) // run after
	r.basic.Handle(method, path.Join(r.prefix, uri), httpRouterHandle(r.ctx, handlers))
	return r
}

type notFoundHandler struct{}

func (h *notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	content, _ := json.Marshal(map[string]string{"code": "404", "error": "page not found"})
	w.WriteHeader(404)
	w.Write(content)
}

/************* Error **************/

// DataBody is a common response format
type DataBody struct {
	Code  int         `json:"code"`            // the error code, http code encouraged
	Error string      `json:"error,emitempty"` // the error message
	Data  interface{} `json:"data"`
}

// Data return the data body with given data
func Data(data interface{}) DataBody {
	return DataBody{
		Code: 0,
		Data: data,
	}
}

// Error return DataBody with given error message
func Error(err interface{}) DataBody {
	return ErrorWithCode(1, err)
}

// ErrorWithCode return DataBody with given error message
func ErrorWithCode(code int, err interface{}) DataBody {
	d := DataBody{
		Code: code,
	}
	switch e := err.(type) {
	case error:
		d.Error = e.Error()
	case string:
		d.Error = e
	case []byte:
		d.Error = string(e)
	default:
		d.Error = fmt.Sprintf("%v", e)
	}
	return d
}
