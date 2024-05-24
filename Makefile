GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOARCH=$(shell go env GOARCH)
GOOS=$(shell go env GOOS )
VERSION=$(shell cat VERSION)

GoVersion=$(shell go version)
BASEPATH=$(shell pwd)
BUILD_TIME=$(shell date +"%Y%m%d%H%M%S")
BUILDDIR=$(BASEPATH)/dist
DASHBOARDDIR=$(BASEPATH)/web/dashboard
MAIN=$(BASEPATH)/cmd/main.go
APPVERSION=$(VERSION)

APP_NAME=kubackup-server-$(GOOS)-$(GOARCH)-$(APPVERSION)

LDFLAGS=-ldflags "-s -w -X backup.GitTag=${GITVERSION} -X backup.BuildTime=${BUILD_TIME} -X backup.V=${VERSION}"

# 构建 web dashboard
build_web_dashboard:
	cd $(DASHBOARDDIR) && npm config set registry https://registry.npm.taobao.org && npm install && npm run build:prod

build_go:
	go mod download
	go mod tidy
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -trimpath $(LDFLAGS) -o $(BUILDDIR)/$(APP_NAME) $(MAIN)

# 构建二进制文件和 web dashboard
build_bin: build_web_dashboard build_go

# 构建 Docker 镜像到私库
build_image:
	docker buildx build -t kubackup/kubackup:${VERSION} --platform=linux/arm64,linux/amd64,darwin/amd64,darwin/arm64 . --push

