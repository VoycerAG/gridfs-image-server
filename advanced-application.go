// +build facedetection

package main

import (
	"flag"
	"log"

	"github.com/VoycerAG/gridfs-image-server/server/paint"
	"github.com/VoycerAG/gridfs-image-server/server/resizer"
)

// main starts the server and returns an invalid result as exit code
func main() {
	haarcascade := flag.String("haarcascade", "", "haarcascade file path")
	flag.Parse()

	if *configurationFilepath == "" {
		log.Fatal("configuration must be given")
		return
	}
	if *haarcascade == "" {
		log.Fatal("haarcascade file must be set")
		return
	}

	smartcrop := resizer.NewSmartcrop(*haarcascade, paint.CropResizer{})
	paint.AddResizer(resizer.TypeSmartcrop, smartcrop)

	run(*host, *configurationFilepath, *newrelicKey, *serverPort)
}
