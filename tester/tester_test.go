package tester

import (
	"testing"

	"github.com/cloudfly/mowa"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestTester(t *testing.T) {
	api := mowa.New()
	api.Get("/api/hello", func(ctx *fasthttp.RequestCtx) interface{} { return "hello" })
	tester := New(api)

	req := fasthttp.AcquireRequest()
	req.SetRequestURI("http://localhost/api/hello")
	resp, err := tester.Test(req)
	assert.NoError(t, err)
	CheckResponseBody(t, resp, "hello")

	req.SetRequestURI("http://localhost/abc")
	resp, err = tester.Test(req)
	assert.NoError(t, err)
	t.Log(string(resp.Body()))
	CheckResponseBody(t, resp, `{"code":404,"error":"page not found","data":null}`)
}
