FROM golang:1.27.2 AS build

WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags '-w -s' .

FROM scratch
COPY --from=build /app/DockerImageSave /DockerImageSave
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=30s --start-period=1s --retries=3 CMD [ "/DockerImageSave", "--healthcheck" ]

CMD [ "/DockerImageSave" ]
