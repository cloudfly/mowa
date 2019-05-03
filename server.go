package mowa

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/http2"
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
	return s
}

func (api *Mowa) run(addr string, certFile, keyFile string, h2 bool) error {
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

	if h2 {
		if err := http2.ConfigureServer(api.server, nil); err != nil {
			return err
		}
	}

	if certFile != "" && keyFile != "" {
		return api.server.ServeTLS(api.listener, certFile, keyFile)
	}
	return api.server.Serve(api.listener)
}

// Run the server, and listen to given addr
func (api *Mowa) Run(addr string) error {
	return api.run(addr, "", "", false)
}

// RunTLS run tls server, and listen to given addr
func (api *Mowa) RunTLS(addr, certFile, keyFile string) error {
	return api.run(addr, certFile, keyFile, false)
}

// RunWithListener serve the http service using the given listener
func (api *Mowa) runWithListener(listener net.Listener, certFile, keyFile string, h2 bool) error {
	api.Lock()
	api.listener = listener
	api.Unlock()

	if h2 {
		if err := http2.ConfigureServer(api.server, nil); err != nil {
			return err
		}
	}

	if certFile != "" && keyFile != "" {
		return api.server.ServeTLS(api.listener, certFile, keyFile)
	}
	return api.server.Serve(api.listener)
}

// RunWithListener serve the http service using the given listener
func (api *Mowa) RunWithListener(listener net.Listener) error {
	return api.runWithListener(listener, "", "", false)
}

// RunTLSWithListener serve the https service using the given listener
func (api *Mowa) RunTLSWithListener(listener net.Listener, certFile, keyFile string) error {
	return api.runWithListener(listener, certFile, keyFile, false)
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
