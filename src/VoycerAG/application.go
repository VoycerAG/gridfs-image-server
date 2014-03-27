package main

import (
	"VoycerAG/server"
	"os"
)

func main() {
	exitCode := server.Deliver()
	os.Exit(exitCode)
}
