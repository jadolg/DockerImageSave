#! /bin/bash
CGO_ENABLED=0

function buildExecutable() {
    export GOOS=${1}
    export GOARCH=${2}
    if [[ ${GOOS} == "windows" ]]; then
        EXTENSION=".exe"
    else
        EXTENSION=""
    fi
    go build -o DockerImageSave-${GOOS}-${GOARCH}${EXTENSION} github.com/jadolg/DockerImageSave/cmd/DockerImageSave

    if [[ $? -ne 0 ]]; then
        echo An error has occurred building executable for ${GOOS}/${GOARCH}
        exit 1
    fi
    echo Executable built for ${GOOS}/${GOARCH}
}

buildExecutable linux amd64
buildExecutable darwin amd64
buildExecutable windows amd64
