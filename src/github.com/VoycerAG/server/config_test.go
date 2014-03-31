package server

import (
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
	"syscall"
)

var testfile *os.File

// Checker: IsNil, ErrorMatches, Equals, HasLen, FitsTypeof, DeepEquals, NotNil, Not(Checker)
// Bootstrap unit test suite.

// SetUpTest creates a test file for all tests to use.
func (s *ServerTestSuite) SetUpTest(c *C) {
	var err error

	testfile, err = ioutil.TempFile("", "test.json")

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
			"height" : 300,
			"type" : "cut"
		}
	]
}`

	err = ioutil.WriteFile(testfile.Name(), []byte(exampleConfig), 0777)

	c.Assert(err, IsNil)
}

// TearDownTest removes the created test file.
func (s *ServerTestSuite) TearDownTest(c *C) {
	defer syscall.Unlink(testfile.Name())
}

// TestOpenFileErrorOnFail tests openFile to return an error.
func (s *ServerTestSuite) TestOpenFileErrorOnFail(c *C) {
	_, err := CreateConfigFromFile("/")

	expected := "read /: is a directory"

	c.Assert(err, ErrorMatches, expected)
}

// TestCreateConfigFromFile tests that a config file can be created and has entries.
func (s *ServerTestSuite) TestCreateConfigFromFile(c *C) {
	f := testfile
	c.Assert(f, FitsTypeOf, &os.File{})

	configObject, err := CreateConfigFromFile(f.Name())

	c.Assert(err, IsNil, Commentf("loading failed because of %s", err))
	c.Assert(configObject.AllowedEntries, HasLen, 2)

	firstEntry := configObject.AllowedEntries[0]
	c.Assert(firstEntry, FitsTypeOf, Entry{})

	secondEntry := configObject.AllowedEntries[1]
	c.Assert(secondEntry, FitsTypeOf, Entry{})

	c.Assert(firstEntry.Name, Equals, "peter")
	c.Assert(firstEntry.Width, Equals, int64(100))
	c.Assert(firstEntry.Height, Equals, int64(200))

	c.Assert(secondEntry.Name, Equals, "stefan")
	c.Assert(secondEntry.Width, Equals, int64(200))
	c.Assert(secondEntry.Height, Equals, int64(300))
}

// TestCreateConfigFromFileOpenFileFailed tests that opening an invalid file will fail.
func (s *ServerTestSuite) TestCreateConfigFromFileOpenFileFailed(c *C) {
	configObject, err := CreateConfigFromFile("/")
	c.Assert(err, NotNil)

	expected := "read /: is a directory"

	c.Assert(err, ErrorMatches, expected)
	c.Assert(configObject.AllowedEntries, HasLen, 0)
}

// TestGetConfigElementByName tests that the config element can return a specific configuration element by its name.
func (s *ServerTestSuite) TestGetConfigEntryByName(c *C) {
	f := testfile
	c.Assert(f, FitsTypeOf, &os.File{})

	configObject, _ := CreateConfigFromFile(f.Name())

	stefanConfigElement, err := configObject.GetEntryByName("stefan")

	c.Assert(err, IsNil)
	c.Assert(stefanConfigElement, FitsTypeOf, &Entry{})
	c.Assert(stefanConfigElement.Width, Equals, int64(200))
	c.Assert(stefanConfigElement.Height, Equals, int64(300))

	peterConfigElement, err := configObject.GetEntryByName("peter")

	c.Assert(err, IsNil)
	c.Assert(peterConfigElement, FitsTypeOf, &Entry{})
	c.Assert(peterConfigElement.Width, Equals, int64(100))
	c.Assert(peterConfigElement.Height, Equals, int64(200))

	notExistingElement, err := configObject.GetEntryByName("notExisting")

	c.Assert(err, NotNil)
	c.Assert(notExistingElement, IsNil)
}

// TestValidateConfig tests that the config elements will be validated correctly.
func (s *ServerTestSuite) TestValidateConfigValid(c *C) {
	f := testfile

	c.Assert(f, FitsTypeOf, &os.File{})

	configObject, _ := CreateConfigFromFile(f.Name())

	err := configObject.validateConfig()
	c.Assert(err, IsNil)

	invalidEntry := Entry{
		Name:   "invalid",
		Width:  -1,
		Height: -1}

	configObject.AllowedEntries = append(configObject.AllowedEntries, invalidEntry)

	err = configObject.validateConfig()
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "The width and height of the configuration element with name \"invalid\" are invalid.")
}

// TestValidateConfigInvalidType tests that an error will be returned when an invalid type was given
func (s *ServerTestSuite) TestValidateConfigInvalidType(c *C) {
	f := testfile

	c.Assert(f, FitsTypeOf, &os.File{})

	configObject, _ := CreateConfigFromFile(f.Name())

	err := configObject.validateConfig()
	c.Assert(err, IsNil)

	invalidEntry := Entry{
		Name:   "invalid",
		Width:  320,
		Height: 240,
		Type:   "none-defined"}

	configObject.AllowedEntries = append(configObject.AllowedEntries, invalidEntry)

	err = configObject.validateConfig()
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "Type must be either cut or resize at element \"invalid\"")
}

// TestValidateConfigInvalidType tests that an error will be returned when an invalid type was given
func (s *ServerTestSuite) TestValidateConfigValidType(c *C) {
	f := testfile

	c.Assert(f, FitsTypeOf, &os.File{})

	configObject, _ := CreateConfigFromFile(f.Name())

	err := configObject.validateConfig()
	c.Assert(err, IsNil)

	invalidEntry := Entry{
		Name:   "invalid",
		Width:  320,
		Height: 240,
		Type:   TypeResize}

	configObject.AllowedEntries = append(configObject.AllowedEntries, invalidEntry)

	err = configObject.validateConfig()
	c.Assert(err, IsNil)
}

// TestConfigurationDefaultTypeIfNoneSet test the default case for configurations
func (s *ServerTestSuite) TestConfigurationDefaultTypeIfNoneSet(c *C) {
	f := testfile
	c.Assert(f, FitsTypeOf, &os.File{})

	configObject, _ := CreateConfigFromFile(f.Name())

	err := configObject.validateConfig()

	c.Assert(err, IsNil)
	c.Assert(configObject.AllowedEntries, HasLen, 2)
	c.Assert(configObject.AllowedEntries[0].Type, Equals, TypeResize)
}
