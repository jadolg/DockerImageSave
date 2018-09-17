package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
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

func pullImage(imageName string) bool {
	pullImage, err := PullImageRequest(imageName)

	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	for pullImage.Status != "Downloaded" {
		if pullImage.Status == "Error" {
			fmt.Println("\nCan not pull image")
			return false
		}
		pullImage, err = PullImageRequest(imageName)
		if err != nil {
			fmt.Println(err.Error())
			return false
		}
		time.Sleep(time.Second * 5)
	}
	return true
}

func saveImage(imageName string) (bool, string) {
	saveImage, err := SaveImageRequest(imageName)
	if err != nil {
		fmt.Println(err.Error())
		return false, ""
	}

	for saveImage.Status != "Ready" {
		saveImage, err = SaveImageRequest(imageName)
		if err != nil {
			fmt.Println(err.Error())
			return false, ""
		}
		time.Sleep(time.Second * 5)
	}
	return true, saveImage.URL
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
	if match, _ := regexp.MatchString("(.*/)?.+:.+", imageName); !match || strings.Count(imageName, "/") > 1 {
		fmt.Printf("%s is not a valid image name. Use image:tag or user/image:tag\nOnly DockerHub images supported so far.\n", imageName)
		os.Exit(1)
	}
	fmt.Println("Downloading image: " + imageName)

	spinner := startSpinner("Downloading image")
	pulledImage := pullImage(imageName)
	for !pulledImage {
		fmt.Println("Retrying...")
		time.Sleep(time.Second * 3)
		pulledImage = pullImage(imageName)
	}
	stopSpinner(spinner, "Image downloaded on remote host")

	savedImage, url := saveImage(imageName)
	for !savedImage {
		fmt.Println("Retrying...")
		time.Sleep(time.Second * 3)
		savedImage, url = saveImage(imageName)
	}
	spinner = startSpinner("Saving image")

	stopSpinner(spinner, "Image saved and compressed on remote host")

	download := downloadFile(ServiceURL + url)
	for !download {
		fmt.Println("Retrying download...")
		download = downloadFile(ServiceURL + url)
	}
}
