package main

// import "github.com/ShashankRaoCoding/main/server"

func WithDBPath(path string) Option {
	return func(s *Server) {
		s.dbPath = path
	}
}
