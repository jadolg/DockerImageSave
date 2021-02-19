package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/InVisionApp/tabular"
	"github.com/briandowns/spinner"
	"github.com/kyokomi/emoji"
)

// ServiceURL URL of the service with trailing /
var ServiceURL = "https://dockerimagesave.copincha.org/"
var showAnimations = false

func startSpinner(message string) *spinner.Spinner {
	if showAnimations {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " " + message
		err := s.Color("magenta")
		if err != nil {
			log.Print(err)
		}
		s.Start()
		time.Sleep(4 * time.Second)
		return s
	}
	return nil
}

func stopSpinner(s *spinner.Spinner, message string) {
	if showAnimations {
		s.Stop()
		_, _ = emoji.Println(":ok: " + message)
	}
}

func printBanner() {
	banner := `
 ___            _               _                         ___                   _              _           
| . \ ___  ___ | |__ ___  _ _  | |._ _ _  ___  ___  ___  | . \ ___  _ _ _ ._ _ | | ___  ___  _| | ___  _ _ 
| | |/ . \/ | '| / // ._>| '_> | || ' ' |<_> |/ . |/ ._> | | |/ . \| | | || ' || |/ . \<_> |/ . |/ ._>| '_>
|___/\___/\_|_.|_\_\\___.|_|   |_||_|_|_|<___|\_. |\___. |___/\___/|__/_/ |_|_||_|\___/<___|\___|\___.|_|  
                                              <___'    by Cuban developers for Cuban developers
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
	image := flag.String("i", "", "Image to download")
	server := flag.String("s", ServiceURL, "URL of the Docker Image Download Server")
	noAnimations := flag.Bool("no-animations", false, "Hide animations and decorations")
	noDownload := flag.Bool("no-download", false, "Do all the work but downloading the image")
	search := flag.String("search", "", "A search query")

	flag.Parse()

	showAnimations = !*noAnimations

	if showAnimations {
		printBanner()
	}

	if *server != ServiceURL {
		if strings.HasSuffix(*server, "/") {
			ServiceURL = *server
		} else {
			ServiceURL = *server + "/"
		}
	}
	fmt.Println("Using server: " + ServiceURL)

	if *search != "" {
		log.Printf("Searching for '%s'", *search)
		searchResponse, err := SearchRequest(*search)
		if err != nil {
			log.Fatal(err)
		}

		tab := tabular.New()
		tab.Col("name", "Name", 50)
		tab.Col("description", "Description", 100)
		tab.Col("stars", "Stars", 5)
		tab.Col("official", "Official", 5)
		format := tab.Print("name", "description", "stars", "official")
		for _, x := range searchResponse.SearchResult {
			fmt.Printf(format, x.Name, x.Description, x.StarCount, x.IsOfficial)
		}

		fmt.Println()

		os.Exit(0)
	}

	if *image == "" {
		fmt.Println("You must specify an image to download.\nUse -h to see application details.")
		os.Exit(1)
	}

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

	spinner = startSpinner("Saving image")
	savedImage, url := saveImage(imageName)
	for !savedImage {
		fmt.Println("Retrying...")
		time.Sleep(time.Second * 3)
		savedImage, url = saveImage(imageName)
	}

	stopSpinner(spinner, "Image saved and compressed on remote host")

	fmt.Println(ServiceURL + url)
	if !*noDownload {
		download := downloadFile(ServiceURL+url, showAnimations)
		for !download {
			fmt.Println("Retrying download...")
			download = downloadFile(ServiceURL+url, showAnimations)
		}
	}
}
