package server

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"image"
	"io"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"time"
)

const JpegMaximumQuality = 100
const ImageCacheDuration = 315360000

var Connection *mgo.Session
var Configuration *Config

// VarsHandler is a simple wrapper so the request params can be injected into the main handler
type VarsHandler func(http.ResponseWriter, *http.Request, *ServerConfiguration)

// ServeHTTP wraps the imageHandler function and validates request parameters
// in order to create a ServerConfiguration object
func (h VarsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	requestConfig, validateError := CreateConfigurationFromVars(r, vars)

	if validateError != nil {
		log.Printf("%d invalid request parameters given.\n", http.StatusNotFound)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	h(w, r, requestConfig)
}

// addImageMetaData adds data to the image the image
func addImageMetaData(targetImage *mgo.GridFile, imageData image.Image, imageFormat string, originalImage *mgo.GridFile, entry *Entry) {
	width := imageData.Bounds().Dx()
	height := imageData.Bounds().Dy()
	originalRef := mgo.DBRef{"fs.files", originalImage.Id(), ""}

	metadata := bson.M{
		"width":            width,
		"height":           height,
		"original":         originalRef,
		"originalFilename": originalImage.Name(),
		"resizeType":       entry.Type,
		"size":             fmt.Sprintf("%dx%d", width, height)}

	targetImage.SetContentType(fmt.Sprintf("image/%s", imageFormat))
	targetImage.SetMeta(metadata)
}

// isModified returns true if the file must be delivered, false otherwise.
func isModified(file *mgo.GridFile, header *http.Header) bool {
	md5 := file.MD5()
	modifiedHeader := header.Get("If-Modified-Since")
	modifiedTime := time.Now()

	if modifiedHeader != "" {
		modifiedTime, _ = time.Parse(time.RFC1123, modifiedHeader)
	}

	// normalize upload date to use the same format as the browser
	uploadDate, _ := time.Parse(time.RFC1123, file.UploadDate().Format(time.RFC1123))

	if header.Get("Cache-Control") == "no-cache" {
		log.Printf("Is modified, because caching not enabled.")
		return true
	}

	if uploadDate.After(modifiedTime) {
		log.Printf("Is modified, because upload date after modified date.\n")
		return true
	}

	if md5 != header.Get("If-None-Match") {
		log.Printf("Is modified, because md5 mismatch. %s != %s", md5, header.Get("If-None-Match"))
		return true
	}

	return false
}

// setCacheHeaders sets the cache headers into the http.ResponseWriter
func setCacheHeaders(file *mgo.GridFile, w http.ResponseWriter) {
	w.Header().Set("Etag", file.MD5())
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", ImageCacheDuration))
	d, _ := time.ParseDuration(fmt.Sprintf("%ds", ImageCacheDuration))

	expires := file.UploadDate().Add(d)

	w.Header().Set("Last-Modified", file.UploadDate().Format(time.RFC1123))
	w.Header().Set("Expires", expires.Format(time.RFC1123))
	w.Header().Set("Date", file.UploadDate().Format(time.RFC1123))
}

// imageHandler the main handler
func imageHandler(w http.ResponseWriter, r *http.Request, requestConfig *ServerConfiguration) {
	log.Printf("Request on %s", r.URL)

	gridfs := Connection.DB(requestConfig.Database).GridFS("fs")

	resizeEntry, _ := Configuration.GetEntryByName(requestConfig.FormatName)
	foundImage, _ := FindImageByParentFilename(requestConfig.Filename, resizeEntry, gridfs)

	// case that we do not want resizing and did not find any image
	if foundImage == nil && resizeEntry == nil {
		log.Printf("%d invalid request parameters given.\n", http.StatusNotFound)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// we found a image but did not want resizing
	if foundImage != nil {
		if !isModified(foundImage, &r.Header) {
			w.WriteHeader(http.StatusNotModified)
			log.Printf("%d Returning cached image.\n", http.StatusNotModified)
			return
		}

		setCacheHeaders(foundImage, w)

		io.Copy(w, foundImage)
		foundImage.Close()
		log.Printf("%d Image found, no resizing.\n", http.StatusOK)
		return
	}

	if foundImage == nil && resizeEntry != nil {
		// generate new image
		foundImage, _ = FindImageByParentFilename(requestConfig.Filename, nil, gridfs)

		if foundImage == nil {
			log.Printf("%d Could not find original image.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resizedImage, imageFormat, imageErr := ResizeImageFromGridfs(foundImage, resizeEntry)

		// in this case, resizing for this image does not work, therefore, we at least return the original image
		if imageErr != nil {

			// this might be a problem at the moment, go does not support interlaced pngs
			// http://code.google.com/p/go/issues/detail?id=6293
			// at the moment, we return a not found...
			log.Printf("%d image could not be decoded.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// return the image to the client if all cache headers could be set
		targetfile, _ := gridfs.Create(GetRandomFilename(imageFormat))
		encodeErr := EncodeImage(targetfile, *resizedImage, imageFormat)

		if encodeErr != nil {
			log.Fatalf(imageErr.Error())
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("%d image could not be encoded.\n", http.StatusBadRequest)
			return
		}

		addImageMetaData(targetfile, *resizedImage, imageFormat, foundImage, resizeEntry)

		targetfile.Close()

		fp, readErr := gridfs.Open(targetfile.Name())

		if fp != nil {
			if !isModified(fp, &r.Header) {
				w.WriteHeader(http.StatusNotModified)
				log.Printf("%d Returning cached image.\n", http.StatusNotModified)
				return
			}

			setCacheHeaders(fp, w)

			io.Copy(w, fp)
			fp.Close()
			log.Printf("%d image succesfully resized and returned.\n", http.StatusOK)
		} else {
			log.Printf(readErr.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

//Deliver is the startup method that parses configuration files and opens the mongo connection
func Deliver() int {
	configurationFilepath := flag.String("config", "configuration.json", "path to the configuration file")
	serverPort := flag.Int("port", 8000, "the server port where we will serve images")
	host := flag.String("host", "localhost:27017", "the database host with an optional port, localhost would suffice")

	flag.Parse()

	var err error

	Configuration, err = CreateConfigFromFile(*configurationFilepath)

	if err != nil {
		fmt.Printf("Error %s\n", err)
		return -2
	}

	fmt.Printf("Server started. Listening on %d database host is %s\n", *serverPort, *host)

	// in order to simple configure the image server in the proxy configuration of nginx
	// we will be getting every database variable from the request
	serverRoute := "/{database}/{filename}"

	Connection, err = mgo.Dial(*host)

	if err != nil {
		log.Fatal("Cannot connect to database")
		return -1
	}

	Connection.SetMode(mgo.Eventual, true)
	Connection.SetSyncTimeout(0)

	r := mux.NewRouter()
	r.HandleFunc("/", welcomeHandler)
	r.Handle(serverRoute, VarsHandler(imageHandler))

	http.Handle("/", r)

	err = http.ListenAndServe(fmt.Sprintf(":%d", *serverPort), nil)

	if err != nil {
		log.Fatal(err)
		return -1
	}

	return 0
}

// just a static welcome handler
func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html>")
	fmt.Fprintf(w, "<h1>Image Server.</h1>")
	fmt.Fprintf(w, "</html>")
}
