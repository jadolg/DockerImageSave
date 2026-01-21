package main

import (
	"flag"
	"fmt"
	"log"
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
		log.Println("No config file found, using defaults (port: 8080)")
	}

	server := NewServer(addr, cacheDir)
	log.Fatal(server.Run())
}
