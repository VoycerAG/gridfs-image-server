package main

import (
	"github.com/VoycerAG/server"
	"log"
	"os"
)

func main() {
	exitCode := server.Deliver()
	log.Printf("Exiting with status code %d", exitCode)
	os.Exit(exitCode)
}
