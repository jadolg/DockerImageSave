# Docker Image Save

![logo](./logo.png)

[![Go](https://github.com/jadolg/DockerImageSave/actions/workflows/go.yml/badge.svg)](https://github.com/jadolg/DockerImageSave/actions/workflows/go.yml)

## Why?

Cuba is actively blocked by Docker and this makes difficult to obtain docker images since there is no direct access to
the registry, also Cuba's internet access is restricted and slow in most cases, so a way to download these images that
can be resumed is needed by thousands of developers.

## Official Docker image

Docker image is being deployed with the CI as `guamulo/dockerimagesave`

## Usage

### Server side

#### docker-compose.yml

This will spawn a dockerimagesave server with caddy as a reverse proxy with automatic https using let's encrypt.
Remember to update the domain name in the Caddyfile.

`docker compose up -d`

#### docker run (direct usage without reverse proxy)

`docker run -v $PWD/config.yaml:/config.yaml -p 8080:8080 -d guamulo/dockerimagesave`

### Client side

#### Direct pipe (simple)

```bash
wget --tries=5 --waitretry=3 -q -O - "http://localhost:8080/image?name=ubuntu:25.04" | docker load
```

#### With resume support (for large images)

```bash
wget -c --tries=5 --waitretry=3 --content-disposition "http://localhost:8080/image?name=ubuntu:25.04" && docker load -i ubuntu_25_04.tar
```
