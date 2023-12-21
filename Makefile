GOVER := $(shell go version)
CGO := 1

GOOS    := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
GOARCH  := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
GOENV   := GOOS=$(GOOS) GOARCH=$(GOARCH)
GO      := $(GOENV) go
GOBUILD := $(GO) build $(BUILD_FLAG)
GOTEST  := $(GO) test -p 3
SHELL   := /usr/bin/env bash

COMMIT    := $(shell git describe --no-match --always --dirty)
BRANCH    := $(shell git rev-parse --abbrev-ref HEAD)
BUILDTIME := $(shell date '+%Y-%m-%d %T %z')

REPO := github.com/caldog20/go-overlay

LDFLAGS := -ldflags '-w -s'

rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

FILES = $(call rwildcard,./,*.go)


all: server peer

vendor:
	@go mod vendor

tidy:
	@go mod tidy

server:
	$(GOBUILD) $(LDFLAGS) -o ./bin/controller ./cmd/controller

peer:
	$(GOBUILD) $(LDFLAGS) -o ./bin/node ./cmd/node

server-mips:
	GOOS=linux GOARCH=mipsle go build $(LDFLAGS) -o ./bin/controller ./cmd/controller
	scp ./bin/controller root@10.170.241.1:~
	ssh root@10.170.241.1 -t './controller'

peer-test:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ./bin/node ./cmd/node
	scp ./bin/node yatesca@10.170.241.66:~
	ssh yatesca@10.170.241.66 -t 'sudo ~/node'

peer-test2:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ./bin/node ./cmd/node
	scp ./bin/node yatesca@10.170.241.11:~
	ssh yatesca@10.170.241.11 -t 'sudo ~/node'
#test:
#	$(GOTEST) `go list ./... | grep -v tools | grep -v systray`

protogen:
	echo "Generating protobuf and twirp code"
	#@protoc --proto_path=msg --go_out=msg --go_opt=paths=source_relative --go-grpc_out=msg --go-grpc_opt=paths=source_relative msg/*.proto
	@protoc --proto_path=proto --go_out=proto --go_opt=paths=source_relative --twirp_out=proto  --twirp_opt=paths=source_relative proto/control.proto

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

runnode: peer


clean:
	rm -rf ./bin

.PHONY: clean fmt lint runpeer runserver
