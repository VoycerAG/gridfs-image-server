package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"gopkg.in/mgo.v2"

	"github.com/VoycerAG/gridfs-image-server/cmd/advanced/resizer"
	"github.com/VoycerAG/gridfs-image-server/server"
	"github.com/VoycerAG/gridfs-image-server/server/paint"
)

//init will automatically register the smartcrop resizer
func init() {
	paint.AddResizer(resizer.TypeSmartcrop, resizer.NewSmartcrop())
}

// main starts the server and returns an invalid result as exit code
func main() {
	configurationFilepath := flag.String("config", "configuration.json", "path to the configuration file")
	serverPort := flag.Int("port", 8000, "the server port where we will serve images")
	host := flag.String("host", "localhost:27017", "the database host with an optional port, localhost would suffice")
	newrelicKey := flag.String("license", "", "your newrelic license key in order to enable monitoring")

	flag.Parse()

	if *configurationFilepath == "" {
		log.Fatal("configuration must be given")
		return
	}

	config, err := server.NewConfigFromFile(*configurationFilepath)
	if err != nil {
		log.Fatal(err)
		return
	}

	session, err := mgo.Dial(*host)
	if err != nil {
		log.Fatal(err)
		return
	}

	session.SetSyncTimeout(0)
	session.SetMode(mgo.Eventual, true)

	storage, err := server.NewGridfsStorage(session)
	if err != nil {
		log.Fatal(err)
		return
	}

	imageServer := server.NewImageServerWithNewRelic(config, storage, *newrelicKey)

	handler := imageServer.Handler()

	log.Printf("Server started. Listening on %d database host is %s\n", *serverPort, *host)

	err = http.ListenAndServe(fmt.Sprintf(":%d", *serverPort), handler)
	if err != nil {
		log.Fatal(err)
	}
}
