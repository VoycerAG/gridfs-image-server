package server

import (
	. "launchpad.net/gocheck"
	"net/http"
	"testing"
)

// Checker: IsNil, ErrorMatches, Equals, HasLen, FitsTypeof, DeepEquals, NotNil, Not(Checker)
// Bootstrap unit test suite.
type ServerConfigTestSuite struct{}

var _ = Suite(&ServerConfigTestSuite{})

func Test(t *testing.T) {
	TestingT(t)
}

func (s *ServerConfigTestSuite) TestCreateConfigurationFromVars(c *C) {
	request, _ := http.NewRequest("GET", "http://example.com/database/filename.jpg?size=test", nil)

	vars := make(map[string]string)
	vars["database"] = "database"
	vars["filename"] = "filename.jpg"

	requestConfig, err := CreateConfigurationFromVars(request, vars)

	c.Assert(err, IsNil)

	c.Assert(requestConfig.FormatName, Equals, "test")
	c.Assert(requestConfig.Database, Equals, "database")
	c.Assert(requestConfig.Filename, Equals, "filename.jpg")

	vars["filename"] = ""
	requestConfig, err = CreateConfigurationFromVars(request, vars)

	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "filename must not be empty")
	c.Assert(requestConfig, IsNil)

	vars["database"] = ""
	requestConfig, err = CreateConfigurationFromVars(request, vars)

	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "database must not be empty")
	c.Assert(requestConfig, IsNil)
}
