package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func printBanner() {
	banner := `
 ___            _               _                         ___                   _              _             ___                          
| . \ ___  ___ | |__ ___  _ _  | |._ _ _  ___  ___  ___  | . \ ___  _ _ _ ._ _ | | ___  ___  _| | ___  _ _  / __> ___  _ _  _ _  ___  _ _ 
| | |/ . \/ | '| / // ._>| '_> | || ' ' |<_> |/ . |/ ._> | | |/ . \| | | || ' || |/ . \<_> |/ . |/ ._>| '_> \__ \/ ._>| '_>| | |/ ._>| '_>
|___/\___/\_|_.|_\_\\___.|_|   |_||_|_|_|<___|\_. |\___. |___/\___/|__/_/ |_|_||_|\___/<___|\___|\___.|_|   <___/\___.|_|  |__/ \___.|_|  
                                              <___'    by Cuban developers for Cuban developers
	`
	fmt.Println(banner)
}

func main() {
	configPath := flag.String("config", "", "Path to YAML configuration file")
	flag.Parse()

	printBanner()

	var addr string
	var cacheDir string

	// Try to load config from specified path, or fall back to config.yaml if it exists
	configFile := *configPath
	if configFile == "" {
		// Check if default config.yaml exists
		if _, err := os.Stat("config.yaml"); err == nil {
			configFile = "config.yaml"
		}
	}

	if configFile != "" {
		config, err := LoadConfig(configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		addr = fmt.Sprintf(":%d", config.Port)
		cacheDir = config.CacheDir
		config.ApplyCredentials()

		log.Printf("Loaded configuration from %s", configFile)
	} else {
		addr = ":8080"
		cacheDir = ""
		log.Println("No config file found, using defaults (port: 8080)")
	}

	server := NewServer(addr, cacheDir)
	log.Fatal(server.Run())
}
