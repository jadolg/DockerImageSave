FROM golang:1.12-buster

ENV GO111MODULE=on
COPY . /go/src/github.com/jadolg/DockerImageSave/
WORKDIR /go/src/github.com/jadolg/DockerImageSave/

RUN go build github.com/jadolg/DockerImageSave/cmd/DockerImageSaveServer
RUN /bin/bash build_executables.sh

CMD [ "./DockerImageSaveServer" ]
