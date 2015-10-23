package server

import (
	"errors"
	"net/http"
)

// Configuration is a wrapper object for request parameters.
type Configuration struct {
	Database   string
	FormatName string
	Filename   string
}

// CreateConfigurationFromVars validate all necessary request parameters
func CreateConfigurationFromVars(r *http.Request, vars map[string]string) (*Configuration, error) {

	database := vars["database"]

	if database == "" {
		return nil, errors.New("database must not be empty")
	}

	filename := vars["filename"]

	if filename == "" {
		return nil, errors.New("filename must not be empty")
	}

	formatName := r.URL.Query().Get("size")

	return &Configuration{Database: database, FormatName: formatName, Filename: filename}, nil
}
