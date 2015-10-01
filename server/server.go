package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/yvasiyarov/gorelic"
	"gopkg.in/mgo.v2/bson"
)

const (
	//JpegMaximumQuality quality for jpeg compression
	JpegMaximumQuality = 100

	//ImageCacheDuration caching time for images
	ImageCacheDuration = 315360000
)

type imageServer struct {
	imageConfiguration *Config
	storage            GridfsStorage
	handlerMux         http.Handler
}

//Server interface for our server
type Server interface {
	Handler() http.Handler
}

//NewImageServer returns a new image server
func NewImageServer(config *Config, storage GridfsStorage) Server {
	return NewImageServerWithNewRelic(config, storage, "")
}

//NewImageServerWithNewRelic will return an image server with newrelic monitoring
//licenseKey must be your newrelic license key
func NewImageServerWithNewRelic(config *Config, storage GridfsStorage, licenseKey string) Server {
	var handler http.Handler
	// in order to simple configure the image server in the proxy configuration of nginx
	// we will be getting every database variable from the request
	serverRoute := "/{database}/{filename}"

	r := mux.NewRouter()
	r.HandleFunc("/", welcomeHandler)
	//TODO refactor depedency mess
	r.Handle(serverRoute, func(storage GridfsStorage, z *Config) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)

			requestConfig, validateError := CreateConfigurationFromVars(r, vars)

			if validateError != nil {
				log.Printf("%d invalid request parameters given.\n", http.StatusNotFound)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			imageHandler(w, r, requestConfig, storage, z)
		}
	}(storage, config))
	http.Handle("/", r)

	handler = http.DefaultServeMux

	if licenseKey != "" {
		agent := gorelic.NewAgent()
		agent.NewrelicLicense = licenseKey
		agent.NewrelicName = "Go image server"
		agent.CollectHTTPStat = true
		agent.Run()
		handler = agent.WrapHTTPHandler(handler)
	}

	handlerMux := context.ClearHandler(handler)
	return &imageServer{imageConfiguration: config, handlerMux: handlerMux}
}

//Handler is the startup method that parses configuration files and opens the mongo connection
func (i imageServer) Handler() http.Handler {
	return i.handlerMux
}

// isModified returns true if the file must be delivered, false otherwise.
func isModified(c Cacheable, header *http.Header) bool {
	md5 := c.CacheIdentifier()
	modifiedHeader := header.Get("If-Modified-Since")
	modifiedTime := time.Now()

	if modifiedHeader != "" {
		modifiedTime, _ = time.Parse(time.RFC1123, modifiedHeader)
	}

	// normalize upload date to use the same format as the browser
	uploadDate, _ := time.Parse(time.RFC1123, c.LastModified().Format(time.RFC1123))

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

	log.Println("not modified")

	return false
}

// setCacheHeaders sets the cache headers into the http.ResponseWriter
func setCacheHeaders(c Cacheable, w http.ResponseWriter) {
	w.Header().Set("Etag", c.CacheIdentifier())
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", ImageCacheDuration))
	d, _ := time.ParseDuration(fmt.Sprintf("%ds", ImageCacheDuration))

	expires := c.LastModified().Add(d)

	w.Header().Set("Last-Modified", c.LastModified().Format(time.RFC1123))
	w.Header().Set("Expires", expires.Format(time.RFC1123))
	w.Header().Set("Date", c.LastModified().Format(time.RFC1123))
}

// imageHandler the main handler
func imageHandler(w http.ResponseWriter, r *http.Request, requestConfig *Configuration, storage GridfsStorage, imageConfig *Config) {
	log.Printf("Request on %s", r.URL)

	if imageConfig == nil {
		log.Printf("imageConfiguration object is not set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resizeEntry, _ := imageConfig.GetEntryByName(requestConfig.FormatName)

	var foundImage Cacheable

	if bson.IsObjectIdHex(requestConfig.Filename) {
		foundImage, _ = storage.FindImageByParentID(requestConfig.Database, requestConfig.Filename, resizeEntry)
	} else {
		//FindImageByParentFilename will not look for parent filename if
		//entry is not given rofl
		foundImage, _ = storage.FindImageByParentFilename(requestConfig.Database, requestConfig.Filename, resizeEntry)
	}

	// case that we do not want resizing and did not find any image
	if foundImage == nil && resizeEntry == nil {
		log.Printf("%d file not found.\n", http.StatusNotFound)
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

		io.Copy(w, foundImage.Data())
		foundImage.Data().Close()
		log.Printf("%d Image found, no resizing.\n", http.StatusOK)
		return
	}

	if foundImage == nil && resizeEntry != nil {
		// generate new image
		if bson.IsObjectIdHex(requestConfig.Filename) {
			foundImage, _ = storage.FindImageByParentID(requestConfig.Database, requestConfig.Filename, nil)
		} else {
			foundImage, _ = storage.FindImageByParentFilename(requestConfig.Database, requestConfig.Filename, nil)
		}

		if foundImage == nil {
			log.Printf("%d Could not find original image.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resizedImage, imageFormat, imageErr := ResizeImageByEntry(foundImage.Data(), resizeEntry)

		// in this case, resizing for this image does not work, therefore, we at least return the original image
		if imageErr != nil {
			log.Printf("%d image could not be decoded.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		targetfile, err := storage.NewImage(requestConfig.Database, GetRandomFilename(imageFormat), imageFormat, resizedImage, foundImage, resizeEntry, map[string]interface{}{})
		if err != nil {
			log.Fatal(err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		setCacheHeaders(targetfile, w)
		EncodeImage(w, resizedImage, imageFormat)

		log.Printf("%d image succesfully resized and returned.\n", http.StatusOK)
	}
}

// just a static welcome handler
func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html>")
	fmt.Fprintf(w, "<h1>Image Server.</h1>")
	fmt.Fprintf(w, "</html>")
}
