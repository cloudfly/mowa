package mowa

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

var (
	testC *Context
)

func init() {
	testC = &Context{
		RequestCtx: newRequest("GET", "http://localhost:1234/hello/world?name=chen&age=25&name=yun"),
	}
}

func newRequest(method, url string) *fasthttp.RequestCtx {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(url)
	req.Header.SetMethod(method)
	return &fasthttp.RequestCtx{
		Request: *req,
	}
}

func TestServer(t *testing.T) {
	api := New(WithReadTimeout(time.Second))
	go api.Run(":10000")
	api.Get("/test", func(c *Context) (int, interface{}) {
		return 200, "test"
	})
	defer api.Shutdown()
	resp, err := http.Get("http://localhost:10000/test")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 200, resp.StatusCode)
	content, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, `test`, string(content))
}

func TestServeHTTP(t *testing.T) {
	handler := func(c *Context) (int, interface{}) {
		return 200, c.QueryArgs().Peek("return")
	}

	router := newRouter()
	router.Group("/api/v1").Get("/chen", handler)
	router.Get("/yun", handler)
	router.Get("/fei/:age", handler)

	req := newRequest("GET", "http://localhost/chen")
	router.Handler(req)
	assert.Equal(t, 404, req.Response.StatusCode())

	req = newRequest("GET", "http://localhost/api/v1/chen?return=hello")
	router.Handler(req)
	assert.Equal(t, 200, req.Response.StatusCode())
	assert.Equal(t, `hello`, string(req.Response.Body()))

	req = newRequest("GET", "http://localhost/yun?return=yun")
	router.Handler(req)
	assert.Equal(t, 200, req.Response.StatusCode())
	assert.Equal(t, `yun`, string(req.Response.Body()))

	req = newRequest("GET", "http://localhost/fei/23?return=23")
	router.Handler(req)
	assert.Equal(t, `23`, string(req.Response.Body()))
}

func TestHook(t *testing.T) {
	num := 0
	router := newRouter()
	router.BeforeRequest(func(ctx *Context) {
		println("before request")
		num++
	})
	router.Get("/test", func(ctx *Context) {
		println("in request")
		num++
	})
	router.BeforeRequest(func(ctx *Context) {
		println("before request(2)")
		num++
	})
	router.AfterRequest(func(ctx *Context) {
		println("after request")
		num++
	})

	req := newRequest("GET", "http://localhost/test")
	router.Handler(req)
	assert.Equal(t, 4, num)
}

func BenchmarkServeHTTPString(b *testing.B) {
	api := New()
	api.Get("/string", func(c *Context) (int, interface{}) {
		return 200, "test"
	})
	req := newRequest("GET", "http://localhost/string")
	for i := 0; i < b.N; i++ {
		api.Handler(req)
	}
}

func BenchmarkServeHTTPBytes(b *testing.B) {
	api := New()
	api.Get("/bytes", func(c *Context) (int, interface{}) {
		return 200, "test"
	})
	req := newRequest("GET", "http://localhost/bytes")
	for i := 0; i < b.N; i++ {
		api.Handler(req)
	}
}

func BenchmarkServeHTTPJSON(b *testing.B) {
	api := New()
	api.Get("/json", func(c *Context) (int, interface{}) {
		return 200, []int{1, 2, 34, 2, 1}
	})
	req := newRequest("GET", "http://localhost/json")
	for i := 0; i < b.N; i++ {
		api.Handler(req)
	}
}
