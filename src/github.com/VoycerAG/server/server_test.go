package server

import (
	"fmt"
	"image"
	"image/jpeg"
	"labix.org/v2/mgo"
	. "launchpad.net/gocheck"
	"net/http"
	"os"
	"time"
)

type ServerTestSuite struct{}

var _ = Suite(&ServerTestSuite{})

var testMongoFile *mgo.GridFile
var TestCon *mgo.Session

// SetUpTest creates files for further tests to use
func (s *ServerTestSuite) SetUpTest(c *C) {
	filename, _ := os.Getwd()
	imageFile, err := os.Open(filename + "/../testdata/image.jpg")
	c.Assert(err, IsNil)
	TestCon, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestCon.SetMode(mgo.Monotonic, true)

	tempMongo, mongoErr := TestCon.DB("unittest").GridFS("fs").Create("test.jpg")
	c.Assert(mongoErr, IsNil)

	dIm, _, _ := image.Decode(imageFile)

	jpeg.Encode(tempMongo, dIm, &jpeg.Options{JpegMaximumQuality})
	tempMongo.Close()

	var openErr error

	testMongoFile, openErr = TestCon.DB("unittest").GridFS("fs").Open("test.jpg")

	c.Assert(openErr, IsNil)

	c.Assert(testMongoFile.MD5(), Equals, "d5b390993a34a440891a6f20407f9dde")
}

// TearDownTest removes the created test file.
func (s *ServerTestSuite) TearDownTest(c *C) {
	TestCon, _ = mgo.Dial("localhost")

	if Connection != nil {
		Connection.DB("unittest").DropDatabase()
	}
}

// TestIsModifiedNoCache
func (s *ServerTestSuite) TestIsModifiedNoHeaders(c *C) {
	header := http.Header{}

	c.Assert(isModified(testMongoFile, &header), Equals, true)
}

// TestIsModifiedNoCache
func (s *ServerTestSuite) TestIsModifiedNoCache(c *C) {

	header := http.Header{}
	header.Set("Cache-Control", "no-cache")

	c.Assert(isModified(testMongoFile, &header), Equals, true)
}

// TestIsModifiedMd5Mismatch
func (s *ServerTestSuite) TestIsModifiedMd5Mismatch(c *C) {

	header := http.Header{}
	header.Set("If-None-Match", "invalid md5")

	c.Assert(isModified(testMongoFile, &header), Equals, true)
}

// TestCacheToOldHeader
func (s *ServerTestSuite) TestCacheToOldHeader(c *C) {
	modified := time.Unix(0, 0).Format(time.RFC1123)

	header := http.Header{}
	header.Set("If-None-Match", testMongoFile.MD5())
	header.Set("If-Modified-Since", modified)

	c.Assert(isModified(testMongoFile, &header), Equals, true)
}

// TestCacheHitSuccess
func (s *ServerTestSuite) TestCacheHitSuccess(c *C) {
	modified := time.Now().Format(time.RFC1123)

	header := http.Header{}
	header.Set("If-None-Match", testMongoFile.MD5())
	header.Set("If-Modified-Since", modified)

	c.Assert(isModified(testMongoFile, &header), Equals, false)
}

type ResponseWriterMock struct {
	HeaderData http.Header
	HeaderCode int
}

func (t *ResponseWriterMock) Header() http.Header {
	return t.HeaderData
}

func (t *ResponseWriterMock) Write(b []byte) (int, error) {
	return -1, fmt.Errorf("not implemented")
}

func (t *ResponseWriterMock) WriteHeader(code int) {
	t.HeaderCode = code
}

// TestSetCacheHeaders uses a mocked response writer in order to get header values from method
func (s *ServerTestSuite) TestSetCacheHeaders(c *C) {
	header := http.Header{}
	responseWriter := ResponseWriterMock{header, -1}

	d, _ := time.ParseDuration(fmt.Sprintf("%ds", ImageCacheDuration))

	expires := testMongoFile.UploadDate().Add(d)

	setCacheHeaders(testMongoFile, &responseWriter)

	expectedLastModified := testMongoFile.UploadDate().Format(time.RFC1123)
	expectedExpiryDate := expires.Format(time.RFC1123)
	expectedDate := expectedLastModified

	c.Assert(testMongoFile.MD5(), Equals, header.Get("Etag"))
	c.Assert(fmt.Sprintf("max-age=%d", ImageCacheDuration), Equals, header.Get("Cache-Control"))
	c.Assert(expectedLastModified, Equals, header.Get("Last-Modified"))
	c.Assert(expectedExpiryDate, Equals, header.Get("Expires"))
	c.Assert(expectedDate, Equals, header.Get("Date"))
}

func (s *ServerTestSuite) TestimageHandlerConfigurationNotFound(c *C) {
	Connection, _ = mgo.Dial("localhost")
	Configuration = nil

	requestConfig := ServerConfiguration{
		Database: "unittest",
		FormatName: "jpg",
		Filename: "test.jpg"}

	header := http.Header{}
	responseWriter := ResponseWriterMock{header, -1}

	r, _ := http.NewRequest("GET", "test-url", nil)

	imageHandler(&responseWriter, r, &requestConfig)

	c.Assert(responseWriter.HeaderCode, Equals, 500)
}


func (s *ServerTestSuite) TestimageHandlerConnectionNotFound(c *C) {
	config := Config{}
	Configuration = &config
	Connection = nil

	requestConfig := ServerConfiguration{
		Database: "unittest",
		FormatName: "jpg",
		Filename: "test.jpg"}

	header := http.Header{}
	responseWriter := ResponseWriterMock{header, -1}

	r, _ := http.NewRequest("GET", "test-url", nil)

	imageHandler(&responseWriter, r, &requestConfig)

	c.Assert(responseWriter.HeaderCode, Equals, 500)
}

func (s *ServerTestSuite) TestimageHandlerImageNotFound(c *C) {
	Connection, _ = mgo.Dial("localhost")

	config := Config{}
	config.AllowedEntries = append(config.AllowedEntries, Entry{
			Name: "test",
			Width: 100,
			Height: 200,
			Type: "crop"})

	Configuration = &config

	requestConfig := ServerConfiguration{
		Database: "unittest",
		FormatName: "test",
		Filename: "notexisting.jpg"}

	header := http.Header{}
	responseWriter := ResponseWriterMock{header, -1}

	r, _ := http.NewRequest("GET", "test-url", nil)

	imageHandler(&responseWriter, r, &requestConfig)

	c.Assert(responseWriter.HeaderCode, Equals, 404)
}

func (s *ServerTestSuite) TestimageHandlerImageNotFoundWithoutResize(c *C) {
	Connection, _ = mgo.Dial("localhost")

	config := Config{}
	config.AllowedEntries = append(config.AllowedEntries, Entry{
			Name: "test",
			Width: 100,
			Height: 200,
			Type: "crop"})

	Configuration = &config

	requestConfig := ServerConfiguration{
		Database: "unittest",
		FormatName: "notexisting",
		Filename: "notexisting.jpg"}

	header := http.Header{}
	responseWriter := ResponseWriterMock{header, -1}

	r, _ := http.NewRequest("GET", "test-url", nil)

	imageHandler(&responseWriter, r, &requestConfig)

	c.Assert(responseWriter.HeaderCode, Equals, 400)
}

func (s *ServerTestSuite) TestimageHandlerImageCached(c *C) {
	Connection, _ = mgo.Dial("localhost")

	config := Config{}
	config.AllowedEntries = append(config.AllowedEntries, Entry{
			Name: "test",
			Width: 100,
			Height: 200,
			Type: "crop"})

	Configuration = &config

	requestConfig := ServerConfiguration{
		Database: "unittest",
		FormatName: "",
		Filename: "test.jpg"}

	modified := time.Now().Format(time.RFC1123)

	header := http.Header{}

	responseWriter := ResponseWriterMock{header, -1}

	r, _ := http.NewRequest("GET", "test-url", nil)
	r.Header.Set("If-None-Match", "d5b390993a34a440891a6f20407f9dde")
	r.Header.Set("Cache-Control", fmt.Sprintf("max-age=%d", 1000))
	r.Header.Set("If-Modified-Since", modified)

	imageHandler(&responseWriter, r, &requestConfig)

	c.Assert(responseWriter.HeaderCode, Equals, 304)
}
