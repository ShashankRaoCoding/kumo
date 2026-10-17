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
}

func New(options ...Option) (*Server, error) {
	var err error 
	s := Server{} 
	s.Mux = http.NewServeMux() 

	s.Listener, err = net.Listen("tcp", ":0") 
	if err != nil {
		return &s, err
	}

	s.ApplyOptions(options...)

	// create or open db 
	s.DB, err = sql.Open("sqlite3", s.dbPath)
	if err != nil {
		return &s, err
	}

	return &s, nil 
}

func (s *Server) ApplyOptions(options ...Option) {
	for _, option := range options {
		option(s) 
	}
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








































