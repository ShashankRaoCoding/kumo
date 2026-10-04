package main

import (
	"log" 
	"github.com/ShashankRaoCoding/kumo/structs/server"
)

func main() {
	s, err := server.New()
	if err != nil {
		log.Println(err.Err()) 
		return 
	}
	log.Println(s.Listener.Addr()) 
}










































