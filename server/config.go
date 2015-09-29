package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Config contains entries for
// possible image configurations
type Config struct {
	AllowedEntries []Entry `json:allowedEntries`
}

const (
	// TypeResize will either force the given sizes, or resize via original ratio when either height or width is not specified
	TypeResize = "resize"
	// TypeCrop will generate an image with exact sizes, but only a part of the image is visible
	TypeCrop = "crop"
	//TypeFit will resize the image according to the original ratio, but will not exceed the given bounds
	TypeFit = "fit"
)

// Entry is one allowed image configuration
type Entry struct {
	Name   string `json:name`
	Width  int64  `json:width`
	Height int64  `json:height`
	Type   string `json:type`
}

//NewConfigFromBytes generates a new config object by a byte stream
func NewConfigFromBytes(b []byte) (*Config, error) {
	result := Config{}
	err := json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}

	err = result.validateConfig()

	return &result, err
}

// NewConfigFromFile returns an Config object from a given file.
func NewConfigFromFile(file string) (*Config, error) {
	config, err := ioutil.ReadFile(file)

	if err != nil {
		return nil, err
	}

	return NewConfigFromBytes(config)
}

// validateConfig validates the configuration and fills elements with default types.
func (config *Config) validateConfig() error {
	for index, element := range config.AllowedEntries {
		if element.Width <= 0 && element.Height <= 0 {
			return fmt.Errorf("The width and height of the configuration element with name \"%s\" are invalid.", element.Name)
		}

		if element.Type == "" {
			config.AllowedEntries[index].Type = TypeResize
			continue
		}

		if element.Type != TypeResize && element.Type != TypeCrop && element.Type != TypeFit {
			return fmt.Errorf("Type must be either %s, %s or %s at element \"%s\"", TypeCrop, TypeResize, TypeFit, element.Name)
		}
	}

	return nil
}

// GetEntryByName Returns an entry the the name.
func (config *Config) GetEntryByName(name string) (*Entry, error) {
	for _, element := range config.AllowedEntries {
		if element.Name == name {
			return &element, nil
		}
	}

	return nil, fmt.Errorf("No Entry found in configuration for given name %s", name)
}
