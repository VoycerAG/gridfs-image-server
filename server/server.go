package server

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"

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

// imageHandler the main handler
func imageHandler(
	w http.ResponseWriter,
	r *http.Request,
	requestConfig *Configuration,
	storage Storage,
	imageConfig *Config) {
	log.Printf("Request on %s", r.URL)

	if imageConfig == nil {
		log.Printf("imageConfiguration object is not set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resizeEntry, _ := imageConfig.GetEntryByName(requestConfig.FormatName)

	var foundImage Cacheable
	var notFoundErr error

	if storage.IsValidID(requestConfig.Filename) {
		foundImage, notFoundErr = storage.FindImageByParentID(requestConfig.Database, requestConfig.Filename, resizeEntry)
	} else {
		foundImage, notFoundErr = storage.FindImageByParentFilename(requestConfig.Database, requestConfig.Filename, resizeEntry)
	}

	found := notFoundErr == nil

	// case that we do not want resizing and did not find any image
	if !found && resizeEntry == nil {
		log.Printf("%d file not found.\n", http.StatusNotFound)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// we found an image but did not want resizing
	if found {
		w.Header().Set("Etag", foundImage.CacheIdentifier())

		http.ServeContent(w, r, "", foundImage.LastModified(), foundImage.Data())
		log.Printf("%d Image found, no resizing.\n", http.StatusOK)
		return
	}

	if !found && resizeEntry != nil {
		var notFoundErr error
		if storage.IsValidID(requestConfig.Filename) {
			foundImage, notFoundErr = storage.FindImageByParentID(requestConfig.Database, requestConfig.Filename, nil)
		} else {
			foundImage, notFoundErr = storage.FindImageByParentFilename(requestConfig.Database, requestConfig.Filename, nil)
		}

		found = notFoundErr == nil

		if !found {
			log.Printf("%d Could not find original image.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		customResizers := paint.GetCustomResizers()
		controller, err := paint.NewController(foundImage.Data(), customResizers)

		if err != nil {
			log.Printf("%d image could not be decoded. Reason: [%s].\n", http.StatusNotFound, err.Error())
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
		controller.Encode(buffer)
		buffer.Flush()
		data := b.Bytes()

		targetfile, err := storage.StoreChildImage(
			requestConfig.Database,
			controller.Format(),
			bytes.NewReader(data),
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

		w.Header().Set("Etag", targetfile.CacheIdentifier())
		http.ServeContent(w, r, "", targetfile.LastModified(), bytes.NewReader(data))

		log.Printf("%d image succesfully resized and returned.\n", http.StatusOK)
	}
}

// just a static welcome handler
func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html>")
	fmt.Fprintf(w, "<h1>Image Server.</h1>")
	fmt.Fprintf(w, "</html>")
}
