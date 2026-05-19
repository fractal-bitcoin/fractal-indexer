FROM golang:1.26.1-alpine AS build
ARG GO_OS="linux"
ARG GO_ARCH="amd64"

# see: https://stackoverflow.com/questions/42500973/compiling-go-library-without-gco-to-run-on-alpine-error-in-libczmq
RUN apk add --no-cache build-base util-linux-dev

WORKDIR /usr/local/build/
COPY ./go.mod .
COPY ./go.sum .
COPY ./api/lib/brc20_swap/go.mod ./api/lib/brc20_swap/
RUN GOOS=${GO_OS} GOARCH=${GO_ARCH} go mod download

COPY . .

# Build binary output
RUN GOOS=${GO_OS} GOARCH=${GO_ARCH} go build -o fractal-indexer -ldflags '-s -w' main.go

FROM alpine:3.18.4
RUN apk add tzdata && cp /usr/share/zoneinfo/Asia/Hong_Kong /etc/localtime && echo "Asia/Hong_Kong" > /etc/timezone
RUN apk add --no-cache libsodium

RUN adduser -u 1000 -D sato -h /data
USER sato
WORKDIR /data/

COPY --chown=sato --from=build /usr/local/build/fractal-indexer /data/fractal-indexer

ENTRYPOINT ["./fractal-indexer"]
