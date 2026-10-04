package server

import (
	"net/http" 
	"net"
	"log"
)

type Server struct {
	Mux *http.ServeMux 
	Paths map[string]func(http.Writer, *http.Request) 
	Listener net.Listener
	Logger *log.Logger 
}












































