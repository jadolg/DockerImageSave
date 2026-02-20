package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	configPath := flag.String("config", "config.yaml", "Path to YAML configuration file")
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
	}

	server := NewServer(addr, cacheDir)
	srv, err := server.Start()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received %s, shutting down gracefully...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
