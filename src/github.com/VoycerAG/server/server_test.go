package server

import (
	_ "io/ioutil"
	. "launchpad.net/gocheck"
	"net/http"
	_ "syscall"
	"testing"
//	"github.com/gorilla/mux"
//	"github.com/gorilla/context"
//	"fmt"
)

type key int

const VarsKey key = iota

// Checker: IsNil, ErrorMatches, Equals, HasLen, FitsTypeof, DeepEquals, NotNil, Not(Checker)
// Bootstrap unit test suite.
type ServerTestSuite struct{}

var _ = Suite(&ServerTestSuite{})

func Test(t *testing.T) {
	TestingT(t)
}

type MyHttpRequest interface {
	SomeMethod(x int64, y string)
}

func (s *ServerTestSuite) TestValidateVars(c *C) {

	request, _ := http.NewRequest("GET", "http://example.com/database/filename.jpg?size=test", nil)

	vars := make(map[string]string)
	vars["database"] = "database"
	vars["filename"] = "filename.jpg"

	requestConfig, err := validateVars(request, vars)

	c.Assert(err, IsNil)

	c.Assert(requestConfig.FormatName, Equals, "test")
	c.Assert(requestConfig.Database, Equals, "database")
	c.Assert(requestConfig.Filename, Equals, "filename.jpg")
}
