package server

import (
	"io"
	"os"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	. "launchpad.net/gocheck"
)

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
	io.Copy(gridfsfile, testJpeg)

	gridfsfile.Close()

	// create a new file with a parent
	childFile, _ := gridfs.Create("child.jpg")

	metadata := bson.M{
		"width":            100,
		"height":           200,
		"size":             "100x200",
		"originalFilename": "original.jpg",
		"resizeType":       "crop"}

	childFile.SetMeta(metadata)

	// set a jpeg image to gridfs
	defer testJpeg.Close()
	io.Copy(childFile, testJpeg)
	childFile.Close()

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
	c.Assert(file.Name(), Equals, "image.jpg")
	c.Assert(err, IsNil)

	file, err = FindImageByParentFilename("imagenotexisting.jpg", nil, gridfs)

	c.Assert(file, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "no image found for imagenotexisting.jpg")

	entry := Entry{
		Width:  100,
		Height: 200,
		Type:   "crop"}

	file, err = FindImageByParentFilename("original.jpg", &entry, gridfs)

	c.Assert(file, NotNil)
	c.Assert(file.Name(), Equals, "child.jpg")
	c.Assert(err, IsNil)
}
