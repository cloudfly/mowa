package mowa

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"net/http"
	"path"
	"reflect"
)

// two ways to return code and data, one is to set into context, another is to return (int, interface{})
//type Handler func(*Context) (int, interface{})
type Handler interface{}

func HttpRouterHandle(handlers ...reflect.Value) httprouter.Handle {
	var f httprouter.Handle = func(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		var c *Context = &Context{
			Context: context.TODO(),
			Request: req,
			Writer:  rw,
			Code:    500,
			Data:    "",
			Return:  false,
		}

		c.Context = context.WithValue(c.Context, "params", ps)

		// defer to recover in case of some panic, assert in context use this
		defer func() {
			if r := recover(); r != nil {
				// TODO error struct
				c.JSON(500, map[string]interface{}{
					"code": 500,
					"msg":  r.(error).Error(),
				})
			}
		}()

		c.Request.ParseForm()

		// run handler
		for _, handler := range handlers {
			ret := handler.Call([]reflect.Value{reflect.ValueOf(c)})
			if len(ret) == 2 {
				c.Code, c.Data = int(ret[0].Int()), ret[1].Interface()
			}
			if c.Return {
				goto RETURN
			}
		}

	RETURN:
		if c.Data != nil {
			c.JSON(c.Code, c.Data)
		}
	}

	return f
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

func (r *Router) PreHook(hooks ...Handler) *Router {
	r.hooks[0] = hooks
	return r
}

func (r *Router) PreUse(hooks ...Handler) *Router {
	r.hooks[0] = append(r.hooks[0], hooks...)
	return r
}

func (r *Router) PostHook(hooks ...Handler) *Router {
	r.hooks[1] = hooks
	return r
}

func (r *Router) PostUse(hooks ...Handler) *Router {
	r.hooks[1] = append(r.hooks[1], hooks...)
	return r
}

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

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	r.basic.ServeHTTP(rw, req)
}

func (r *Router) Method(method, uri string, handler ...Handler) {
	handler = append(append(r.hooks[0], handler...), r.hooks[1]...)
	values := make([]reflect.Value, len(handler), len(handler))
	for i, item := range handler {
		if reflect.TypeOf(item).Kind() != reflect.Func {
			panic(fmt.Errorf("Handler must be a function"))
		}
		values[i] = reflect.ValueOf(item)
	}
	r.basic.Handle(method, path.Join(r.prefix, uri), HttpRouterHandle(values...))
}

func (r *Router) Get(uri string, handler ...Handler) {
	r.Method("GET", uri, handler...)
}

func (r *Router) Post(uri string, handler ...Handler) {
	r.Method("POST", uri, handler...)
}

func (r *Router) Put(uri string, handler ...Handler) {
	r.Method("PUT", uri, handler...)
}

func (r *Router) Patch(uri string, handler ...Handler) {
	r.Method("PATCH", uri, handler...)
}

func (r *Router) Delete(uri string, handler ...Handler) {
	r.Method("DELETE", uri, handler...)
}

func (r *Router) Head(uri string, handler ...Handler) {
	r.Method("HEAD", uri, handler...)
}

func (r *Router) Options(uri string, handler ...Handler) {
	r.Method("OPTIONS", uri, handler...)
}
