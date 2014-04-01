package main

import (
	"github.com/VoycerAG/server"
	"log"
	"os"
)

// main starts the server and returns an invalid result as exit code
func main() {
	exitCode := server.Deliver()
	log.Printf("Exiting with status code %d", exitCode)
	os.Exit(exitCode)
}
