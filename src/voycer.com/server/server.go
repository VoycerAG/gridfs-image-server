package server

import (
	"code.google.com/p/graphics-go/graphics"
	_ "crypto/sha256"
	_ "encoding/base64"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"strconv"
	"time"
	"voycer.com/config"
)

var Connection *mgo.Session

// just a static welcome handler
func welcomeHandler(w http.ResponseWriter, r *http.Request) {
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

//
func legacyHandler(w http.ResponseWriter, r *http.Request) {
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

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("[ERROR][500] Bad Request for %s\n", md5)
		return
	}
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

	objectId := vars["objectId"]
	width := vars["width"]
	height := vars["height"]

	if objectId == "" || width == "" || height == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var parseError error
	var intWidth, intHeight int64

	intWidth, parseError = strconv.ParseInt(width, 10, 64)

	if parseError != nil {
		fmt.Printf("parse error")
		return
	}

	intHeight, parseError = strconv.ParseInt(height, 10, 64)

	if parseError != nil {
		fmt.Printf("parse error")
		return
	}

	gridfs := Connection.DB(database).GridFS("fs")

	var fp *mgo.GridFile

	mongoId := bson.ObjectIdHex(objectId)
	query := bson.M{"metadata.parentId": mongoId, "metadata.width": intWidth, "metadata.height": intHeight}
	iter := gridfs.Find(query).Iter()
	gridfs.OpenNext(iter, &fp)

	if fp == nil {
		// schema valid? ist 130x260 erlaubt. Wenn ja: generiere und speichere und liefer aus

		// todo find via id but parentId must be null
		fp, _ = gridfs.OpenId(mongoId)

		if fp != nil {
			fmt.Printf("parent found")

			imageSrc, imageFormat, imgErr := image.Decode(fp)

			if imgErr != nil {
				fmt.Printf("Error is %s", imgErr)
				return
			}

			dst := image.NewRGBA(image.Rect(0, 0, int(intWidth), int(intHeight)))
			graphics.Thumbnail(dst, imageSrc)
			targetFilename := fmt.Sprintf("%d", time.Now().Nanosecond())
			fp, imgErr = gridfs.Create(targetFilename)

			if imgErr != nil {
				defer fp.Close()
				w.WriteHeader(http.StatusNotFound)
				return
			}

			metadata := bson.M{
				"parentId": mongoId,
				"width":    int(intWidth),
				"height":   int(intHeight),
				"size":     fmt.Sprintf("%dx%d", intWidth, intHeight)}

			fp.SetContentType(fmt.Sprintf("image/%s", imageFormat))

			fp.SetMeta(metadata)

			if imageFormat == "png" {
				png.Encode(fp, dst)
			} else if imageFormat == "jpeg" {
				jpeg.Encode(fp, dst, &jpeg.Options{jpeg.DefaultQuality})
			} else {
				fmt.Printf("invalid image type %s", imageFormat)
				return
			}

			fp.Close()

			fp, _ = gridfs.OpenId(fp.Id())

			if fp != nil {
				defer fp.Close()
			} else {
				fmt.Printf("generated image could not be found")
				w.WriteHeader(http.StatusNotFound)
				return
			}

		} else {
			w.WriteHeader(http.StatusNotFound)
			return
		}
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

	_, err := io.Copy(w, fp)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("[ERROR][500] Bad Request for %s\n", md5)
		return
	}
}

//
func Deliver() int {
	configurationFilepath := flag.String("config", "configuration.json", "path to the configuration file")
	serverPort := flag.Int("port", 8000, "the server port where we will serve images")
	host := flag.String("host", "localhost", "the database host")

	flag.Parse()

	_, err := config.CreateConfigFromFile(*configurationFilepath)

	if err != nil {
		fmt.Printf("Error %s", err)
		return -2
	}

	fmt.Printf("Server started. Listening on %d database host is %s\n", *serverPort, *host)

	// in order to simple configure the image server in the proxy configuration of nginx
	// we will be getting every database variable from the request
	serverRoute := "/{database}/{port}/{objectId}/{width}/{height}.jpg"
	fallbackRoute := "/{database}/{port}/{image}"

	Connection, err = mgo.Dial(*host)

	if err != nil {
		log.Fatal("Cannot connect to database")
		return -1
	}

	Connection.SetMode(mgo.Eventual, true)
	Connection.SetSyncTimeout(0)

	r := mux.NewRouter()
	r.HandleFunc("/", welcomeHandler)
	r.HandleFunc(serverRoute, imageHandler)
	r.HandleFunc(fallbackRoute, legacyHandler)

	http.Handle("/", r)

	err = http.ListenAndServe(fmt.Sprintf(":%d", *serverPort), nil)

	if err != nil {
		log.Fatal(err)
		return -1
	}

	return 0
}
