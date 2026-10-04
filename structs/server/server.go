package server

import (
	"net/http" 
	"net"
	"log"
	"fmt" 
)

type Server struct {
	Mux *http.ServeMux 
	Paths map[string]func(http.Writer, *http.Request) 
	Listener net.Listener
	Logger *log.Logger 
}

func New() (*Server, error) {
	s := Server{} 
	s.Mux = http.NewServeMux() 
	s.Listener, err = net.Listen("tcp", "") 

	return &s, nil 
}










































