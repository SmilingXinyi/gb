package utils

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads YAML data from an io.Reader and unmarshals it into the out interface.
func Load(reader io.Reader, out interface{}) error {
	decoder := yaml.NewDecoder(reader)
	return decoder.Decode(out)
}

// LoadFile reads a YAML file from the given path and unmarshals it into the out interface.
func LoadFile(path string, out interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return Load(file, out)
}
