FROM golang:1.11.0-alpine3.8

RUN apk update
RUN apk add git
RUN go get -u github.com/golang/dep/...

COPY . /go/src/github.com/jadolg/DockerImageSave/
WORKDIR /go/src/github.com/jadolg/DockerImageSave/

RUN dep ensure

RUN go build github.com/jadolg/DockerImageSave/cmd/DockerImageSaveServer
RUN go build github.com/jadolg/DockerImageSave/cmd/DockerImageSave
RUN GOOS=windows go build github.com/jadolg/DockerImageSave/cmd/DockerImageSave

ENTRYPOINT [ "./DockerImageSaveServer" ]
