# Docker Image Save
This package consists of two commands. 
* `DockerImageSaveServer` is a server that pulls, saves, compresses and serves via http docker images. Docker has to be installed in the computer this service is deployed. 
* `DockerImageSave` is the terminal application that talks with `DockerImageSaveServer` and downloads the zip compressed docker images.

## Why?
Cuba is actively blocked by Docker and this makes difficult to obtain docker images since there is no direct access to the registry, also Cuba's internet access is restricted and slow in most cases, so a way to download this images that can be resumed is needed by thousands of developers.
