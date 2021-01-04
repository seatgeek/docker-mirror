FROM golang:1.10-alpine
# Adding ca-certificates for external communication and git for dep installation
RUN apk add --update ca-certificates git \
    && rm -rf /var/cache/apk/*
RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR /go/src/github.com/seatgeek/docker-mirror/
COPY . /go/src/github.com/seatgeek/docker-mirror/
RUN dep ensure -vendor-only
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/docker-mirror  .

FROM 093535234988.dkr.ecr.us-east-1.amazonaws.com/docker-base:5.1.0
COPY --from=0 /go/src/github.com/seatgeek/docker-mirror/build/docker-mirror /usr/local/bin/
CMD ["docker-mirror"]
