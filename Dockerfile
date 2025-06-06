FROM golang:1.24

COPY . /go/src/github.com/jadolg/DockerImageSave/
WORKDIR /go/src/github.com/jadolg/DockerImageSave/

RUN CGO_ENABLED=0 go build -ldflags '-w -s' -a -installsuffix cgo github.com/jadolg/DockerImageSave/cmd/DockerImageSaveServer

FROM alpine:3.22
COPY --from=0 /go/src/github.com/jadolg/DockerImageSave/DockerImageSaveServer /executables/DockerImageSaveServer
WORKDIR /executables/
CMD [ "./DockerImageSaveServer" ]
