package main.go

import (
	"log" 
	"github.com/ShashankRaoCoding/kumo/structs/server"
)

func main() {
	s, err := server.New()
	log.Println(s.Listener.Addr()) 
}










































