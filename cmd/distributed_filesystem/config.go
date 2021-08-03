package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/basharal/filesystem/client"
)

// Conf represents a configuration
type Conf struct {
	Servers []client.Server `json:"servers"`
}

// Parse parses the config file
func Parse(path string) (*Conf, error) {
	c := &Conf{}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}
