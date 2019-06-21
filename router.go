package mowa

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/pprof"
	"path"
	"reflect"
	"runtime"
	"strings"

	"github.com/julienschmidt/httprouter"
)

const (
	headHook = 0
	tailHook = 1
)

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
	AddResource(uri string, resource interface{}) Router
	Any(uri string, handler ...interface{}) Router
	Method(method, uri string, handler ...interface{}) Router
	NotFound(handler http.Handler) Router
}

// router is default router type, a realization of Router interface
type router struct {
	ctx    context.Context
	parent *router
	basic  *httprouter.Router
	prefix string
	hooks  [2]Handlers // hooks[0] is pre run handler, hooks[1] is post run handler
}

// newRouter create a default router
func newRouter(ctx context.Context) *router {
	r := &router{
		ctx:    ctx,
		basic:  httprouter.New(),
		prefix: "/",
		hooks:  [2]Handlers{nil, nil},
	}
	r.basic.NotFound = notFoundHandler{}
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
	case "/debug/vars":
		varHandler.ServeHTTP(rw, req)
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

func (r *router) processHooks(ctx *Context, hookIndex int) (continuous bool) {
	if r.parent != nil {
		if continuous := r.parent.processHooks(ctx, hookIndex); !continuous {
			return false
		}
	}
	return r.hooks[hookIndex].handle(ctx)
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
func (r *router) Any(uri string, handler ...interface{}) Router {
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
		r.Method(method, uri, handler...)
	}
	return r
}
func (r *router) NotFound(handler http.Handler) Router { r.basic.NotFound = handler; return r }

// Group create a router group with the uri prefix
func (r *router) Group(prefix string) Router {
	return &router{
		parent: r,
		ctx:    r.ctx,
		basic:  r.basic,
		prefix: path.Join(r.prefix, prefix),
		hooks:  r.hooks,
	}
}

// Method is a raw function route for handler, the method can be 'GET', 'POST'...
func (r *router) Method(method, uri string, handler ...interface{}) Router {
	handlers := make([]Handler, 0, len(handler))
	for _, h := range handler {
		tmp, err := NewHandler(h)
		if err != nil {
			panic(err)
		}
		handlers = append(handlers, tmp)
	}
	r.basic.Handle(method, path.Join(r.prefix, uri), httpRouterHandler(r, handlers))
	return r
}

func (r *router) AddResource(uri string, resource interface{}) Router {
	value := reflect.ValueOf(resource)
	beforeMethod := value.MethodByName("BeforeRequest")
	afterMethod := value.MethodByName("AfterRequest")
	for _, name := range [...]string{"Get", "Post", "Put", "Patch", "Delete", "Head", "Options"} {
		method := value.MethodByName(name)
		if !method.IsValid() {
			continue
		}
		handlers := make([]interface{}, 0, 3)
		if beforeMethod.IsValid() {
			handlers = append(handlers, beforeMethod.Interface())
		}
		handlers = append(handlers, method.Interface())
		if afterMethod.IsValid() {
			handlers = append(handlers, afterMethod.Interface())
		}
		r.Method(strings.ToUpper(name), uri, handlers...)
	}
	return r
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
				c.Writer.Header().Set("Content-Type", "application/text; charset=utf-8")
			case []byte:
				content = d
				c.Writer.Header().Set("Content-Type", "application/text; charset=utf-8")
			default:
				content, err = json.Marshal(c.Data)
				if err != nil {
					content, _ = json.Marshal(Error("json format error, " + err.Error()))
				}
				c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
			}

			c.Writer.WriteHeader(c.Code)
			c.Writer.Write(content)
		}
	}
}
