# Docker Image Save
This package consists of two commands. 
* `DockerImageSaveServer` is a server that pulls, saves, compresses and serves via http docker images. Docker has to be installed in the computer this service is deployed. 
* `DockerImageSave` is the terminal application that talks with `DockerImageSaveServer` and downloads the zip compressed docker images.

## Why?
Cuba is actively blocked by Docker and this makes difficult to obtain docker images since there is no direct access to the registry, also Cuba's internet access is restricted and slow in most cases, so a way to download these images that can be resumed is needed by thousands of developers.

## Official Docker image
Docker image is being deployed with the CI as `guamulo/dockerimagesave`

## How to use the client:

### Download

Download it for your distribution from the releases page on GitHub (https://github.com/jadolg/DockerImageSave/releases).
Not there? Create an issue and I'll start shipping specially for you ;-)

or

### Install as a snap (https://snapcraft.io/docker-image-save)

`snap install docker-image-save` 

### Help

The client comes with help included. Please use it ;-)

```
Usage of ./DockerImageSave-linux-amd64:
  -i string
        Image to download
  -no-animations
        Hide animations and decorations
  -no-download
        Do all the work but downloading the image
  -s string
        URL of the Docker Image Download Server (default "https://dockerimagesave.copincha.org/")
  -search string
        A search query
```

If you are using it from a script I recommend to use the `-no-animations` flag to make it less noisy.
Also if planning to use curl for downloading you might want to use the `-no-download` flag.

## How to use the server

You are able and encouraged to deploy your own server. The best way to use it is of course the docker version.
Just run `docker run -p 6060:6060 -v /var/run/docker.sock:/var/run/docker.sock:rw -d guamulo/dockerimagesave:latest`.
There are no "official" builds of the server as a binary.

## API Documentation

Want to write yor own client? Here is what you need to know.

### Pulling the image on the server

This API call will pull the image to your server. It is exactly the same as doing `docker pull image` on the server.
- path: **/pull/{id}**
- method: GET
- curl: `curl https://dockerimagesave.copincha.org/pull/alpine:latest`
- response: `{"id":"alpine:latest","status":"Downloaded"}`

Wait for the status to be **Downloaded** so you can save the image for downloading.
It does not matter how many times you call this endpoint with the same image it won't re-download it.

### Saving the image

This API call will save and compress an already pulled image making it ready for download. 
- path: **/save/{id}**
- method: GET
- curl: `curl https://dockerimagesave.copincha.org/save/alpine:latest`
- response: `{"id":"alpine:latest","url":"download/alpine:latest.tar.zip","size":2214576,"status":"Ready"}`

Wait for the status to be **Ready** so you can download.

### Downloading the image

Finally you can download the image doing with the url provided after saving it.

- Path: **/download/{url}**
- Method: GET
- curl: `url https://dockerimagesave.copincha.org/download/alpine:latest.tar.zip -o alpine:latest.tar.zip`

### Loading the image into your local Docker
- Unzip the downloaded file `unzip alpine\:latest.tar.zip`
- Load it into Docker `docker load -i alpine\:latest.tar`
Now you should be able to see it on the images list on `docker images | grep alpine`
