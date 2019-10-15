package mowa

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// Mowa represent a http server
type Mowa struct {
	Router // the router of server
	sync.Mutex
	server   *http.Server
	ctx      context.Context
	listener net.Listener
}

// New create a new http server
func New(ctx context.Context) *Mowa {
	if ctx == nil {
		ctx = context.Background()
	}
	s := &Mowa{
		Router: newRouter(ctx),
		server: new(http.Server),
		ctx:    ctx,
	}
	s.server.Handler = s
	s.Recovery(Recovery) // 使用默认的 recovery
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
func (api *Mowa) Shutdown(timeout time.Duration) error {
	api.Lock()
	defer api.Unlock()
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return api.server.Shutdown(c)
}

// Listener return the net.TCPListener http service serve on
func (api *Mowa) Listener() net.Listener {
	api.Lock()
	defer api.Unlock()
	return api.listener
}
