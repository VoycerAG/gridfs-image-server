package server

import (
	"labix.org/v2/mgo"
	. "launchpad.net/gocheck"
	//	"labix.org/v2/mgo/bson"
	//	"image"
	"io"
	"os"
)

// Checker: IsNil, ErrorMatches, Equals, HasLen, FitsTypeof, DeepEquals, NotNil, Not(Checker)
// Bootstrap unit test suite.
type DatabaseTestSuite struct{}

var _ = Suite(&DatabaseTestSuite{})

// SetUpTest creates test datasets to the local mongodb server.
func (s *DatabaseTestSuite) SetUpTest(c *C) {
	Connection, _ = mgo.Dial("localhost")
	gridfs := Connection.DB("unittest").GridFS("fs")

	// create a new file
	gridfsfile, _ := gridfs.Create("image.jpg")

	// set a jpeg image to gridfs
	testJpeg, _ = os.Open("/../testdata/image.jpg")
	defer testJpeg.Close()
	io.Copy(gridfsfile, testJpeg)

	gridfsfile.Close()
}

// TearDownTest removes the created test file.
func (s *DatabaseTestSuite) TearDownTest(c *C) {
	Connection, _ = mgo.Dial("localhost")
	Connection.DB("unittest").DropDatabase()
}

// TestFindImageByParentFilename tests the FindImageByParentFile func
func (s *DatabaseTestSuite) TestFindImageByParentFilename(c *C) {
	Connection, _ = mgo.Dial("localhost")
	gridfs := Connection.DB("unittest").GridFS("fs")

	file, err := FindImageByParentFilename("image.jpg", nil, gridfs)

	c.Assert(file, NotNil)
	c.Assert(err, IsNil)

	file, err = FindImageByParentFilename("imagenotexisting.jpg", nil, gridfs)

	c.Assert(file, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "no image found for imagenotexisting.jpg")
}
