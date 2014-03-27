package config

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
	"syscall"
	"testing"
)

// Checker: IsNil, ErrorMatches, Equals, HasLen, FitsTypeof, DeepEquals, NotNil, Not(Checker)
type TestSuite struct{}

var _ = Suite(&TestSuite{})

func Test(t *testing.T) {
	TestingT(t)
}

// createConfiguration is an utility function for setup of an example configuration.
func createConfiguration(c *C) (*os.File, error) {
	f, err := ioutil.TempFile("", "test.json")

	c.Assert(err, IsNil)

	exampleConfig := `{
	"allowedEntries" : [
		{
			"name" : "peter", 
			"width" : 100, 
			"height" : 200
		}, 
		{
			"name" : "stefan", 
			"width" : 200, 
			"height" : 300
		}
	]
}`

	err = ioutil.WriteFile(f.Name(), []byte(exampleConfig), 0777)

	c.Assert(err, IsNil)

	return f, err
}

// TestOpenFileErrorOnFail tests openFile to return an error.
func (s *TestSuite) TestOpenFileErrorOnFail(c *C) {
	_, err := openFile("/")

	expected := "read /: is a directory"

	c.Assert(err, ErrorMatches, expected)
}

// TestCreateConfigFromFile tests that a config file can be created and has entries.
func (s *TestSuite) TestCreateConfigFromFile(c *C) {
	f, setupErr := createConfiguration(c)

	c.Assert(setupErr, IsNil)

	//cleanup temp file
	defer syscall.Unlink(f.Name())

	configObject, err := CreateConfigFromFile(f.Name())

	c.Assert(err, IsNil, Commentf("loading failed because of %s", err))
	c.Assert(configObject.AllowedEntries, HasLen, 2)
}

// TestCreateConfigFromFileOpenFileFailed tests that opening an invalid file will fail.
func (s *TestSuite) TestCreateConfigFromFileOpenFileFailed(c *C) {
	configObject, err := CreateConfigFromFile("/")
	c.Assert(err, NotNil)

	expected := "read /: is a directory"

	c.Assert(err, ErrorMatches, expected)
	c.Assert(configObject.AllowedEntries, HasLen, 0)
}

// TestOpenFileSuccessCase tests a successful file can be openeded and has the real length.
func (s *TestSuite) TestOpenFileSuccessCase(c *C) {
	f, setupErr := createConfiguration(c)
	c.Assert(setupErr, IsNil)

	//cleanup temp file
	defer syscall.Unlink(f.Name())

	stream, err := openFile(f.Name())
	c.Assert(err, IsNil)
	c.Assert(stream, HasLen, 165)
}
