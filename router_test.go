package mowa

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestServeHTTP(t *testing.T) {
	router := NewRouter()
	router.Group("/api/v1").Get("/chen", handle1)
	router.Get("/yun", handle2)
	router.Get("/fei/:age", handle3)

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
