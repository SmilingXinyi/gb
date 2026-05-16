package main

import (
	"fmt"
	"log"
	"os"

	"github.com/SmilingXinyi/gb/utils"
)

type AppConfig struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		URL string `yaml:"url"`
	} `yaml:"database"`
}

func main() {
	// Create a temporary YAML file for demonstration
	yamlContent := `
server:
  host: localhost
  port: 8080
database:
  url: postgres://user:pass@localhost:5432/db
`
	filename := "config.yaml"
	err := os.WriteFile(filename, []byte(yamlContent), 0644)
	if err != nil {
		log.Fatalf("Failed to create config file: %v", err)
	}
	defer os.Remove(filename)

	// Load configuration from file
	var config AppConfig
	if err := utils.LoadFile(filename, &config); err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	fmt.Printf("Server Host: %s\n", config.Server.Host)
	fmt.Printf("Server Port: %d\n", config.Server.Port)
	fmt.Printf("Database URL: %s\n", config.Database.URL)
}
