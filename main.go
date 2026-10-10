package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func printBanner() {
	banner := `
  _____             _           _____                             _____
 |  __ \           | |         |_   _|                           / ____|
 | |  | | ___   ___| | _____ _ __| |  _ __ ___   __ _  __ _  ___| (___   __ ___   _____
 | |  | |/ _ \ / __| |/ / _ \ '__| | | '_ ' _ \ / _' |/ _' |/ _ \\___ \ / _' \ \ / / _ \
 | |__| | (_) | (__|   <  __/ | _| |_| | | | | | (_| | (_| |  __/____) | (_| |\ V /  __/
 |_____/ \___/ \___|_|\_\___|_||_____|_| |_| |_|\__,_|\__, |\___|_____/ \__,_| \_/ \___|
         for Cuban developers, by Cuban developers     __/ |
                                                      |___/
	`
	fmt.Println(banner)
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to YAML configuration file")
	healthcheck := flag.Bool("healthcheck", false, "Call the local server health endpoint")
	flag.Parse()

	var addr string
	var cacheDir string
	var maxCacheAge time.Duration
	port := 8080

	config, err := LoadConfig(*configPath)
	if err != nil {
		log.WithError(err).Warn("No config file loaded, using defaults")
		addr = fmt.Sprintf(":%d", port)
		cacheDir = ""
	} else {
		port = config.Port
		addr = fmt.Sprintf(":%d", config.Port)
		cacheDir = config.CacheDir
		config.ApplyCredentials()
		maxCacheAge = config.MaxCacheAge

		log.WithField("path", *configPath).Debug("Loaded configuration")
	}

	if *healthcheck {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

	log.WithFields(log.Fields{
		"cache_dir": cacheDir,
		"max_age":   maxCacheAge,
	}).Info("Using cache directory")

	printBanner()

	server := NewServer(addr, cacheDir, maxCacheAge)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := server.Start(ctx)
	if err != nil {
		log.WithError(err).Fatal("Failed to start server")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.WithField("signal", sig).Info("Received signal, shutting down gracefully")

	// Cancel the context for background tasks
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Fatal("Server forced to shutdown")
	}

	log.Info("Server stopped")
}
