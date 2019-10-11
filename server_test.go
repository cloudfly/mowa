package mowa

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	testC *Context
)

func init() {
	req, _ := http.NewRequest("Get", "http://localhost:1234/hello/world?name=chen&age=25&name=yun", nil)
	testC = &Context{
		Request: req,
	}
}

func TestServer(t *testing.T) {
	api := New(context.Background())
	go api.Run(":10000")
	api.Get("/test", func(c *Context) (int, interface{}) {
		return 200, "test"
	})
	defer api.Shutdown(time.Second)
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
		return 200, c.Query("return", "")
	}

	router := newRouter(context.Background())
	router.Group("/api/v1").Get("/chen", handler)
	router.Get("/yun", handler)
	router.Get("/fei/:age", handler)

	req, err := http.NewRequest("GET", "http://localhost/chen", nil)
	if err != nil {
		t.Error(err)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)

	req, err = http.NewRequest("GET", "http://localhost/api/v1/chen?return=hello", nil)
	if err != nil {
		t.Error(err)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `hello`, w.Body.String())

	req, err = http.NewRequest("GET", "http://localhost/yun?return=yun", nil)
	if err != nil {
		t.Error(err)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `yun`, w.Body.String())

	req, err = http.NewRequest("GET", "http://localhost/fei/23?return=23", nil)
	if err != nil {
		t.Error(err)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, `23`, w.Body.String())
}

func TestHook(t *testing.T) {
	num := 0
	router := newRouter(context.Background())
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

	req, err := http.NewRequest("GET", "http://localhost/test", nil)
	if err != nil {
		t.Error(err)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 4, num)
}

func BenchmarkServeHTTPString(b *testing.B) {
	api := New(context.Background())
	api.Get("/string", func(c *Context) (int, interface{}) {
		return 200, "test"
	})
	req, err := http.NewRequest("GET", "http://localhost/string", nil)
	if err != nil {
		b.Fatal(err)
	}
	rw := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		api.ServeHTTP(rw, req)
	}
}

func BenchmarkServeHTTPBytes(b *testing.B) {
	api := New(context.Background())
	resp := []byte("test")
	api.Get("/bytes", func(c *Context) (int, interface{}) {
		return 200, resp
	})
	req, err := http.NewRequest("GET", "http://localhost/bytes", nil)
	if err != nil {
		b.Fatal(err)
	}
	rw := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		api.ServeHTTP(rw, req)
	}
}

func BenchmarkServeHTTPJSON(b *testing.B) {
	api := New(context.Background())
	api.Get("/json", func(c *Context) (int, interface{}) {
		return 200, []int{1, 2, 34, 2, 1}
	})
	req, err := http.NewRequest("GET", "http://localhost/json", nil)
	if err != nil {
		b.Fatal(err)
	}
	rw := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		api.ServeHTTP(rw, req)
	}
}
