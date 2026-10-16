package main

import (
	"log" 
	"github.com/ShashankRaoCoding/kumo/structs/server"
	"github.com/ShashankRaoCoding/kumo/structs/database"
)

func main() {
	s, err := server.New()
	if err != nil {
		log.Println(err.Error()) 
		return 
	}
	log.Println(s.Listener.Addr()) 

	err = s.Start() 
	if err == nil {
		return 
	}
	log.Println(err.Error()) 
}
