package option

import "github.com/ShashankRaoCoding/kumo/server" 

func WithDBPath(path string) Option {
	return func(s *Server) {
		s.dbPath = path
	}
}







































