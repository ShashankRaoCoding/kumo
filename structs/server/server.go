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
	var err error 
	s := Server{} 
	s.Mux = http.NewServeMux() 
	s.Listener, err = net.Listen("tcp", ":0") 

	return &s, nil 
}










































