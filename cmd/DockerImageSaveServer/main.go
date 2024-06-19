package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var downloadsFolder = "/tmp"

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
	printBanner()
	folder := flag.String("folder", "/tmp", "Folder to save the docker images")
	port := flag.String("port", "6060", "port to be used by the service")
	flag.Parse()

	downloadsFolder = *folder

	router := mux.NewRouter()
	router.HandleFunc("/pull/{id}", PullImageHandler).Methods("GET")
	router.HandleFunc("/search", SearchHandler).Methods("GET")
	router.HandleFunc("/save/{id}", SaveImageHandler).Methods("GET")
	router.HandleFunc("/pull/{user}/{id}", PullImageHandler).Methods("GET")
	router.HandleFunc("/save/{user}/{id}", SaveImageHandler).Methods("GET")
	router.PathPrefix("/download/").Handler(http.StripPrefix("/download/",
		handlers.CombinedLoggingHandler(log.Writer(), http.FileServer(http.Dir(downloadsFolder)))))
	router.HandleFunc("/healthcheck", HealthCheckHandler).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	fmt.Println("Listening on port " + *port)
	fmt.Println("Downloading files on " + downloadsFolder)
	log.Fatal(http.ListenAndServe(":"+*port, router))
}
