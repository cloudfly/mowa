package mowa

import (
	"net/http"
)

// represent a http server
type Mowa struct {
	// the router of server
	*Router
	// the address to listen on
	Addr   string
	server *http.Server
}

// Create a new http server
func New() *Mowa {
	s := &Mowa{
		Router: NewRouter(),
		server: new(http.Server),
	}
	s.server.Handler = s
	return s
}

// Run the server, and listen to given addr
func (api *Mowa) Run(addr string) error {
	api.server.Addr = addr
	return api.server.ListenAndServe()
}
