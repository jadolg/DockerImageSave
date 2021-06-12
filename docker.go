package dockerimagesave

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	docker "github.com/fsouza/go-dockerclient"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// PullImage pulls a docker image to local Docker
func PullImage(imageid string) error {
	ctx := context.Background()
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	authConfig := types.AuthConfig{
		Username: os.Getenv("DOCKER_USER"),
		Password: os.Getenv("DOCKER_PASSWORD"),
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	out, err := dockerClient.ImagePull(ctx, imageid, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		return err
	}
	defer out.Close()
	io.Copy(os.Stdout, out)
	return nil
}

// SaveImage saves a docker image as tar file on specified folder
func SaveImage(imageid string, folder string) error {
	ctx := context.Background()
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
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
	data, err := dockerClient.ImageSave(ctx, []string{imageid})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, data)
	if err != nil {
		return err
	}
	w.Flush()
	return nil
}

// ImageExists checks if image is downloaded
func ImageExists(imageid string) (bool, error) {
	ctx := context.Background()
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false, err
	}
	imgs, err := dockerClient.ImageList(ctx, types.ImageListOptions{
		All:     false,
		Filters: filters.Args{},
	})

	if err != nil {
		return false, err
	}

	for _, img := range imgs {
		for _, repotag := range img.RepoTags {
			if repotag == imageid {
				return true, nil
			}
		}
	}

	return false, nil
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
