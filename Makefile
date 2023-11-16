GOVER := $(shell go version)

GOOS    := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
GOARCH  := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
GOENV   := GO111MODULE=on CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH)
GO      := $(GOENV) go
GOBUILD := $(GO) build $(BUILD_FLAG)
GOTEST  := GO111MODULE=on CGO_ENABLED=1 $(GO) test -p 3
SHELL   := /usr/bin/env bash

COMMIT    := $(shell git describe --no-match --always --dirty)
BRANCH    := $(shell git rev-parse --abbrev-ref HEAD)
BUILDTIME := $(shell date '+%Y-%m-%d %T %z')

REPO := github.com/caldog20/go-overlay
LDFLAGS := -w -s
LDFLAGS += -X "$(REPO)/version.GitHash=$(COMMIT)"
LDFLAGS += -X "$(REPO)/version.GitBranch=$(BRANCH)"
LDFLAGS += $(EXTRA_LDFLAGS)

rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

FILES = $(call rwildcard,./,*.go)


all: server peer

server:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o ./bin/controller ./cmd/controller

peer:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o ./bin/node ./cmd/node

server-mips:
	GOOS=linux GOARCH=mipsle go build -ldflags '$(LDFLAGS)' -o ./bin/controller ./cmd/controller
	scp ./bin/controller root@10.170.241.1:~

#test:
#	$(GOTEST) `go list ./... | grep -v tools | grep -v systray`

protogen:
	echo "Generating protobuf and grpc code"
	@protoc --proto_path=msg --go_out=msg --go_opt=paths=source_relative --go-grpc_out=msg --go-grpc_opt=paths=source_relative msg/*.proto

fmt:
	@echo "gofmt (simplify)"
	@gofmt -s -l -w $(FILES) 2>&1

lint:
	@echo "running golint"
	@golint ./...

vet:
	$(GO) vet ./...

check: vet fmt

runserver: server
	./bin/controller

clean:
	rm -rf ./bin

.PHONY: clean fmt lint runpeer runserver
