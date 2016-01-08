// +build !facedetection

package main

import (
	"flag"
	"log"
)

// main starts the server and returns an invalid result as exit code
func main() {
	flag.Parse()

	if *configurationFilepath == "" {
		log.Fatal("configuration must be given")
		return
	}

	run(*host, *configurationFilepath, *newrelicKey, *serverPort)
}
