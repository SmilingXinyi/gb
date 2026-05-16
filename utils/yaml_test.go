package utils

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Config struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Port    int    `yaml:"port"`
}

func TestLoad(t *testing.T) {
	yamlData := `
name: test-app
version: 1.0.0
port: 8080
`
	var config Config
	reader := strings.NewReader(yamlData)
	err := Load(reader, &config)

	assert.NoError(t, err)
	assert.Equal(t, "test-app", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, 8080, config.Port)
}

func TestLoadFile(t *testing.T) {
	yamlData := `
name: file-app
version: 2.0.0
port: 9090
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(yamlData)
	assert.NoError(t, err)
	tmpFile.Close()

	var config Config
	err = LoadFile(tmpFile.Name(), &config)

	assert.NoError(t, err)
	assert.Equal(t, "file-app", config.Name)
	assert.Equal(t, "2.0.0", config.Version)
	assert.Equal(t, 9090, config.Port)
}
