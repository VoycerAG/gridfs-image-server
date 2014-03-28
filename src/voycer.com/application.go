package main

import (
	"os"
	"voycer.com/server"
)

func main() {
	exitCode := server.Deliver()
	os.Exit(exitCode)
}
