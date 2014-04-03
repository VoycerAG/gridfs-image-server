package server

import (
	"errors"
	"net/http"
)

// ServerConfiguration is a wrapper object for request parameters.
type ServerConfiguration struct {
	Database   string
	FormatName string
	Filename   string
}

// createConfigurationFromVars validate all necessary request parameters
func CreateConfigurationFromVars(r *http.Request, vars map[string]string) (*ServerConfiguration, error) {
	config := ServerConfiguration{}

	database := vars["database"]

	if database == "" {
		return nil, errors.New("database must not be empty")
	}

	filename := vars["filename"]

	if filename == "" {
		return nil, errors.New("filename must not be empty")
	}

	formatName := r.URL.Query().Get("size")

	config.Database = database
	config.FormatName = formatName
	config.Filename = filename

	return &config, nil
}
