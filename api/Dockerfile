FROM golang:1.22-alpine AS build
ARG GO_OS="linux"
ARG GO_ARCH="amd64"
WORKDIR /usr/local/build/
COPY ./go.mod .
COPY ./go.sum .
COPY ./lib/brc20_swap/go.mod ./lib/brc20_swap/

# Build binary output
# RUN GOPROXY=https://goproxy.cn,direct GOOS=${GO_OS} GOARCH=${GO_ARCH} go get -u github.com/swaggo/swag/cmd/swag@v1.6.7
# RUN GOPROXY=https://goproxy.cn,direct GOOS=${GO_OS} GOARCH=${GO_ARCH} swag init

# RUN GOPROXY=https://goproxy.cn,direct GOOS=${GO_OS} GOARCH=${GO_ARCH} go mod download

RUN GOOS=${GO_OS} GOARCH=${GO_ARCH} go mod download

COPY . .

RUN GOOS=${GO_OS} GOARCH=${GO_ARCH} go build -o fractal-query -ldflags '-s -w' main.go

FROM alpine:3.18.4
RUN apk add tzdata && cp /usr/share/zoneinfo/Asia/Hong_Kong /etc/localtime && echo "Asia/Hong_Kong" > /etc/timezone

RUN adduser -u 1000 -D sato -h /data
USER sato
WORKDIR /data/

COPY --chown=sato --from=build /usr/local/build/fractal-query /data/fractal-query
COPY --chown=sato --from=build /usr/local/build/docs /data/docs

ENV LISTEN 0.0.0.0:8000
EXPOSE 8000
CMD ["./fractal-query"]
