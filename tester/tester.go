package tester

import (
	"bufio"
	"sync"
	"testing"

	"github.com/cloudfly/mowa"
	"github.com/k0kubun/pp"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// Tester 是用来测试 mowa api 的工具
type Tester struct {
	listener *fasthttputil.InmemoryListener
	once     sync.Once
	api      *mowa.Mowa
}

// New create a tester
func New(api *mowa.Mowa) *Tester {
	tester := &Tester{
		api:      api,
		listener: fasthttputil.NewInmemoryListener(),
	}
	go api.RunWithListener(tester.listener)
	return tester
}

// Test handle the request and return the response, it's no need to call Run() before Test request
func (tester *Tester) Test(request *fasthttp.Request) (*fasthttp.Response, error) {
	conn, err := tester.listener.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	bw := bufio.NewWriter(conn)
	if err := request.Write(bw); err != nil {
		pp.Println(err)
		return nil, err
	}
	bw.Flush()
	resp := fasthttp.AcquireResponse()
	br := bufio.NewReader(conn)
	if err := resp.Read(br); err != nil {
		return nil, err
	}
	return resp, nil
}

// MustTest handle the request and return the response, it's no need to call Run() before Test request
func (tester *Tester) MustTest(request *fasthttp.Request) *fasthttp.Response {
	resp, err := tester.Test(request)
	if err != nil {
		panic(err)
	}
	return resp
}

// Get send a gt request
func (tester *Tester) Get(path string) (*fasthttp.Response, error) {
	return tester.method("GET", path, nil, nil)
}

// Post send a gt request
func (tester *Tester) Post(path string, args *fasthttp.Args, body []byte) (*fasthttp.Response, error) {
	return tester.method("POST", path, args, body)
}

// Put send a gt request
func (tester *Tester) Put(path string, args *fasthttp.Args, body []byte) (*fasthttp.Response, error) {
	return tester.method("PUT", path, args, body)
}

// Patch send a gt request
func (tester *Tester) Patch(path string, args *fasthttp.Args, body []byte) (*fasthttp.Response, error) {
	return tester.method("PATCH", path, args, body)
}

// Delete send a gt request
func (tester *Tester) Delete(path string) (*fasthttp.Response, error) {
	return tester.method("DELETE", path, nil, nil)
}

func (tester *Tester) method(method, path string, args *fasthttp.Args, body []byte) (*fasthttp.Response, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.SetMethod(method)
	if args == nil {
		args.WriteTo(req.BodyWriter())
	}
	if len(body) > 0 {
		req.SetBody(body)
	}
	req.SetRequestURI("http://localhost" + path)
	return tester.Test(req)
}

// CheckResponseBody 检测 body 值
func CheckResponseBody(t *testing.T, resp *fasthttp.Response, body string) {
	assert.Equal(t, body, string(resp.Body()))
}

// CheckResponseStatus 检测 status 值
func CheckResponseStatus(t *testing.T, resp *fasthttp.Response, status int) {
	assert.Equal(t, status, resp.StatusCode())
}

// CheckResponseHeader 检测 header 值
func CheckResponseHeader(t *testing.T, resp *fasthttp.Response, key, value string) {
	assert.Equal(t, value, resp.Header.Peek(key))
}
