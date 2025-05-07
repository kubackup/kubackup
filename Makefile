GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOARCH=$(shell go env GOARCH)
GOOS=$(shell go env GOOS )
VERSION=$(shell cat VERSION)

GoVersion=$(shell go version)
BASEPATH=$(shell pwd)
BUILD_TIME=$(shell date +"%Y%m%d%H%M")
BUILDDIR=$(BASEPATH)/dist
DASHBOARDDIR=$(BASEPATH)/web/dashboard
MAIN=$(BASEPATH)/cmd/main.go
APPVERSION=$(VERSION)

APP_NAME=kubackup_server_$(APPVERSION)_$(GOOS)_$(GOARCH)

LDFLAGS=-ldflags "-s -w -X github.com/kubackup/kubackup.BuildTime=${BUILD_TIME} -X github.com/kubackup/kubackup.V=${VERSION}"


all: build_web_dashboard all_bin
	$(BASEPATH)/hashsum.sh

all_bin: clean build_linux_amd64 build_linux_arm64 build_osx_amd64 build_osx_arm64

clean:
	rm -rf $(BUILDDIR)

# 构建 web dashboard
build_web_dashboard:
	cd $(DASHBOARDDIR) && npm config set registry https://registry.npmmirror.com && npm install && npm run build:prod

build_go:
	go mod download
	go mod tidy
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -trimpath $(LDFLAGS) -o $(BUILDDIR)/$(APP_NAME) $(MAIN)

# 构建二进制文件和 web dashboard
build_bin: build_web_dashboard clean build_go

build_linux_amd64:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -trimpath $(LDFLAGS) -o $(BUILDDIR)/kubackup_server_$(APPVERSION)_linux_amd64 $(MAIN)

build_linux_arm64:
	GOOS=linux GOARCH=arm64 $(GOBUILD) -trimpath $(LDFLAGS) -o $(BUILDDIR)/kubackup_server_$(APPVERSION)_linux_arm64 $(MAIN)

build_osx_amd64:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -trimpath $(LDFLAGS) -o $(BUILDDIR)/kubackup_server_$(APPVERSION)_darwin_amd64 $(MAIN)

build_osx_arm64:
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -trimpath $(LDFLAGS) -o $(BUILDDIR)/kubackup_server_$(APPVERSION)_darwin_arm64 $(MAIN)

# 构建 Docker 镜像
build_image:
	docker buildx build -t kubackup/kubackup:${VERSION} -t kubackup/kubackup:latest --build-arg VERSION=${VERSION} --platform=linux/arm64,linux/amd64 . --push

