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

fmt:
	gofmt -l -w -s ./

vet:
	go vet ./...

lint: install-golint
	golint ./...

install-golint:
	which golint || go install golang.org/x/lint/golint@latest

govulncheck: install-govulncheck
	govulncheck ./...

install-govulncheck:
	which govulncheck || go install golang.org/x/vuln/cmd/govulncheck@latest

errcheck: install-errcheck
	errcheck -exclude=errcheck_excludes.txt ./...

install-errcheck:
	which errcheck || go install github.com/kisielk/errcheck@latest

golangci-lint: install-golangci-lint
	golangci-lint run --exclude '(SA4003|SA1019|SA5011):' -D errcheck -D structcheck --timeout 2m

install-golangci-lint:
	which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.48.0

check: fmt vet lint errcheck golangci-lint govulncheck

