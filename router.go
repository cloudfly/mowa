package mowa

import (
	"path"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

const (
	headHook = 0
	tailHook = 1
)

// router is default router type, a realization of *router interface
type router struct {
	basic  *fasthttprouter.Router
	parent *router
	prefix string
	hooks  [2]Handlers // hooks[0] is pre run handler, hooks[1] is post run handler
}

// newRouter create a default router
func newRouter() *router {
	r := &router{
		basic:  fasthttprouter.New(),
		prefix: "/",
		hooks:  [2]Handlers{nil, nil},
	}

	r.basic.GET("/debug/pprof/:name", pprofHandler)
	r.basic.NotFound = notFoundHandler
	r.basic.MethodNotAllowed = notFoundHandler
	r.basic.PanicHandler = panicHandler
	return r
}

func (r *router) Handler(ctx *fasthttp.RequestCtx) {
	r.basic.Handler(ctx)
}

// ServeFiles serve the static files
func (r *router) ServeFiles(uri string, root string) {
	r.basic.ServeFiles(uri, root)
}

func (r *router) setHook(i int, hooks ...interface{}) *router {
	for _, hook := range hooks {
		h, err := NewHandler(hook)
		if err != nil {
			panic(err)
		}
		r.hooks[i] = append(r.hooks[i], h)
	}
	return r
}

func (r *router) processHooks(ctx *fasthttp.RequestCtx, hookIndex int) (int, interface{}, bool) {
	var (
		code int
		data interface{}
	)
	if r.parent != nil {
		c, d, b := r.parent.processHooks(ctx, hookIndex)
		if c > 0 {
			code = c
		}
		if d != nil {
			data = d
		}
		if !b {
			return code, data, false
		}
	}
	if len(r.hooks[hookIndex]) > 0 {
		c, d, b := r.hooks[hookIndex].handle(ctx)
		if c > 0 {
			code = c
		}
		if d != nil {
			data = d
		}
		if !b {
			return code, data, false
		}
	}
	return code, data, true
}

// Before set the pre hook for router, Before will run before handlers
func (r *router) BeforeRequest(hooks ...interface{}) *router { return r.setHook(0, hooks...) }

// After set the post hook for router, After will run after handlers
func (r *router) AfterRequest(hooks ...interface{}) *router { return r.setHook(1, hooks...) }

func (r *router) Get(uri string, handler ...interface{}) *router {
	return r.Method("GET", uri, handler...)
}
func (r *router) Post(uri string, handler ...interface{}) *router {
	return r.Method("POST", uri, handler...)
}
func (r *router) Put(uri string, handler ...interface{}) *router {
	return r.Method("PUT", uri, handler...)
}
func (r *router) Patch(uri string, handler ...interface{}) *router {
	return r.Method("PATCH", uri, handler...)
}
func (r *router) Delete(uri string, handler ...interface{}) *router {
	return r.Method("DELETE", uri, handler...)
}
func (r *router) Head(uri string, handler ...interface{}) *router {
	return r.Method("HEAD", uri, handler...)
}
func (r *router) Options(uri string, handler ...interface{}) *router {
	return r.Method("OPTIONS", uri, handler...)
}
func (r *router) Any(uri string, handler ...interface{}) *router {
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
		r.Method(method, uri, handler...)
	}
	return r
}

// Group create a router group with the uri prefix
func (r *router) Group(prefix string) *router {
	return &router{
		parent: r,
		basic:  r.basic,
		prefix: path.Join(r.prefix, prefix),
		hooks:  r.hooks,
	}
}

// Method is a raw function route for handler, the method can be 'GET', 'POST'...
func (r *router) Method(method, uri string, handler ...interface{}) *router {
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
