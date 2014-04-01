package server

import (
	. "launchpad.net/gocheck"
)

// Checker: IsNil, ErrorMatches, Equals, HasLen, FitsTypeof, DeepEquals, NotNil, Not(Checker)
// Bootstrap unit test suite.
type UtilTestSuite struct{}

var _ = Suite(&UtilTestSuite{})

func (s *UtilTestSuite) TestGetRandomFilename(c *C) {
	filename := GetRandomFilename("jpg")
	filenameTwo := GetRandomFilename("jpg")

	c.Assert(filename, Not(Equals), filenameTwo)
	c.Assert(filename, FitsTypeOf, "")
	c.Assert(filenameTwo, FitsTypeOf, "")
	c.Assert(filename, HasLen, 68)
	c.Assert(filenameTwo, HasLen, 68)
}

// TestGetRandomFilenameMultiple this test generates 100 unique filenames.
//There is a small chance that this test fails but it should be extremely low and never happen
func (s *UtilTestSuite) TestGetRandomFilenameMultiple(c *C) {
	stringMap := make(map[string]string, 100)
	var i int
	for i = 0; i < 100; i++ {
		filename := GetRandomFilename("jpg")

		// assert that it is not already generated
		for _, value := range stringMap {
			c.Assert(value, Not(Equals), filename)
		}

		stringMap[filename] = filename
	}
}
