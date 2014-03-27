package config

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	AllowedEntries []Entry `json:allowedEntries`
}

type Entry struct {
	Name   string `json:name`
	Width  int64  `json:width`
	Height int64  `json:height`
}

// CreateConfigFromFile returns an Config object from a given file
func CreateConfigFromFile(file string) (*Config, error) {
	result := Config{}

	config, err := ioutil.ReadFile(file)

	if err != nil {
		return &result, err
	}

	err = json.Unmarshal(config, &result)

	return &result, err
}
