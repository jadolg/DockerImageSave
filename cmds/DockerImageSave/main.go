package main

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/kyokomi/emoji"
)

// ServiceURL URL of the service with trailing /
const ServiceURL = "http://localhost:6060/"

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
	if len(os.Args) < 2 {
		fmt.Println("Not enough arguments. Please specify image name.")
		os.Exit(1)
	}

	fmt.Println("Using server: " + ServiceURL)
	fmt.Println("Downloading image: " + ServiceURL)

	imageName := os.Args[1]

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

	downloadFile(ServiceURL + saveImage.URL)
}
