package mowa

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
)

var testC *Context

func init() {
	req, _ := http.NewRequest("Get", "http://localhost:1234/hello/world?name=chen&age=25&name=yun", nil)
	testC = &Context{
		Request: req,
	}
}

func handle1(c *Context) (int, interface{}) {
	return 200, "handle1: hello world"
}

func handle2(c *Context) (int, interface{}) {
	return 203, map[string]interface{}{
		"one": 1,
		"two": "two",
	}
}

func handle3(c *Context) (int, interface{}) {

	return 202, map[string]interface{}{
		"one": 1,
		"age": c.Int("age", 20),
	}
}

func TestServer(t *testing.T) {
	api := Default()
	go api.Run(":10000")
	api.Get("/test", func(c *Context) (int, interface{}) {
		return 200, "test"
	})
	resp, err := http.Get("http://localhost:10000/test")
	if err != nil {
		t.Error(err)
	}
	content, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Error(err)
	}
	t.Log(string(content))

}

func TestServeHTTP(t *testing.T) {
	router := NewRouter([]Handler{ParamChecker})
	router.Group("/api/v1").Get("/chen", handle1)
	router.Get("/yun", handle2)
	router.Get("/fei/:age:int", handle3)

	for _, uri := range []string{"/chen", "/api/v1/chen", "/yun", "/fei/aa", "/fei/25"} {

		req, err := http.NewRequest("GET", "http://localhost"+uri, nil)
		if err != nil {
			t.Error(err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		t.Log("\n\nRequest for " + uri)
		t.Log(w.Code)
		t.Log(w.Body.String())
	}

}
