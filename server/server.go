package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/VoycerAG/gridfs-image-server/server/paint"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/yvasiyarov/gorelic"
)

const (
	//ImageCacheDuration caching time for images
	ImageCacheDuration = 315360000
)

type imageServer struct {
	imageConfiguration *Config
	storage            Storage
	handlerMux         http.Handler
}

//Server interface for our server
type Server interface {
	Handler() http.Handler
}

//NewImageServer returns a new image server
func NewImageServer(config *Config, storage Storage) Server {
	return NewImageServerWithNewRelic(config, storage, "")
}

//NewImageServerWithNewRelic will return an image server with newrelic monitoring
//licenseKey must be your newrelic license key
func NewImageServerWithNewRelic(config *Config, storage Storage, licenseKey string) Server {
	var handler http.Handler
	// in order to simple configure the image server in the proxy configuration of nginx
	// we will be getting every database variable from the request
	serverRoute := "/{database}/{filename}"

	r := mux.NewRouter()
	r.HandleFunc("/", welcomeHandler)
	//TODO refactor depedency mess
	r.Handle(serverRoute, func(storage Storage, z *Config) http.HandlerFunc {
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
func imageHandler(w http.ResponseWriter, r *http.Request, requestConfig *Configuration, storage Storage, imageConfig *Config) {
	log.Printf("Request on %s", r.URL)

	if imageConfig == nil {
		log.Printf("imageConfiguration object is not set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resizeEntry, _ := imageConfig.GetEntryByName(requestConfig.FormatName)

	var foundImage Cacheable

	if storage.IsValidID(requestConfig.Filename) {
		foundImage, _ = storage.FindImageByParentID(requestConfig.Database, requestConfig.Filename, resizeEntry)
	} else {
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
		if storage.IsValidID(requestConfig.Filename) {
			foundImage, _ = storage.FindImageByParentID(requestConfig.Database, requestConfig.Filename, nil)
		} else {
			foundImage, _ = storage.FindImageByParentFilename(requestConfig.Database, requestConfig.Filename, nil)
		}

		if foundImage == nil {
			log.Printf("%d Could not find original image.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		controller, err := paint.NewController(foundImage.Data())

		if err != nil {
			log.Printf("%d image could not be decoded.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		err = controller.Resize(resizeEntry.Type, int(resizeEntry.Width), int(resizeEntry.Height))
		if err != nil {
			log.Printf("%d image could not be resized.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return

		}

		var b bytes.Buffer
		buffer := bufio.NewWriter(&b)
		writer := io.MultiWriter(w, buffer)
		controller.Encode(writer)
		buffer.Flush()

		targetfile, err := storage.StoreChildImage(
			requestConfig.Database,
			controller.Format(),
			bytes.NewReader(b.Bytes()),
			controller.Image().Bounds().Dx(),
			controller.Image().Bounds().Dy(),
			foundImage,
			resizeEntry,
		)

		if err != nil {
			log.Fatal(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		setCacheHeaders(targetfile, w)

		log.Printf("%d image succesfully resized and returned.\n", http.StatusOK)
	}
}

// just a static welcome handler
func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html>")
	fmt.Fprintf(w, "<h1>Image Server.</h1>")
	fmt.Fprintf(w, "</html>")
}
