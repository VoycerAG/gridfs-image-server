package server

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"labix.org/v2/mgo"
	"log"
	"net/http"
	"time"
)

var Connection *mgo.Session

// just a static welcome handler
func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html>")
	fmt.Fprintf(w, "<h1>Image Server.</h1>")
	fmt.Fprintf(w, "</html>")
}

//
func imageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	database := vars["database"]

	if database == "" {
		log.Fatal("database must not be empty")
		return
	}

	port := vars["port"]

	if port == "" {
		port = "27017"
	}

	filename := vars["image"]

	if filename == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	gridfs := Connection.DB(database).GridFS("fs")
	fp, err := gridfs.Open(filename)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	} else {
		// if the file pointer is not set, fp.Close() throws an error.. FIX this upstream?
		defer fp.Close()
	}

	md5 := fp.MD5()

	modifiedHeader := r.Header.Get("If-Modified-Since")
	modifiedTime := time.Now()
	cachingEnabled := r.Header.Get("Cache-Control") != "no-cache"

	if modifiedHeader != "" {
		modifiedHeader = ""
	} else {
		modifiedTime, _ = time.Parse(time.RFC1123, modifiedHeader)
	}

	if hasCached(md5, r.Header.Get("If-None-Match"), modifiedTime, fp.UploadDate()) && cachingEnabled {
		w.WriteHeader(http.StatusNotModified)
		fmt.Printf("[DEBUG][304] Returning cached image for %s\n", md5)
		return
	}

	w.Header().Set("Etag", md5)
	w.Header().Set("Cache-Control", "max-age=315360000")
	d, _ := time.ParseDuration("315360000s")

	expires := fp.UploadDate().Add(d)

	w.Header().Set("Last-Modified", fp.UploadDate().Format(time.RFC1123))
	w.Header().Set("Expires", expires.Format(time.RFC1123))
	w.Header().Set("Date", fp.UploadDate().Format(time.RFC1123))

	fmt.Printf("[DEBUG][200] Returning raw image for %s\n", md5)

	_, err = io.Copy(w, fp)

	HandleError(err)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("[ERROR][500] Bad Request for %s\n", md5)
		return
	}
}

//
func Deliver() int {
	err := errors.New("")

	serverPort := flag.Int("port", 8000, "the server port where we will serve images")
	host := flag.String("host", "localhost", "the database host")

	flag.Parse()

	fmt.Printf("Server started. Listening on %d database host is %s\n", *serverPort, *host)

	// in order to simple configure the image server in the proxy configuration of nginx
	// we will be getting every database variable from the request
	serverRoute := "/{database}/{port}/{image}"

	Connection, err = mgo.Dial(*host)
	Connection.SetMode(mgo.Eventual, true)
	Connection.SetSyncTimeout(0)

	if err != nil {
		log.Fatal("Cannot connect to database")
		return -1
	}

	r := mux.NewRouter()
	r.HandleFunc("/", welcomeHandler)
	r.HandleFunc(serverRoute, imageHandler)

	http.Handle("/", r)

	err = http.ListenAndServe(fmt.Sprintf(":%d", *serverPort), nil)

	if err != nil {
		log.Fatal(err)
		return -1
	}

	return 0
}
