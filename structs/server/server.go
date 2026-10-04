package server

import (
	"net/http" 
	"net"
	"log"
)

type Server struct {
	Mux *http.ServeMux 
	Listener net.Listener
}












































