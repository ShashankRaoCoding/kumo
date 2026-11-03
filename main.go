package main

import (
	"github.com/ShashankRaoCoding/kumo/server"
	"log"
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
