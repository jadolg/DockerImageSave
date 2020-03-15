# Docker Image Save
This package consists of two commands. 
* `DockerImageSaveServer` is a server that pulls, saves, compresses and serves via http docker images. Docker has to be installed in the computer this service is deployed. 
* `DockerImageSave` is the terminal application that talks with `DockerImageSaveServer` and downloads the zip compressed docker images.

## Why?
Cuba is actively blocked by Docker and this makes difficult to obtain docker images since there is no direct access to the registry, also Cuba's internet access is restricted and slow in most cases, so a way to download this images that can be resumed is needed by thousands of developers.

## Official Docker image
Docker image is being deployed with the CI as `guamulo/dockerimagesave`

## API Documentation

Want to write yor own client? Here is what you need to know.

### Pulling the image on the server

This API call will pull the image to your server. It is exactly the same as doing `docker pull image` on the server.
- path: **/pull/{id}**
- method: GET
- curl: `curl https://docker-image-save.aleph.engineering/pull/alpine:latest`
- response: `{"id":"alpine:latest","status":"Downloaded"}`

Wait for the status to be **Downloaded** so you can save the image for downloading.
It does not matter how many times you call this endpoint with the same image it won't re-download it.

### Saving the image

This API call will save and compress an already pulled image making it ready for download. 
- path: **/save/{id}**
- method: GET
- curl: `curl https://docker-image-save.aleph.engineering/save/alpine:latest`
- response: `{"id":"alpine:latest","url":"download/alpine:latest.tar.zip","size":2214576,"status":"Ready"}`

Wait for the status to be **Ready** so you can download.

### Downloading the image

Finally you can download the image doing with the url provided after saving it.

- Path: **/download/{url}**
- Method: GET
- curl: `url https://docker-image-save.aleph.engineering/download/alpine:latest.tar.zip -o alpine:latest.tar.zip`

### Loading the image into your local Docker
- Unzip the downloaded file `unzip alpine\:latest.tar.zip`
- Load it into Docker `docker load -i alpine\:latest.tar`
Now you should be able to see it on the images list on `docker images | grep alpine`
