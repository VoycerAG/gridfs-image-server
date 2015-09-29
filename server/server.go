package server

import (
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/yvasiyarov/gorelic"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	//JpegMaximumQuality quality for jpeg compression
	JpegMaximumQuality = 100

	//ImageCacheDuration caching time for images
	ImageCacheDuration = 315360000
)

type imageServer struct {
	connection         *mgo.Session
	imageConfiguration *Config
	handlerMux         http.Handler
}

//Server interface for our server
type Server interface {
	Handler() http.Handler
}

//NewImageServer returns a new image server
func NewImageServer(config *Config, db *mgo.Session) Server {
	return NewImageServerWithNewRelic(config, db, "")
}

//NewImageServerWithNewRelic will return an image server with newrelic monitoring
//licenseKey must be your newrelic license key
func NewImageServerWithNewRelic(config *Config, db *mgo.Session, licenseKey string) Server {
	var handler http.Handler
	// in order to simple configure the image server in the proxy configuration of nginx
	// we will be getting every database variable from the request
	serverRoute := "/{database}/{filename}"

	r := mux.NewRouter()
	r.HandleFunc("/", welcomeHandler)
	//TODO refactor depedency mess
	r.Handle(serverRoute, func(y *mgo.Session, z *Config) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)

			requestConfig, validateError := CreateConfigurationFromVars(r, vars)

			if validateError != nil {
				log.Printf("%d invalid request parameters given.\n", http.StatusNotFound)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			imageHandler(w, r, requestConfig, y, z)
		}
	}(db, config))
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
		"size":             fmt.Sprintf("%dx%d", entry.Width, entry.Height)}

	originalMetadata := bson.M{}

	if err := originalImage.GetMeta(&originalMetadata); err != nil {
		log.Println("Original image data not found.")
	} else {
		for k, v := range originalMetadata {
			if _, exists := metadata[k]; !exists {
				metadata[k] = v
			}
		}
	}

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
func imageHandler(w http.ResponseWriter, r *http.Request, requestConfig *Configuration, connection *mgo.Session, imageConfig *Config) {
	log.Printf("Request on %s", r.URL)

	if connection == nil {
		log.Printf("Connection is not set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if imageConfig == nil {
		log.Printf("imageConfiguration object is not set.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gridfs := connection.DB(requestConfig.Database).GridFS("fs")

	resizeEntry, _ := imageConfig.GetEntryByName(requestConfig.FormatName)

	var foundImage *mgo.GridFile

	if bson.IsObjectIdHex(requestConfig.Filename) {
		foundImage, _ = FindImageByParentID(requestConfig.Filename, resizeEntry, gridfs)
	} else {
		//FindImageByParentFilename will not look for parent filename if
		//entry is not given rofl
		foundImage, _ = FindImageByParentFilename(requestConfig.Filename, resizeEntry, gridfs)
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

		io.Copy(w, foundImage)
		foundImage.Close()
		log.Printf("%d Image found, no resizing.\n", http.StatusOK)
		return
	}

	if foundImage == nil && resizeEntry != nil {
		// generate new image
		if bson.IsObjectIdHex(requestConfig.Filename) {
			foundImage, _ = FindImageByParentID(requestConfig.Filename, nil, gridfs)
		} else {
			foundImage, _ = FindImageByParentFilename(requestConfig.Filename, nil, gridfs)
		}

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

		if targetfile == nil {
			log.Printf("new gridfs file could not be created")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if encodeErr != nil {
			log.Fatalf(imageErr.Error())
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("%d image could not be encoded.\n", http.StatusBadRequest)
			return
		}

		addImageMetaData(targetfile, *resizedImage, imageFormat, foundImage, resizeEntry)

		targetfile.Close()

		setCacheHeaders(targetfile, w)
		EncodeImage(w, *resizedImage, imageFormat)

		log.Printf("%d image succesfully resized and returned.\n", http.StatusOK)
	}
}

// just a static welcome handler
func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html>")
	fmt.Fprintf(w, "<h1>Image Server.</h1>")
	fmt.Fprintf(w, "</html>")
}
