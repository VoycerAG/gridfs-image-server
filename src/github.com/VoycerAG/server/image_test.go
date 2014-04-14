package server

import (
	"image"
	"io"
	"labix.org/v2/mgo"
	. "launchpad.net/gocheck"
	"os"
)

type ImageTestSuite struct{}

var testJpeg *os.File
var testMongoJpeg *mgo.GridFile
var TestConnection *mgo.Session

var _ = Suite(&ImageTestSuite{})

// SetUpTest creates files for further tests to use
func (s *ImageTestSuite) SetUpTest(c *C) {
	filename, _ := os.Getwd()
	var err error
	testJpeg, err = os.Open(filename + "/../testdata/image.jpg")
	c.Assert(err, IsNil)
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	var mongoErr error

	testMongoJpeg, mongoErr = TestConnection.DB("unittest").GridFS("fs").Create("test.jpg")
	c.Assert(mongoErr, IsNil)
}

// TearDownTest removes the created test files.
func (s *ImageTestSuite) TearDownTest(c *C) {
	TestConnection.DB("unittest").DropDatabase()
}

// TestResizeImageInvalidEntryGiven
func (s *ImageTestSuite) TestResizeImageInvalidEntryGiven(c *C) {
	entry := Entry{"test", -1, -1, ""}

	imageData, imageFormat, imageError := ResizeImage(nil, "jpeg", &entry)

	c.Assert(imageData, IsNil)
	c.Assert(imageFormat, Equals, "")
	c.Assert(imageError, ErrorMatches, "At least one parameter of width or height must be specified")
}

// TestValidEntryTypeResizeAndFormatForwarding
func (s *ImageTestSuite) TestValidEntryTypeResizeAndFormatForwarding(c *C) {
	entry := Entry{"test", 350, 400, TypeResize}

	imageStream, _, imgErr := image.Decode(testJpeg)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	imageData, imageFormat, imageError := ResizeImage(imageStream, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 350)
	c.Assert((*imageData).Bounds().Dy(), Equals, 400)
}

// TestValidEntryTypeResizeAndFormatForwardingHeightMissing
func (s *ImageTestSuite) TestValidEntryTypeResizeAndFormatForwardingHeightMissing(c *C) {
	entry := Entry{"test", 350, -1, TypeResize}

	imageStream, _, imgErr := image.Decode(testJpeg)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	imageData, imageFormat, imageError := ResizeImage(imageStream, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 350)
	c.Assert((*imageData).Bounds().Dy(), Equals, 263)
}

// TestValidEntryTypeResizeAndFormatForwardingWidthMissing
func (s *ImageTestSuite) TestValidEntryTypeResizeAndFormatForwardingWidthMissing(c *C) {
	entry := Entry{"test", -1, 400, TypeResize}

	imageStream, _, imgErr := image.Decode(testJpeg)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	imageData, imageFormat, imageError := ResizeImage(imageStream, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 534)
	c.Assert((*imageData).Bounds().Dy(), Equals, 400)
}

// TestValidEntryTypeCropAndNonHeightGiven
func (s *ImageTestSuite) TestValidEntryTypeCropAndNonHeightGiven(c *C) {
	entry := Entry{"test", 400, -1, TypeCrop}

	imageStream, _, imgErr := image.Decode(testJpeg)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	imageData, imageFormat, imageError := ResizeImage(imageStream, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 400)
	c.Assert((*imageData).Bounds().Dy(), Equals, 300)
}

// TestValidEntryTypeCropAndNonWidthGiven
func (s *ImageTestSuite) TestValidEntryTypeCropAndNonWidthGiven(c *C) {
	entry := Entry{"test", -1, 300, TypeCrop}

	imageStream, _, imgErr := image.Decode(testJpeg)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	imageData, imageFormat, imageError := ResizeImage(imageStream, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 400)
	c.Assert((*imageData).Bounds().Dy(), Equals, 300)
}

// TestValidEntryTypeCropAndBothGiven
func (s *ImageTestSuite) TestValidEntryTypeCropAndBothGiven(c *C) {
	entry := Entry{"test", 800, 600, TypeCrop}

	imageStream, _, imgErr := image.Decode(testJpeg)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	imageData, imageFormat, imageError := ResizeImage(imageStream, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 800)
	c.Assert((*imageData).Bounds().Dy(), Equals, 600)
}

// TestEncodeJpegImage
func (s *ImageTestSuite) TestEncodeFunnyImageFormat(c *C) {
	c.Assert(testMongoJpeg, Not(IsNil))

	imageStream, _, imgErr := image.Decode(testJpeg)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	encodeErr := EncodeImage(testMongoJpeg, imageStream, "funny")
	c.Assert(encodeErr, ErrorMatches, "invalid imageFormat given")
}

// TestEncodeJpegImage
func (s *ImageTestSuite) TestEncodeJpegImage(c *C) {
	c.Assert(testMongoJpeg, Not(IsNil))

	imageStream, _, imgErr := image.Decode(testJpeg)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	encodeErr := EncodeImage(testMongoJpeg, imageStream, "jpeg")
	c.Assert(encodeErr, IsNil)

	c.Assert(testMongoJpeg.Size() > 0, Equals, true)
}

// TestEncodePngImageInterlaced
func (s *ImageTestSuite) TestEncodePngImageInterlaced(c *C) {
	c.Skip("This test won't work as long as png does not support interlacing in go @see http://code.google.com/p/go/issues/detail?id=6293")
	filename, _ := os.Getwd()
	testPNG, err := os.Open(filename + "/../testdata/interlaced.png")
	c.Assert(err, IsNil)
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	testMongoPNG, mongoErr := TestConnection.DB("unittest").GridFS("fs").Create("interlaced.png")
	c.Assert(mongoErr, IsNil)
	c.Assert(testMongoPNG, Not(IsNil))

	imageStream, imageType, imgErr := image.Decode(testPNG)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	encodeErr := EncodeImage(testMongoPNG, imageStream, imageType)
	c.Assert(encodeErr, IsNil)
	c.Assert(testMongoPNG.Size() > 0, Equals, true)
}

// TestEncodePngImageNormal
func (s *ImageTestSuite) TestEncodePngImageNormal(c *C) {
	filename, _ := os.Getwd()
	testPNG, err := os.Open(filename + "/../testdata/normal.png")
	c.Assert(err, IsNil)
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	testMongoPNG, mongoErr := TestConnection.DB("unittest").GridFS("fs").Create("normal.png")
	c.Assert(mongoErr, IsNil)
	c.Assert(testMongoPNG, Not(IsNil))

	imageStream, imageType, imgErr := image.Decode(testPNG)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	encodeErr := EncodeImage(testMongoPNG, imageStream, imageType)
	c.Assert(encodeErr, IsNil)
	c.Assert(testMongoPNG.Size() > 0, Equals, true)
}

// TestEncodePngImageTransparent
func (s *ImageTestSuite) TestEncodePngImageTransparent(c *C) {
	filename, _ := os.Getwd()
	testPNG, err := os.Open(filename + "/../testdata/transparent.png")
	c.Assert(err, IsNil)
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	testMongoPNG, mongoErr := TestConnection.DB("unittest").GridFS("fs").Create("transparent.png")
	c.Assert(mongoErr, IsNil)
	c.Assert(testMongoPNG, Not(IsNil))

	imageStream, imageType, imgErr := image.Decode(testPNG)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	encodeErr := EncodeImage(testMongoPNG, imageStream, imageType)
	c.Assert(encodeErr, IsNil)
	c.Assert(testMongoPNG.Size() > 0, Equals, true)
}

// TestEncodeGifNormalImage
func (s *ImageTestSuite) TestEncodeGifImageNormal(c *C) {
	filename, _ := os.Getwd()
	testGif, err := os.Open(filename + "/../testdata/non-animated.gif")
	c.Assert(err, IsNil)
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	testMongoGif, mongoErr := TestConnection.DB("unittest").GridFS("fs").Create("non-animated.gif")
	c.Assert(mongoErr, IsNil)
	c.Assert(testMongoGif, Not(IsNil))

	imageStream, imageType, imgErr := image.Decode(testGif)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 320)
	c.Assert(imageStream.Bounds().Dy(), Equals, 240)

	encodeErr := EncodeImage(testMongoGif, imageStream, imageType)
	c.Assert(encodeErr, IsNil)
	c.Assert(testMongoGif.Size() > 0, Equals, true)
}

// TestEncodeGifNormalImage
func (s *ImageTestSuite) TestEncodeGifImageAnimated(c *C) {
	filename, _ := os.Getwd()
	testGif, err := os.Open(filename + "/../testdata/animated.gif")

	c.Assert(err, IsNil)
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	testMongoGif, mongoErr := TestConnection.DB("unittest").GridFS("fs").Create("animated.gif")
	c.Assert(mongoErr, IsNil)
	c.Assert(testMongoGif, Not(IsNil))

	imageStream, imageType, imgErr := image.Decode(testGif)

	c.Assert(imgErr, IsNil)
	c.Assert(imageStream.Bounds().Dx(), Equals, 306)
	c.Assert(imageStream.Bounds().Dy(), Equals, 350)

	encodeErr := EncodeImage(testMongoGif, imageStream, imageType)
	c.Assert(encodeErr, IsNil)
	c.Assert(testMongoGif.Size() > 0, Equals, true)
}

// TestResizeImageFromGridFsEncodingError
func (s *ImageTestSuite) TestResizeImageFromGridFsInterlacedNoEncodingError(c *C) {
	filename, _ := os.Getwd()
	testPNG, err := os.Open(filename + "/../testdata/interlaced.png")
	c.Assert(err, IsNil)
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	testMongoPNG, mongoErr := TestConnection.DB("unittest").GridFS("fs").Create("interlaced.png")
	c.Assert(mongoErr, IsNil)
	c.Assert(testMongoPNG, Not(IsNil))

	io.Copy(testMongoPNG, testPNG)
	testMongoPNG.Close()

	entry := Entry{"test", 800, 600, TypeCrop}

	testMongoPNG, err = TestConnection.DB("unittest").GridFS("fs").Open("interlaced.png")
	c.Assert(err, IsNil)

	_, _, errResult := ResizeImageFromGridfs(testMongoPNG, &entry)
	c.Assert(errResult, IsNil)
}

//TestResizeImageFromGridFs
func (s *ImageTestSuite) TestResizeImageFromGridFs(c *C) {
	filename, _ := os.Getwd()
	testPNG, err := os.Open(filename + "/../testdata/normal.png")
	c.Assert(err, IsNil)
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	testMongoPNG, mongoErr := TestConnection.DB("unittest").GridFS("fs").Create("normal.png")
	c.Assert(mongoErr, IsNil)
	c.Assert(testMongoPNG, Not(IsNil))

	io.Copy(testMongoPNG, testPNG)
	testMongoPNG.Close()

	entry := Entry{"test", 800, 600, TypeCrop}

	testMongoPNG, err = TestConnection.DB("unittest").GridFS("fs").Open("normal.png")
	c.Assert(err, IsNil)

	imageResult, _, errResult := ResizeImageFromGridfs(testMongoPNG, &entry)
	c.Assert(errResult, IsNil)

	c.Assert((*imageResult).Bounds().Dx(), Equals, 800)
	c.Assert((*imageResult).Bounds().Dy(), Equals, 600)
}

//TestResizeImageFromGridFs
func (s *ImageTestSuite) TestResizeJpegImageFromGridFsCMYK(c *C) {
	filename, _ := os.Getwd()
	testPNG, err := os.Open(filename + "/../testdata/failure.JPG")
	c.Assert(err, IsNil)
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	testMongoPNG, mongoErr := TestConnection.DB("unittest").GridFS("fs").Create("failure.JPG")
	c.Assert(mongoErr, IsNil)
	c.Assert(testMongoPNG, Not(IsNil))

	io.Copy(testMongoPNG, testPNG)
	testMongoPNG.Close()

	entry := Entry{"test", 300, 200, TypeCrop}

	testMongoPNG, err = TestConnection.DB("unittest").GridFS("fs").Open("failure.JPG")
	c.Assert(err, IsNil)

	_, _, errResult := ResizeImageFromGridfs(testMongoPNG, &entry)
	c.Assert(errResult, ErrorMatches, "unsupported JPEG feature: SOF has wrong length")
}

//TestFallbackToImageMagick
func (s *ImageTestSuite) TestFallbackToImageMagick(c *C) {
	filename, _ := os.Getwd()
	testPNG, err := os.Open(filename + "/../testdata/interlaced.png")
	c.Assert(err, IsNil)
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	testMongoPNG, mongoErr := TestConnection.DB("unittest").GridFS("fs").Create("interlaced.png")
	c.Assert(mongoErr, IsNil)
	c.Assert(testMongoPNG, Not(IsNil))

	io.Copy(testMongoPNG, testPNG)
	testMongoPNG.Close()

	testMongoPNG, err = TestConnection.DB("unittest").GridFS("fs").Open("interlaced.png")
	c.Assert(err, IsNil)

	var decodedImage image.Image

	decodedImage, err = imageMagickFallback(testMongoPNG)

	c.Assert(nil, Equals, err)

	c.Assert(decodedImage.Bounds().Dx(), Equals, 320)
	c.Assert(decodedImage.Bounds().Dy(), Equals, 240)
}

//TestFallbackToImageMagickFails
func (s *ImageTestSuite) TestFallbackToImageMagickFails(c *C) {
	var err error
	TestConnection, err = mgo.Dial("localhost")
	c.Assert(err, IsNil)
	TestConnection.SetMode(mgo.Monotonic, true)

	testMongoPNG, mongoErr := TestConnection.DB("unittest").GridFS("fs").Create("interlaced.png")
	c.Assert(mongoErr, IsNil)
	c.Assert(testMongoPNG, Not(IsNil))

	testMongoPNG.Close()

	testMongoPNG, err = TestConnection.DB("unittest").GridFS("fs").Open("interlaced.png")
	c.Assert(err, IsNil)

	_, err = imageMagickFallback(testMongoPNG)

	c.Assert(err, ErrorMatches, "exit status 1")
}

//TestValidEntryTypeFitLandscape
func (s *ImageTestSuite) TestValidEntryTypeFitLandscapeOriginalLandscape(c *C) {
	entry := Entry{"test", 310, 230, TypeFit}

	imageRGBA := image.NewRGBA(image.Rect(0, 0, 168, 105))

	c.Assert(imageRGBA.Bounds().Dx(), Equals, 168)
	c.Assert(imageRGBA.Bounds().Dy(), Equals, 105)

	imageData, imageFormat, imageError := ResizeImage(imageRGBA, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 310)
	c.Assert((*imageData).Bounds().Dy(), Equals, 193)
}

//TestValidEntryTypeFitPortrait
func (s *ImageTestSuite) TestValidEntryTypeFitPortraitOriginalLandscape(c *C) {
	entry := Entry{"test", 130, 260, TypeFit}

	imageRGBA := image.NewRGBA(image.Rect(0, 0, 168, 105))

	c.Assert(imageRGBA.Bounds().Dx(), Equals, 168)
	c.Assert(imageRGBA.Bounds().Dy(), Equals, 105)

	imageData, imageFormat, imageError := ResizeImage(imageRGBA, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 130)
	c.Assert((*imageData).Bounds().Dy(), Equals, 81)
}

//TestValidEntryTypeFitLandscape
func (s *ImageTestSuite) TestValidEntryTypeFitLandscapeOriginalPortrait(c *C) {
	entry := Entry{"test", 310, 230, TypeFit}

	imageRGBA := image.NewRGBA(image.Rect(0, 0, 50, 300))

	c.Assert(imageRGBA.Bounds().Dx(), Equals, 50)
	c.Assert(imageRGBA.Bounds().Dy(), Equals, 300)

	imageData, imageFormat, imageError := ResizeImage(imageRGBA, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 38)
	c.Assert((*imageData).Bounds().Dy(), Equals, 230)
}

//TestValidEntryTypeFitPortrait
func (s *ImageTestSuite) TestValidEntryTypeFitPortraitOriginalPortrait(c *C) {
	entry := Entry{"test", 130, 260, TypeFit}

	imageRGBA := image.NewRGBA(image.Rect(0, 0, 50, 300))

	c.Assert(imageRGBA.Bounds().Dx(), Equals, 50)
	c.Assert(imageRGBA.Bounds().Dy(), Equals, 300)

	imageData, imageFormat, imageError := ResizeImage(imageRGBA, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 43)
	c.Assert((*imageData).Bounds().Dy(), Equals, 260)
}

//TestValidEntryTypeFitPortraitSameRatio
func (s *ImageTestSuite) TestValidEntryTypeFitPortraitSameRatio(c *C) {
	entry := Entry{"test", 130, 260, TypeFit}

	imageRGBA := image.NewRGBA(image.Rect(0, 0, 130, 260))

	c.Assert(imageRGBA.Bounds().Dx(), Equals, 130)
	c.Assert(imageRGBA.Bounds().Dy(), Equals, 260)

	imageData, imageFormat, imageError := ResizeImage(imageRGBA, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 130)
	c.Assert((*imageData).Bounds().Dy(), Equals, 260)
}

//TestValidEntryTypeFitPortraitSameRatio
func (s *ImageTestSuite) TestValidEntryTypeFitLandscapeSameRatio(c *C) {
	entry := Entry{"test", 310, 230, TypeFit}

	imageRGBA := image.NewRGBA(image.Rect(0, 0, 310, 230))

	c.Assert(imageRGBA.Bounds().Dx(), Equals, 310)
	c.Assert(imageRGBA.Bounds().Dy(), Equals, 230)

	imageData, imageFormat, imageError := ResizeImage(imageRGBA, "i do not care", &entry)

	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert((*imageData).Bounds().Dx(), Equals, 310)
	c.Assert((*imageData).Bounds().Dy(), Equals, 230)
}
