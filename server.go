package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"

	_ "modernc.org/sqlite"
)

type Server struct {
	Mux      *http.ServeMux
	Paths    map[string]func(http.ResponseWriter, *http.Request)
	Listener net.Listener
	Logger   *log.Logger
	DB       *sql.DB
	dbPath   string
}

func New(options ...Option) (*Server, error) {
	var err error
	s := Server{}
	s.Mux = http.NewServeMux()

	// loopback only: this server holds personal data and must not be
	// reachable from other machines on the network
	s.Listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return &s, err
	}

	s.ApplyOptions(options...)

	// create or open db
	s.DB, err = sql.Open("sqlite", s.dbPath)
	if err != nil {
		s.Listener.Close()
		return &s, err
	}

	// sql.Open does not connect; Ping makes a bad path fail here
	if err = s.DB.Ping(); err != nil {
		s.DB.Close()
		s.Listener.Close()
		return &s, err
	}

	if _, err = s.DB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		s.DB.Close()
		s.Listener.Close()
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
			function,
		)
	}

	return http.Serve(
		s.Listener,
		s.Mux,
	)
}
