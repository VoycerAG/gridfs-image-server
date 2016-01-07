package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/VoycerAG/gridfs-image-server/server"
	"gopkg.in/mgo.v2"
)

var (
	configurationFilepath *string
	serverPort            *int
	host                  *string
	newrelicKey           *string
)

func init() {
	configurationFilepath = flag.String("config", "configuration.json", "path to the configuration file")
	serverPort = flag.Int("port", 8000, "the server port where we will serve images")
	host = flag.String("host", "localhost:27017", "the database host with an optional port, localhost would suffice")
	newrelicKey = flag.String("license", "", "your newrelic license key in order to enable monitoring")
}

func run(mongoHost, configFile, newrelicToken string, port int) {
	session, err := mgo.Dial(*host)
	if err != nil {
		log.Fatal(err)
		return
	}

	config, err := server.NewConfigFromFile(configFile)
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

	imageServer := server.NewImageServerWithNewRelic(config, storage, newrelicToken)

	handler := imageServer.Handler()

	log.Printf("Server started. Listening on %d database host is %s\n", port, mongoHost)

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
	if err != nil {
		log.Fatal(err)
	}
}
