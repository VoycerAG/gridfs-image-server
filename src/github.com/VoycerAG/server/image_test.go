package server

import (
	"image"
	_ "labix.org/v2/mgo"
	. "launchpad.net/gocheck"
	"os"
)

// Checker: IsNil, ErrorMatches, Equals, HasLen, FitsTypeof, DeepEquals, NotNil, Not(Checker)
// Bootstrap unit test suite.
type ImageTestSuite struct{}

var testJpeg *os.File

var _ = Suite(&ImageTestSuite{})

// SetUpTest creates files for further tests to use
func (s *ImageTestSuite) SetUpTest(c *C) {
	filename, _ := os.Getwd()
	var err error
	testJpeg, err = os.Open(filename + "/../testdata/image.jpg")
	c.Assert(err, IsNil)
}

// TearDownTest removes the created test files.
func (s *ImageTestSuite) TearDownTest(c *C) {

}

func (s *ImageTestSuite) TestResizeImageInvalidEntryGiven(c *C) {
	entry := Entry{"test", -1, -1, ""}

	imageData, imageFormat, imageError := ResizeImage(nil, "jpeg", &entry)

	c.Assert(imageData, IsNil)
	c.Assert(imageFormat, Equals, "")
	c.Assert(imageError, ErrorMatches, "At least one parameter of width or height must be specified")
}

func (s *ImageTestSuite) TestValidEntryTypeResizeAndFormatForwarding(c *C) {
	entry := Entry{"test", 400, 300, TypeResize}

	imageStream, _, imgErr := image.Decode(testJpeg)
	c.Assert(imgErr, IsNil)

	imageData, imageFormat, imageError := ResizeImage(imageStream, "i do not care", &entry)

	//c.Assert(imageData, IsNil)
	c.Assert(imageFormat, Equals, "i do not care")
	c.Assert(imageError, IsNil)
	c.Assert(imageData.Bounds().Dx(), Equals, 400)
}
