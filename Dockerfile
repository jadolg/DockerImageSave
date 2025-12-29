FROM golang:1.25 AS build

WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-w -s' .

FROM scratch
COPY --from=build /app/DockerImageSave /DockerImageSave
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080

CMD [ "/DockerImageSave" ]
