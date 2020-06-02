FROM golang:1.14-buster

ENV GO111MODULE=on
COPY . /go/src/github.com/jadolg/DockerImageSave/
WORKDIR /go/src/github.com/jadolg/DockerImageSave/

RUN CGO_ENABLED=0 go build github.com/jadolg/DockerImageSave/cmd/DockerImageSaveServer
RUN /bin/bash build_executables.sh

FROM alpine:3.12
COPY --from=0 /executables/ /executables/
COPY --from=0 /go/src/github.com/jadolg/DockerImageSave/DockerImageSaveServer /executables/DockerImageSaveServer
WORKDIR /executables/
CMD [ "./DockerImageSaveServer" ]
