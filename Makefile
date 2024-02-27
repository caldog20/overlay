export GO111MODULE := on
export CGO_ENABLED := off

BIN_DIR ?= bin

frontend:
	@docker-compose up --build

docker-controller:
	@docker-compose build controller
	@docker-compose up controller

controller: buf
	go build -o $(BIN_DIR)/controller cmd/controller/main.go

run-controller: controller
	$(BIN_DIR)/controller

buf:
	@buf generate

buf-lint:
	@buf lint

deps:
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

clean:
	rm -rf $(BIN_DIR)
	rm -rf proto/gen
	rm -rf store.db

.PHONY: all controller buf docker-controller deps frontend buf-lint clean





#GOVER := $(shell go version)
#CGO := 0
#
#GOOS    := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
#GOARCH  := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
#GOENV   := GOOS=$(GOOS) GOARCH=$(GOARCH)
#GO      := $(GOENV) go
#GOBUILD := $(GO) build $(BUILD_FLAG)
#GOTEST  := $(GO) test -p 3
#SHELL   := /usr/bin/env bash
#
#COMMIT    := $(shell git describe --no-match --always --dirty)
#BRANCH    := $(shell git rev-parse --abbrev-ref HEAD)
#BUILDTIME := $(shell date '+%Y-%m-%d %T %z')
#
#REPO := github.com/caldog20/overlay
#
#LDFLAGS := -ldflags '-w -s'
#
#rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))
#
#FILES = $(call rwildcard,./,*.go)
#
#
#all: server peer
#
#vendor:
#	@go mod vendor
#
#tidy:
#	@go mod tidy
#
#server:
#	$(GOBUILD) $(LDFLAGS) -o ./bin/controller ./cmd/controller
#
#peer:
#	$(GOBUILD) $(LDFLAGS) -o ./bin/node ./cmd/node
#
#server-remote:
#	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o ./bin/controller ./cmd/controller
#	scp ./bin/controller yatesca@blue:~
#	ssh yatesca@blue -t 'sudo ~/controller'
#
#
#peer-test:
#	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ./bin/node ./cmd/node
#	scp ./bin/node yatesca@10.170.241.11:~
#	ssh yatesca@blue -t 'sudo ~/node'
#
#peer-test2:
#	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ./bin/node ./cmd/node
#	scp ./bin/node yatesca@10.170.241.66:~
#	ssh yatesca@10.170.241.66 -t 'sudo ~/node'
##test:
##	$(GOTEST) `go list ./... | grep -v tools | grep -v systray`
#
#protogen:
#	echo "Generating protobuf and grpc code"
#	@protoc --proto_path=proto --go_out=proto --go_opt=paths=source_relative --go-grpc_out=proto  --go-grpc_opt=paths=source_relative proto/control.proto
#
#fmt:
#	@echo "gofmt (simplify)"
#	@gofmt -s -l -w $(FILES) 2>&1
#
#lint:
#	@echo "running golint"
#	@golint ./...
#
#vet:
#	$(GO) vet ./...
#
#check: vet fmt
#
#runserver: server
#	./bin/controller
#
#run: peer
#	sudo ./bin/node
#
#
#clean:
#	rm -rf ./bin
#
#.PHONY: clean fmt lint runpeer runserver server peer
