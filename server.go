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
	s.Recovery(Recovery) // 使用默认的 recovery
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
