// Package main demonstrates loading YAML config with optional dotenv and env overrides.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/SmilingXinyi/gb/config"
)

type serverConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type appConfig struct {
	Name    string          `yaml:"name"`
	Debug   bool            `yaml:"debug"`
	Timeout config.Duration `yaml:"timeout"`
	Server  serverConfig    `yaml:"server"`
}

func main() {
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	var settings appConfig
	err := config.Load(
		configPath,
		&settings,
		config.WithOptionalDotEnv(".env"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("name=%s debug=%v timeout=%s server=%s:%d\n",
		settings.Name,
		settings.Debug,
		settings.Timeout.Value(),
		settings.Server.Host,
		settings.Server.Port,
	)
}
