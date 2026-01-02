# Docker Image Save

![logo](./logo.png)

[![Go](https://github.com/jadolg/DockerImageSave/actions/workflows/go.yml/badge.svg)](https://github.com/jadolg/DockerImageSave/actions/workflows/go.yml)

## Notice on version 1.x.x

Version 1.x.x is deprecated and will not receive updates or security patches. Please upgrade to version 2.x.x.
The default service is also now running version 2.x.x which means the old client application does no longer works.
Version 2.x.x does not need any client application since it works over HTTP(s).

## Why?

Cuba is actively blocked by Docker and this makes difficult to obtain docker images since there is no direct access to
the registry, also Cuba's internet access is restricted and slow in most cases, so a way to download these images that
can be resumed is needed by thousands of developers.

## Official Docker image

Docker image is being deployed with the CI as `guamulo/dockerimagesave`

## My instance

You can use my public instance at: [https://dockerimagesave.akiel.dev](https://dockerimagesave.akiel.dev)

Metrics available in [Grafana](https://grafana.akiel.dev/d/HU5bfRRnz/dockerimagesave?orgId=2)

Uptime monitor at [Uptime](https://uptime.akiel.dev/status/dockerimagesave)

## Usage

### Server side

#### docker-compose.yml

This will spawn a dockerimagesave server with caddy as a reverse proxy with automatic https using let's encrypt.
Remember to update the domain name in the Caddyfile.

`docker compose up -d`

#### docker run (direct usage without reverse proxy)

`docker run -v $PWD/config.yaml:/config.yaml -p 8080:8080 -d guamulo/dockerimagesave`

### Client side

#### Only get the file

`wget -c --tries=5 --waitretry=3 --content-disposition "https://dockerimagesave.akiel.dev/image?name=ubuntu:25.04"`

#### Direct pipe (simple)

```bash
wget --tries=5 --waitretry=3 -q -O - "https://dockerimagesave.akiel.dev/image?name=ubuntu:25.04" | docker load
```

#### With resume support (for large images or if you want to keep the file)

```bash
wget -c --tries=5 --waitretry=3 --content-disposition "https://dockerimagesave.akiel.dev/image?name=ubuntu:25.04" && docker load -i ubuntu_25_04.tar
```
