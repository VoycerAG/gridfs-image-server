package server

import (
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"html"
	"io"
	"io/ioutil"
	"labix.org/v2/mgo"
	"net/http"
	"os"
	"time"
)

var ServerConfiguration Config

type Config struct {
	Database   string
	Host       string
	Port       int
	ListenPort int
	Route      string
}

func HandleError(err error) {
	if err != nil {
		fmt.Printf("[ERROR] Error occured: %s\n", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html>")
	fmt.Fprintf(w, "<h1>Image Server.</h1>")
	fmt.Fprintf(w, "</html>")
}

func hasCached(etag string, md5 string, modifiedTime time.Time, updateTime time.Time) bool {
	if updateTime.After(modifiedTime) || md5 != etag {
		return false
	}

	return true
}

func ReadConfiguration() (Config, error) {
	filename := "configuration.xml"
	xmlFile, err := os.Open(filename)
	if err != nil {
		return Config{}, errors.New(fmt.Sprintf("[ERROR] configuration file could not be found [%s]", filename))
	}
	defer xmlFile.Close()

	b, _ := ioutil.ReadAll(xmlFile)

	var q Config
	xml.Unmarshal(b, &q)
	return q, nil
}

func mongoHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[DEBUG] Access on %s\n", html.EscapeString(r.URL.Path))

	host := ServerConfiguration.Host
	database := ServerConfiguration.Database
	session, err := mgo.Dial(host)
	HandleError(err)
	session.SetMode(mgo.Monotonic, true)

	vars := mux.Vars(r)

	filename := vars["image"]

	if filename == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	gridfs := session.DB(database).GridFS("fs")
	fp, err := gridfs.Open(filename)

	HandleError(err)

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

func Deliver() {
	var err error
	ServerConfiguration, err = ReadConfiguration()

	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Printf("[INFO] Database [%s] Host [%s] Port [%d]\n", ServerConfiguration.Database, ServerConfiguration.Host, ServerConfiguration.Port)
	fmt.Printf("[INFO] Serving on Route %s\n", ServerConfiguration.Route)

	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.HandleFunc(ServerConfiguration.Route, mongoHandler)
	http.Handle("/", r)
	fmt.Printf("[INFO] Image server started on Port %d\n", ServerConfiguration.ListenPort)
	http.ListenAndServe(fmt.Sprintf(":%d", ServerConfiguration.ListenPort), nil)
}
