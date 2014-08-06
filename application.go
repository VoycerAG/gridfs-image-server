package main

import (
	"log"
	"os"

	"github.com/VoycerAG/gridfs-image-server/server"
)

// main starts the server and returns an invalid result as exit code
func main() {
	exitCode := server.Deliver()
	log.Printf("Exiting with status code %d", exitCode)
	os.Exit(exitCode)
}
