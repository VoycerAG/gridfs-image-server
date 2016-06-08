package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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

			imageHandler(w, r, *requestConfig, storage, *z)
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

func imageHandler(
	w http.ResponseWriter,
	r *http.Request,
	requestConfig Configuration,
	storage Storage,
	imageConfig Config,
) {
	log.Printf("Request on %s", r.URL)

	respondWithImage := func(w http.ResponseWriter, r *http.Request, img Cacheable, data io.ReadSeeker) {
		w.Header().Set("Etag", img.CacheIdentifier())
		http.ServeContent(w, r, "", img.LastModified(), data)
		log.Printf("%d Responding with image.\n", http.StatusOK)
	}

	resizeEntry, err := imageConfig.GetEntryByName(requestConfig.FormatName)
	if err != nil { // no valid resize configuration in request
		img, notFoundErr := getOriginalImage(requestConfig.Filename, requestConfig.Database, storage)

		if notFoundErr != nil {
			log.Printf("%d file not found.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		respondWithImage(w, r, img, img.Data())
		log.Printf("%d Image found, no resizing.\n", http.StatusOK)
		return
	}

	img, notFoundErr := getResizeImage(*resizeEntry, requestConfig.Filename, requestConfig.Database, storage)

	if notFoundErr != nil {
		img, err := getOriginalImage(requestConfig.Filename, requestConfig.Database, storage)

		if err != nil {
			log.Printf("%d original file not found.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		customResizers := paint.GetCustomResizers()
		controller, err := paint.NewController(img.Data(), customResizers)

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
			img,
			resizeEntry,
		)

		if err != nil {
			log.Printf("%d error %s\n", http.StatusInternalServerError, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		respondWithImage(w, r, targetfile, bytes.NewReader(data))
		log.Printf("%d image succesfully resized and returned.\n", http.StatusOK)

		return
	}

	respondWithImage(w, r, img, img.Data())
}

func getResizeImage(entry Entry, filename, database string, storage Storage) (Cacheable, error) {
	var foundImage Cacheable
	var err error
	if storage.IsValidID(filename) {
		foundImage, err = storage.FindImageByParentID(database, filename, &entry)
	} else {
		foundImage, err = storage.FindImageByParentFilename(database, filename, &entry)
	}

	return foundImage, err
}

func getOriginalImage(filename, database string, storage Storage) (Cacheable, error) {
	var foundImage Cacheable
	var err error
	if storage.IsValidID(filename) {
		foundImage, err = storage.FindImageByParentID(database, filename, nil)
	} else {
		foundImage, err = storage.FindImageByParentFilename(database, filename, nil)
	}

	return foundImage, err
}

// just a static welcome handler
func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html>")
	fmt.Fprintf(w, "<h1>Image Server.</h1>")
	fmt.Fprintf(w, "</html>")
}
