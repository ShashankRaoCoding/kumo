package server

import (
	"net/http" 
	"net"
)

type Server struct {
	Mux *http.ServeMux 
	Listener net.Listener
}












































