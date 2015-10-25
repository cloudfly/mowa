package myapi

import (
	"net/http"
	"net/http/httputil"
	"testing"
)

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
