package mowa

import (
	"path"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/expvarhandler"
	"github.com/valyala/fasthttp/pprofhandler"
)

const (
	headHook = 0
	tailHook = 1
)

// router is default router type, a realization of *router interface
type router struct {
	middlewares []Middleware
	basic       *fasthttprouter.Router
	parent      *router
	prefix      string
}

// newRouter create a default router
func newRouter() *router {
	r := &router{
		basic:  fasthttprouter.New(),
		prefix: "/",
	}

	r.basic.GET("/debug/pprof/:name", pprofhandler.PprofHandler)
	r.basic.GET("/debug/vars", expvarhandler.ExpvarHandler)
	r.basic.NotFound = notFoundHandler
	r.basic.MethodNotAllowed = notFoundHandler
	r.basic.PanicHandler = panicHandler
	return r
}

func (r *router) Handle(ctx *fasthttp.RequestCtx) {
	r.basic.Handler(ctx)
}

// ServeFiles serve the static files
func (r *router) ServeFiles(uri string, root string) {
	r.basic.ServeFiles(uri, root)
}

func (r *router) Get(uri string, handler interface{}) *router {
	return r.Method("GET", uri, handler)
}
func (r *router) Post(uri string, handler interface{}) *router {
	return r.Method("POST", uri, handler)
}
func (r *router) Put(uri string, handler interface{}) *router {
	return r.Method("PUT", uri, handler)
}
func (r *router) Patch(uri string, handler interface{}) *router {
	return r.Method("PATCH", uri, handler)
}
func (r *router) Delete(uri string, handler interface{}) *router {
	return r.Method("DELETE", uri, handler)
}
func (r *router) Head(uri string, handler interface{}) *router {
	return r.Method("HEAD", uri, handler)
}
func (r *router) Options(uri string, handler interface{}) *router {
	return r.Method("OPTIONS", uri, handler)
}
func (r *router) Any(uri string, handler interface{}) *router {
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
		r.Method(method, uri, handler)
	}
	return r
}

// Group create a router group with the uri prefix
func (r *router) Group(prefix string, middlewares ...Middleware) *router {
	return &router{
		middlewares: middlewares,
		parent:      r,
		basic:       r.basic,
		prefix:      path.Join(r.prefix, prefix),
	}
}

// Use a middleware to router
func (r *router) Use(middlewares ...Middleware) *router {
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

// Method is a raw function route for handler, the method can be 'GET', 'POST'...
func (r *router) Method(method, uri string, handler interface{}) *router {
	r.basic.Handle(method, path.Join(r.prefix, uri), r.bindMiddleware(NewHandler(handler)))
	return r
}

func (r *router) bindMiddleware(handler Handler) Handler {
	handler = Middlewares(handler, r.middlewares...)
	if r.parent != nil {
		return r.parent.bindMiddleware(handler)
	}
	return handler
}
