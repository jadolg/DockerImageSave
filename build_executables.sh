#! /bin/sh
CGO_ENABLED=0

function buildExecutable() {
    GOOS=${1}
    GOARCH=${2}
    if [[ ${GOOS} == "windows" ]]; then
        EXTENSION=".exe"
    else
        EXTENSION=""
    fi
    go build -o /executables/DockerImageSave-${GOOS}-${GOARCH}${EXTENSION} /go/src/github.com/jadolg/DockerImageSave/cmd/DockerImageSave

    if [[ $? -ne 0 ]]; then
        echo An error has occurred building executable for ${GOOS}/${GOARCH}
        exit 1
    fi
    echo Executable built for ${GOOS}/${GOARCH}
}

mkdir -p /executables/

buildExecutable linux amd64
buildExecutable darwin amd64
buildExecutable windows amd64
