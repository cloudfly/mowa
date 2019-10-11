package mowa

import (
	"context"
	"net/http"
	"net/http/pprof"
	"path"

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
	if len(r.hooks[hookIndex]) > 0 {
		return r.hooks[hookIndex].handle(ctx)
	}
	return true
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
