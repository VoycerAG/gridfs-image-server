package main

import (
	"github.com/VoycerAG/server"
	"os"
)

func main() {
	exitCode := server.Deliver()
	os.Exit(exitCode)
}
