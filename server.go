package mowa

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
	"net/http"
	"path"
	"reflect"
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
	return api.server.ListenAndServe()
}

/****************** Router *********************/
type Handler interface{}

func HttpRouterHandle(handlers ...reflect.Value) httprouter.Handle {
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
				c.Writer.WriteHeader(500)
				b, _ := json.Marshal(NewError(500, "handler panic", r.(error).Error()))
				c.Writer.Write(b)
			}
		}()

		c.Request.ParseForm()

		// run handler
		for _, handler := range handlers {
			ret := handler.Call([]reflect.Value{reflect.ValueOf(c)})
			switch len(ret) {
			case 1:
				c.Code, c.Data = 200, ret[0].Interface()
			case 2:
				c.Code, c.Data = int(ret[0].Int()), ret[1].Interface()
			case 3:
				c.Code, c.Data, b = int(ret[0].Int()), ret[1].Interface(), ret[2].Bool()
				if b {
					break
				}
			}
		}

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

// request for /people/:name:string/:age:int will changed to:
// httprouter uri is: `/people/:name/:age`
// `:string` and `:int` setting will be stored into paramRules
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

func (r *Router) Get(uri string, handler ...Handler)     { r.Method("GET", uri, handler...) }
func (r *Router) Post(uri string, handler ...Handler)    { r.Method("POST", uri, handler...) }
func (r *Router) Put(uri string, handler ...Handler)     { r.Method("PUT", uri, handler...) }
func (r *Router) Patch(uri string, handler ...Handler)   { r.Method("PATCH", uri, handler...) }
func (r *Router) Delete(uri string, handler ...Handler)  { r.Method("DELETE", uri, handler...) }
func (r *Router) Head(uri string, handler ...Handler)    { r.Method("HEAD", uri, handler...) }
func (r *Router) Options(uri string, handler ...Handler) { r.Method("OPTIONS", uri, handler...) }

type notFoundHandler struct{}

func (h *notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	content, _ := json.Marshal(map[string]string{
		"code": "404",
		"msg":  "page not found",
	})
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
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Cause string `json:"cause"`
}

func NewError(code int, format string, v ...interface{}) error {
	return &Error{
		Code: code,
		Msg:  fmt.Sprintf(format, v...),
	}
}

func (err *Error) Error() string {
	return err.Msg
}
