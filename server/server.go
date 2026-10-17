package server

import (
	"net/http" 
	"net"
	"log"
	"database/sql"
	// "fmt" 
)

type Server struct {
	Mux *http.ServeMux 
	Paths map[string]func(http.ResponseWriter, *http.Request) 
	Listener net.Listener
	Logger *log.Logger 
	DB *sql.DB
	Config config
}

type config struct {
	DBPath string 
}

func New() (*Server, error) {
	var err error 
	s := Server{} 
	s.Mux = http.NewServeMux() 

	s.Listener, err = net.Listen("tcp", ":0") 
	if err != nil {
		return &s, err
	}

	return &s, nil 
}

func (s *Server) Start() error {
	for path, function := range s.Paths {
		s.Mux.HandleFunc(
			path,
			function , 
		)
	}

	return http.Serve(
		s.Listener, 
		s.Mux, 
	)
}








































