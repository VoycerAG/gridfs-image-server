package server

import (
	"code.google.com/p/graphics-go/graphics"
	"errors"
	"flag"
	"fmt"
	"github.com/VoycerAG/config"
	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
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
var Configuration *config.Config

// wrapper object for request parameters
type ServerConfiguration struct {
	Database   string
	FormatName string
	Filename   string
}

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

func setCacheHeaders(file *mgo.GridFile, w http.ResponseWriter) {
	w.Header().Set("Etag", file.MD5())
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", ImageCacheDuration))
	d, _ := time.ParseDuration(fmt.Sprintf("%ds", ImageCacheDuration))

	expires := file.UploadDate().Add(d)

	w.Header().Set("Last-Modified", file.UploadDate().Format(time.RFC1123))
	w.Header().Set("Expires", expires.Format(time.RFC1123))
	w.Header().Set("Date", file.UploadDate().Format(time.RFC1123))
}

type VarsHandler func(http.ResponseWriter, *http.Request, map[string]string)

func (h VarsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	h(w, r, vars)
}

// imageHandler blub
func imageHandler(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	log.Printf("Request on %s", r.URL)

	requestConfig, validateError := validateParameters(r)

	if validateError != nil {
		log.Printf("%d invalid request parameters given.\n", http.StatusNotFound)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	gridfs := Connection.DB(requestConfig.Database).GridFS("fs")

	resizeEntry, _ := Configuration.GetEntryByName(requestConfig.FormatName)
	foundImage, _ := findImageByParentFilename(requestConfig.Filename, resizeEntry, gridfs)

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
		foundImage, _ = findImageByParentFilename(requestConfig.Filename, nil, gridfs)

		if foundImage == nil {
			log.Printf("%d Could not find original image.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resizedImage, imageFormat, imageErr := resizeImage(foundImage, resizeEntry)

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
		targetfile, _ := gridfs.Create(generateFilename(imageFormat))
		storeErr := storeImage(targetfile, *resizedImage, imageFormat, foundImage, resizeEntry)

		if storeErr != nil {
			log.Fatalf(imageErr.Error())
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("%d image could not be saved.\n", http.StatusBadRequest)
			return
		}

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

// generateFilename generates a new filename
func generateFilename(imageFormat string) string {
	return fmt.Sprintf("%d.%s", time.Now().Nanosecond(), imageFormat)
}

// storeImage
func storeImage(targetImage *mgo.GridFile, imageData image.Image, imageFormat string, originalImage *mgo.GridFile, entry *config.Entry) error {
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

	switch imageFormat {
	case "jpeg":
		jpeg.Encode(targetImage, imageData, &jpeg.Options{JpegMaximumQuality})
	case "png":
		png.Encode(targetImage, imageData)
	//case "gif":
	//gif.Encode(targetImage, imageData, &gif.Options{256})
	default:
		return fmt.Errorf("invalid imageFormat given")
	}

	targetImage.Close()

	return nil
}

// resizeImage resizes images or crops them if either size is not defined
func resizeImage(originalImage *mgo.GridFile, entry *config.Entry) (*image.Image, string, error) {
	if entry.Width < 0 && entry.Height < 0 {
		return nil, "", fmt.Errorf("At least one parameter of width or height must be specified")
	}

	originalImageData, imageFormat, imgErr := image.Decode(originalImage)

	if imgErr != nil {
		return nil, imageFormat, imgErr
	}

	targetHeight := float64(entry.Height)
	targetWidth := float64(entry.Width)

	if targetWidth < 0 {
		targetWidth = 0
	}

	if targetHeight < 0 {
		targetHeight = 0
	}

	imageRGBA := image.NewRGBA(image.Rect(0, 0, int(targetWidth), int(targetHeight)))
	err := graphics.Thumbnail(imageRGBA, originalImageData)

	var dst image.Image

	if entry.Type == config.TypeResize {
		dst = resize.Resize(uint(targetWidth), uint(targetHeight), originalImageData, resize.Lanczos3)
	} else {
		dst = imageRGBA.SubImage(image.Rect(0, 0, int(targetWidth), int(targetHeight)))
	}

	return &dst, imageFormat, err
}

// findImageByParentFilename returns either the resized image that actually exists, or the original if entry is nil
func findImageByParentFilename(filename string, entry *config.Entry, gridfs *mgo.GridFS) (*mgo.GridFile, error) {
	var fp *mgo.GridFile
	var query bson.M

	if entry == nil {
		query = bson.M{"filename": filename}
	} else {
		query = bson.M{
			"metadata.originalFilename": filename,
			"metadata.width":            entry.Width,
			"metadata.resizeType":       entry.Type,
			"metadata.height":           entry.Height}
	}

	iter := gridfs.Find(query).Iter()
	gridfs.OpenNext(iter, &fp)

	if fp == nil {
		return fp, fmt.Errorf("no image found for %s", filename)
	}

	return fp, nil
}

// validateParameters validate all necessary request parameters
func validateParameters(r *http.Request) (*ServerConfiguration, error) {
	config := ServerConfiguration{}
	vars := mux.Vars(r)

	database := vars["database"]

	if database == "" {
		return nil, errors.New("database must not be empty")
	}

	filename := vars["filename"]

	if filename == "" {
		return nil, errors.New("filename must not be empty")
	}

	formatName := r.URL.Query().Get("size")

	config.Database = database
	config.FormatName = formatName
	config.Filename = filename

	return &config, nil
}

//
func Deliver() int {
	configurationFilepath := flag.String("config", "configuration.json", "path to the configuration file")
	serverPort := flag.Int("port", 8000, "the server port where we will serve images")
	host := flag.String("host", "localhost:27017", "the database host with an optional port, localhost would suffice")

	flag.Parse()

	var err error

	Configuration, err = config.CreateConfigFromFile(*configurationFilepath)

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
