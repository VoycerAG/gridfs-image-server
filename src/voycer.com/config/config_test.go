package config

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"
)

// utility function for setup a example configuration
func createTemporaryConfiguration(t *testing.T) (*os.File, error) {
	f, err := ioutil.TempFile("", "test.json")

	if err != nil {
		t.Errorf("tempfile could not be created")
	}

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
	if err != nil {
		t.Errorf("example config could not be written")
	}

	return f, err
}

func TestCreateConfigFromFile(t *testing.T) {
	f, setupErr := createTemporaryConfiguration(t)

	if setupErr != nil {
		t.Errorf("example config could not be written")
	}

	//cleanup temp file
	defer syscall.Unlink(f.Name())

	configObject, err := CreateConfigFromFile(f.Name())

	if err != nil {
		t.Errorf("loading failed because of %s", err)
	}

	if len(configObject.AllowedEntries) != 2 {
		t.Errorf("not all config entries could be loaded")
	}

}

func TestCreateConfigFromFileOpenFileFailed(t *testing.T) {
	configObject, err := CreateConfigFromFile("/")

	if err == nil {
		t.Errorf("error must be nil")
	}

	expected := "read /: is a directory"

	if err.Error() != expected {
		t.Errorf("invalid message %s != %s", err, expected)
	}

	if len(configObject.AllowedEntries) != 0 {
		t.Error("configObject should be empty.")
	}
}

func TestOpenFileErrorOnFail(t *testing.T) {
	_, err := openFile("/")

	if err == nil {
		t.Errorf("error must be nil")
	}

	expected := "read /: is a directory"

	if err.Error() != expected {
		t.Errorf("invalid message %s != %s", err, expected)
	}
}

func TestOpenFileSuccessCase(t *testing.T) {
	f, setupErr := createTemporaryConfiguration(t)

	if setupErr != nil {
		t.Errorf("example config could not be written")
	}

	//cleanup temp file
	defer syscall.Unlink(f.Name())

	stream, err := openFile(f.Name())

	if err != nil {
		t.Errorf("file could not be loaded")
	}

	if len(stream) != 165 {
		t.Errorf("read configuration does not match written one")
	}
}
