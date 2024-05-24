# syntax = docker/dockerfile:experimental
FROM node:14.18.2 AS buildvue

WORKDIR /dowell/
COPY . /dowell/
RUN --mount=type=cache,target=/dowell/web/dashboard/node_modules cd /dowell/web/dashboard &&\
    npm config set registry https://registry.npmmirror.com && npm install
RUN --mount=type=cache,target=/dowell/web/dashboard/node_modules cd /dowell/web/dashboard &&\
    npm run build:prod

FROM golang:1.16.15-alpine3.15 AS buildbin
ENV GO111MODULE=on
ENV GOPROXY="https://goproxy.cn,direct"
ENV CGO_ENABLED=0
ENV GOPATH=/root/gopath

WORKDIR /dowell/
COPY . /dowell/
RUN --mount=type=cache,target=/root/gopath echo -e 'https://mirrors.ustc.edu.cn/alpine/v3.15/main/\nhttps://mirrors.ustc.edu.cn/alpine/v3.15/community/' > /etc/apk/repositories &&\
    apk update &&\
    apk upgrade &&\
    apk add --no-cache git make libffi-dev openssl-dev libtool tzdata &&\
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime &&  \
    sh prepare.sh
COPY --from=buildvue /dowell /dowell
RUN --mount=type=cache,target=/root/gopath make build_go

FROM alpine:latest
LABEL MAINTAINER="kubackup <bjpoya@163.com>"
ENV LANG C.UTF-8
COPY --from=buildbin /dowell/dist/kubackup-server-* /apps/kubackup-server
COPY --from=buildbin /etc/localtime /etc/localtime

EXPOSE 8012

ENTRYPOINT ["/apps/kubackup-server"]

