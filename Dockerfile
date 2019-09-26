FROM golang:1.12-buster

RUN apt update
RUN apt install -y git
RUN go get -u github.com/golang/dep/...

COPY . /go/src/github.com/jadolg/DockerImageSave/
WORKDIR /go/src/github.com/jadolg/DockerImageSave/

RUN dep ensure

RUN go build github.com/jadolg/DockerImageSave/cmd/DockerImageSaveServer
RUN /bin/bash build_executables.sh

CMD [ "./DockerImageSaveServer" ]
