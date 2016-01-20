package mowa

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"path"
	"strings"
)

/************ API Server **************/
type Myapi struct {
	*Router
	Addr   string
	server *http.Server
}

func New() *Myapi {
	s := &Myapi{
		Router: NewRouter(),
		server: new(http.Server),
	}
	s.server.Handler = s
	return s
}

func (api *Myapi) Run(addr string) error {
	api.server.Addr = addr
	return api.server.ListenAndServe()
}

/****************** Router *********************/
type Handler func(*Context) (int, interface{})

func HttpRouterHandle(paramRules map[string][]string, handlers ...Handler) httprouter.Handle {
	var f httprouter.Handle = func(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		var (
			c *Context = &Context{
				Request:    req,
				Writer:     rw,
				Return:     false,
				Params:     ps,
				ParamRules: paramRules,
			}
		)
		c.Request.ParseForm()
		// run handler
		for _, handler := range handlers {
			c.code, c.data = handler(c)
			if c.Return {
				break
			}
		}
		c.JSON(c.code, c.data)
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

func (r *Router) Group(prefix string, hooks ...[]Handler) *Router {
	gr := &Router{
		basic:  r.basic,
		prefix: path.Join(r.prefix, prefix),
	}
	// combine parent hooks and given hooks
	for i := 0; i < 2; i++ {
		if i < len(hooks) && hooks[i] != nil {
			gr.hooks[i] = append(r.hooks[i], hooks[i]...)
		} else {
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
	paramRules := make(map[string][]string)
	fields := strings.Split(uri, "/")
	fieldsClean := make([]string, len(fields), len(fields))
	for i, field := range fields {
		if len(field) > 0 && field[0] == ':' {
			words := strings.Split(field, ":")
			fieldsClean[i] = ":" + words[1]
			paramRules[words[1]] = words[2:]
		} else {
			fieldsClean[i] = field
		}
	}
	realURI := path.Join(r.prefix, path.Join(fieldsClean...))
	handler = append(append(r.hooks[0], handler...), r.hooks[1]...)
	r.basic.Handle(method, realURI, HttpRouterHandle(paramRules, handler...))
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

/*********** Context *****************/
type Context struct {
	Request    *http.Request
	Writer     http.ResponseWriter
	code       int
	data       interface{}
	Return     bool
	Params     httprouter.Params
	ParamRules map[string][]string
}

func (c *Context) JSON(code int, data interface{}) {
	c.code = code
	c.data = data
ENCODE:
	content, err := json.Marshal(data)
	if err != nil {
		c.code = 500
		c.data = NewError(500, "unvalid return data")
		goto ENCODE
	}
	c.Writer.WriteHeader(c.code)
	c.Writer.Write(content)
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
