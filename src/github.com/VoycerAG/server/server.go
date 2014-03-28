package server

import (
	"code.google.com/p/graphics-go/graphics"
	"errors"
	"flag"
	"fmt"
	"github.com/VoycerAG/config"
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
	"time"
)

var Connection *mgo.Session
var Configuration *config.Config

// wrapper object for request parameters
type ServerConfiguration struct {
	Database   string
	FormatName string
	Filename   string
}

// imageHandler blub
func imageHandler(w http.ResponseWriter, r *http.Request) {
	requestConfig, validateError := validateParameters(r)

	if validateError != nil {
		log.Printf("%d invalid request parameters given.\n", http.StatusNotFound)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	gridfs := Connection.DB(requestConfig.Database).GridFS("fs")
	resizeEntry, _ := Configuration.GetEntryByName(requestConfig.FormatName)
	foundImage, err := findImageByParentFilename(requestConfig.Filename, resizeEntry, gridfs)

	if err != nil {
		log.Fatalf(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// case that we do not want resizing and did not find any image
	if foundImage == nil && resizeEntry == nil {
		log.Printf("%d invalid request parameters given.\n", http.StatusNotFound)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	defer foundImage.Close()

	// we found a image but did not want resizing
	if foundImage != nil {
		io.Copy(w, foundImage)
		return
	}

	if foundImage == nil && resizeEntry != nil {
		// generate new image
		foundImage, _ = findImageByParentFilename(requestConfig.Filename, nil, gridfs)

		if foundImage == nil {
			log.Printf("Could not find original image.\n", http.StatusNotFound)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resizedImage, imageFormat, imageErr := resizeImage(foundImage, resizeEntry)

		if imageErr != nil {
			log.Fatalf(imageErr.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// return the image to the client if all cache headers could be set
		targetfile, _ := gridfs.Create(generateFilename(imageFormat))
		storeErr := storeImage(targetfile, resizedImage, imageFormat, foundImage)

		if storeErr != nil {
			log.Fatalf(imageErr.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fp, _ := gridfs.Open(targetfile.Name())
		defer fp.Close()
		io.Copy(w, fp)
	}

}

// generateFilename generates a new filename
func generateFilename(imageFormat string) string {
	return fmt.Sprintf("%d.%s", time.Now().Nanosecond(), imageFormat)
}

// storeImage
func storeImage(targetImage *mgo.GridFile, imageData *image.RGBA, imageFormat string, originalImage *mgo.GridFile) error {
	width := imageData.Bounds().Dx()
	height := imageData.Bounds().Dy()
	originalRef := mgo.DBRef{"fs.files", originalImage.Id(), ""}

	metadata := bson.M{
		"width":            width,
		"height":           height,
		"original":         originalRef,
		"originalFilename": originalImage.Name(),
		"size":             fmt.Sprintf("%dx%d", width, height)}

	targetImage.SetContentType(fmt.Sprintf("image/%s", imageFormat))
	targetImage.SetMeta(metadata)

	switch imageFormat {
	case "jpeg":
		jpeg.Encode(targetImage, imageData, &jpeg.Options{jpeg.DefaultQuality})
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
func resizeImage(originalImage *mgo.GridFile, entry *config.Entry) (*image.RGBA, string, error) {
	if entry.Width < 0 && entry.Height < 0 {
		return nil, "", fmt.Errorf("At least one parameter of width or height must be specified")
	}

	originalImageData, imageFormat, imgErr := image.Decode(originalImage)

	originalBounds := originalImageData.Bounds()
	originalRatio := float64(originalBounds.Dx()) / float64(originalBounds.Dy())

	if imgErr != nil {
		return nil, imageFormat, imgErr
	}

	targetHeight := float64(entry.Height)
	targetWidth := float64(entry.Width)

	if targetWidth < 0 {
		targetWidth = float64(targetHeight) * originalRatio
	}

	if targetHeight < 0 {
		targetHeight = float64(targetWidth) * originalRatio
	}

	dst := image.NewRGBA(image.Rect(0, 0, int(targetWidth), int(targetHeight)))
	err := graphics.Thumbnail(dst, originalImageData)

	return dst, imageFormat, err
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

	formatName := r.URL.Query().Get("format")

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
		fmt.Printf("Error %s", err)
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
	r.HandleFunc(serverRoute, imageHandler)

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

func hasCached(etag string, md5 string, modifiedTime time.Time, updateTime time.Time) bool {
	if updateTime.After(modifiedTime) || md5 != etag {
		return false
	}

	return true
}
