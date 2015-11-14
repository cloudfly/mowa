package mowa

import (
	"net/http"
)

type Myapi struct {
	*Router
	Addr   string
	server *http.Server
}

func New() *Myapi {
	s := &Myapi{
		Router: NewRouter(),
		server: new(http.Server),
	}
	s.server.Handler = s
	return s
}

func Default() *Myapi {
	pre := []Handler{}
	api := New()
	api.Router = NewRouter(pre)
	return api
}

func (api *Myapi) Run(addr string) error {
	api.server.Addr = addr
	return api.server.ListenAndServe()
}
