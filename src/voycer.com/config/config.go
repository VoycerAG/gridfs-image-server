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

func openFile(file string) ([]byte, error) {
	configFile, err := ioutil.ReadFile(file)

	if err != nil {
		return nil, err
	}

	return configFile, nil
}

func CreateConfigFromFile(file string) (*Config, error) {
	result := Config{}

	config, err := openFile(file)

	if err != nil {
		return &result, err
	}

	err = json.Unmarshal(config, &result)

	return &result, err
}
