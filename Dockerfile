FROM golang:1.20-alpine
# Adding ca-certificates for external communication and git for dependency installation
RUN apk add --no-cache ca-certificates git
WORKDIR /go/src/github.com/seatgeek/docker-mirror/
COPY . /go/src/github.com/seatgeek/docker-mirror/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/docker-mirror  .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=0 /go/src/github.com/seatgeek/docker-mirror/build/docker-mirror /usr/local/bin/
CMD ["docker-mirror"]
