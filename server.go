package mowa

import (
	"net"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

// Mowa represent a http server
type Mowa struct {
	*router
	sync.RWMutex
	server   *fasthttp.Server
	listener net.Listener
}

// New create a new http server
func New(options ...Option) *Mowa {
	router := newRouter()
	s := &Mowa{
		router: router,
		server: &fasthttp.Server{
			Handler: router.Handler,
		},
	}
	for _, op := range options {
		op(s)
	}
	return s
}

// Run will start the web service and listening the tcp addr
func (api *Mowa) Run(addr string) error {
	api.Lock() // lock the api in case of calling Shutdown() before Serve()
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		api.Unlock()
		return err
	}

	listener, err := net.ListenTCP("tcp4", tcpAddr)
	if err != nil {
		api.Unlock()
		return err
	}
	api.listener = listener
	api.Unlock()

	return api.server.Serve(api.listener)
}

// RunWithListener serve the http service using the given listener
func (api *Mowa) RunWithListener(listener net.Listener) error {
	api.Lock()
	api.listener = listener
	api.Unlock()
	return api.server.Serve(api.listener)
}

// Shutdown the server gracefully
func (api *Mowa) Shutdown() error {
	return api.server.Shutdown()
}

// Listener return the net.TCPListener http service serve on
func (api *Mowa) Listener() net.Listener {
	api.RLock()
	defer api.RUnlock()
	return api.listener
}

// Option represents the server's configuration setting
type Option func(s *Mowa)

// WithReadTimeout set the timeout duration for reading body from request
func WithReadTimeout(timeout time.Duration) Option {
	return func(mowa *Mowa) {
		mowa.server.ReadTimeout = timeout
	}
}

// WithWriteTimeout set the timeout duration for writing body to response
func WithWriteTimeout(timeout time.Duration) Option {
	return func(mowa *Mowa) {
		mowa.server.WriteTimeout = timeout
	}
}

// WithPanicHandler setting the recovery function, it will be called when handler panic
func WithPanicHandler(f func(ctx *fasthttp.RequestCtx, err interface{})) Option {
	return func(mowa *Mowa) {
		mowa.router.basic.PanicHandler = f
	}
}

// WithConcurrency set the maximum number of concurrent connections the server may serve.
func WithConcurrency(n int) Option {
	return func(mowa *Mowa) {
		mowa.server.Concurrency = n
	}
}

// WithLogger set the logger
func WithLogger(logger fasthttp.Logger) Option {
	return func(mowa *Mowa) {
		mowa.server.Logger = logger
	}
}

// WithMaxConnsPerIP set maximum number of concurrent client connections allowed per IP.
// by default no limit
func WithMaxConnsPerIP(n int) Option {
	return func(mowa *Mowa) {
		mowa.server.MaxConnsPerIP = n
	}
}

// WithKeepalive will disable keepalive tcp connection, server will close the tcp connection after sending response
// it's enabled by default
func WithKeepalive(enable bool) Option {
	return func(mowa *Mowa) {
		mowa.server.DisableKeepalive = !enable
	}
}

// WithNotFoundHandler set the not found handler
func WithNotFoundHandler(f func(ctx *fasthttp.RequestCtx)) Option {
	return func(mowa *Mowa) {
		mowa.router.basic.NotFound = f
	}
}
