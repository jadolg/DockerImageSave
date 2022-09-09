package dockerimagesave

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	docker "github.com/fsouza/go-dockerclient"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// PullImage pulls a docker image to local Docker
func PullImage(imageid string) error {
	ctx := context.Background()
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	defer func(dockerClient *client.Client) {
		err := dockerClient.Close()
		if err != nil {
			log.Print(err)
		}
	}(dockerClient)
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
	defer func(out io.ReadCloser) {
		err := out.Close()
		if err != nil {
			log.Print(err)
		}
	}(out)
	_, err = io.Copy(os.Stdout, out)
	if err != nil {
		return err
	}
	return nil
}

// SaveImage saves a docker image as tar file on specified folder
func SaveImage(imageid string, folder string) error {
	ctx := context.Background()
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	defer func(dockerClient *client.Client) {
		err := dockerClient.Close()
		if err != nil {
			log.Print(err)
		}
	}(dockerClient)
	if err != nil {
		return err
	}
	imageFileName := strings.ReplaceAll(imageid, "/", "_")
	imageFileName = strings.Replace(imageFileName, ":", "_", 1)
	imageFileName = RemoveDoubleDots(imageFileName)
	f, err := os.Create(folder + "/" + imageFileName + ".tar")
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Print(err)
		}
	}(f)
	w := bufio.NewWriter(f)
	data, err := dockerClient.ImageSave(ctx, []string{imageid})
	if err != nil {
		return err
	}
	_, err = io.Copy(w, data)
	if err != nil {
		return err
	}
	err = w.Flush()
	if err != nil {
		return err
	}
	return nil
}

// ImageExists checks if image is downloaded
func ImageExists(imageid string) (bool, error) {
	ctx := context.Background()
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	defer func(dockerClient *client.Client) {
		err := dockerClient.Close()
		if err != nil {
			log.Print(err)
		}
	}(dockerClient)
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
	if !strings.Contains(imageid, "/") {
		imageAndTag[0] = fmt.Sprintf("library/%s", imageAndTag[0])
	}
	resp, err := http.Get(fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags/%s/", imageAndTag[0], imageAndTag[1]))
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Print(err)
		}
	}(resp.Body)
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	b, _ := io.ReadAll(resp.Body)
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
