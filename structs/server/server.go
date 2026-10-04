package server

import (
	"net/http" 
	"net"
	"log"
)

type Server struct {
	Mux *http.ServeMux 
	Paths map[string]func(w http.Writer, r *http.Request) 
	Listener net.Listener
}












































