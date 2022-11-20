PKG_PREFIX := github.com/iglov/mmdb-editor

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT_TAG := $(shell git rev-parse --short=12 HEAD)
DATEINFO_TAG ?= $(shell date -u +'%Y%m%d%H%M%S')

PKG_TAG ?= $(shell git tag -l --points-at HEAD)
ifeq ($(PKG_TAG),)
PKG_TAG := $(BRANCH)
endif

#LDFLAGS = -X '$(PKG_PREFIX)/main.Version=v$(VERSION)-$(DATEINFO_TAG)-$(COMMIT_TAG)'
LDFLAGS = -X 'main.Version=$(PKG_TAG)-$(DATEINFO_TAG)-$(COMMIT_TAG)'

.PHONY: $(MAKECMDGOALS)

.PHONY: all
all: check build

.PHONY: clean
clean:
	rm -rf bin/*

.PHONY: build
build:
	go build -tags "$(BUILDTAGS)" -ldflags "$(LDFLAGS)" -o bin/${APP_NAME}

.PHONY: fmt
fmt:
	gofmt -l -w -s .

.PHONY: vet
vet:
	go vet .

.PHONY: check
check: fmt vet
