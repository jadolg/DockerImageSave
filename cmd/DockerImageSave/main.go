package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/kyokomi/emoji"
)

// ServiceURL URL of the service with trailing /
var ServiceURL = "http://ddnnss.eu:6060/"

func startSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	s.Color("magenta")
	s.Start()
	time.Sleep(4 * time.Second)
	return s
}

func stopSpinner(s *spinner.Spinner, message string) {
	s.Stop()
	emoji.Println(":ok: " + message)
}

func printBanner() {
	banner := `
 ___            _               _                         ___                   _              _           
| . \ ___  ___ | |__ ___  _ _  | |._ _ _  ___  ___  ___  | . \ ___  _ _ _ ._ _ | | ___  ___  _| | ___  _ _ 
| | |/ . \/ | '| / // ._>| '_> | || ' ' |<_> |/ . |/ ._> | | |/ . \| | | || ' || |/ . \<_> |/ . |/ ._>| '_>
|___/\___/\_|_.|_\_\\___.|_|   |_||_|_|_|<___|\_. |\___. |___/\___/|__/_/ |_|_||_|\___/<___|\___|\___.|_|  
                                              <___'    Sponsored by Cuban.Engineer [https://cuban.engineer]
	`
	fmt.Println(banner)
}

func main() {
	printBanner()
	image := flag.String("i", "", "Image to download")
	server := flag.String("s", ServiceURL, "URL of the Docker Image Download Server")

	flag.Parse()

	if *image == "" {
		fmt.Println("You must specify an image to download.\nUse -h to see application details.")
		os.Exit(1)
	}

	if *server != ServiceURL {
		if strings.HasSuffix(*server, "/") {
			ServiceURL = *server
		} else {
			ServiceURL = *server + "/"
		}
	}

	fmt.Println("Using server: " + ServiceURL)

	imageName := *image
	fmt.Println("Downloading image: " + imageName)

	pullImage, err := PullImageRequest(imageName)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	spinner := startSpinner("Downloading image")
	for pullImage.Status != "Downloaded" {
		if pullImage.Status == "Error" {
			fmt.Println("\nCan not pull image")
			os.Exit(1)
		}
		pullImage, err = PullImageRequest(imageName)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		time.Sleep(time.Second * 10)
	}
	stopSpinner(spinner, "Image downloaded on remote host")

	saveImage, err := SaveImageRequest(imageName)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	spinner = startSpinner("Saving image")
	for saveImage.Status != "Ready" {
		if pullImage.Status == "Error" {
			fmt.Println("\nCan not save image")
			os.Exit(1)
		}
		saveImage, err = SaveImageRequest(imageName)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		time.Sleep(time.Second * 10)
	}
	stopSpinner(spinner, "Image saved and compressed on remote host")

	download := downloadFile(ServiceURL + saveImage.URL)
	for !download {
		fmt.Println("Retrying download...")
		download = downloadFile(ServiceURL + saveImage.URL)
	}
}
