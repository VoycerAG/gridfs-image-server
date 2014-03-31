package server

import (
	_ "io/ioutil"
	. "launchpad.net/gocheck"
	"net/http"
	_ "syscall"
	"testing"
)

// Checker: IsNil, ErrorMatches, Equals, HasLen, FitsTypeof, DeepEquals, NotNil, Not(Checker)
// Bootstrap unit test suite.
type ServerTestSuite struct{}

var _ = Suite(&ServerTestSuite{})

func Test(t *testing.T) {
	TestingT(t)
}

func (s *ServerTestSuite) TestCreateConfigurationFromVars(c *C) {
	request, _ := http.NewRequest("GET", "http://example.com/database/filename.jpg?size=test", nil)

	vars := make(map[string]string)
	vars["database"] = "database"
	vars["filename"] = "filename.jpg"

	requestConfig, err := createConfigurationFromVars(request, vars)

	c.Assert(err, IsNil)

	c.Assert(requestConfig.FormatName, Equals, "test")
	c.Assert(requestConfig.Database, Equals, "database")
	c.Assert(requestConfig.Filename, Equals, "filename.jpg")

	vars["filename"] = ""
	requestConfig, err = createConfigurationFromVars(request, vars)

	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "filename must not be empty")
	c.Assert(requestConfig, IsNil)

	vars["database"] = ""
	requestConfig, err = createConfigurationFromVars(request, vars)

	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "database must not be empty")
	c.Assert(requestConfig, IsNil)
}
