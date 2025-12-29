package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to YAML configuration file")
	flag.Parse()

	var addr string
	var cacheDir string

	if *configPath != "" {
		config, err := LoadConfig(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		addr = fmt.Sprintf(":%d", config.Port)
		cacheDir = config.CacheDir
		config.ApplyCredentials()

		log.Printf("Loaded configuration from %s", *configPath)
	} else {
		addr = ":8080"
		cacheDir = ""
	}

	server := NewServer(addr, cacheDir)
	log.Fatal(server.Run())
}
