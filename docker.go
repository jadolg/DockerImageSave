package dockerimagesave

import (
	"bufio"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

// PullImage pulls a docker image to local Docker
func PullImage(imageid string) error {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		return err
	}
	opts := docker.PullImageOptions{Repository: imageid}
	err = dockerClient.PullImage(opts, docker.AuthConfiguration{})
	if err != nil {
		return err
	}
	return nil
}

// SaveImage saves a docker image as tar file on specified folder
func SaveImage(imageid string, folder string) error {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		return err
	}
	imageFileName := strings.ReplaceAll(imageid, "/", "_")
	imageFileName = strings.Replace(imageFileName, ":", "_", 1)
	f, err := os.Create(folder + "/" + imageFileName + ".tar")
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	opts := docker.ExportImagesOptions{Names: []string{imageid}, OutputStream: w}
	if err := dockerClient.ExportImages(opts); err != nil {
		return err
	}
	w.Flush()
	return nil
}

// ImageExists checks if image is downloaded
func ImageExists(imageid string) (bool, error) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		return false, err
	}

	imgs, err := dockerClient.ListImages(docker.ListImagesOptions{Filter: imageid})

	if err != nil {
		return false, err
	}

	return len(imgs) > 0, nil
}

// ImageExistsInRegistry determines if an image exists in the docker registry
func ImageExistsInRegistry(imageid string) (bool, error) {
	if !strings.Contains(imageid, ":") {
		return false, errors.New("The use of a Tag is obligatory")
	}
	imageAndTag := strings.Split(imageid, ":")
	resp, err := http.Get("https://index.docker.io/v1/repositories/" + imageAndTag[0] + "/tags/" + imageAndTag[1])
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	return string(b) != "\"Resource not found\"" && string(b) != "Tag not found", nil
}

// Search does a docker search
func Search(term string) ([]docker.APIImageSearch, error) {
	dockerClient, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	imageSearch, err := dockerClient.SearchImages(term)
	if err != nil {
		return nil, err
	}
	return imageSearch, nil
}
