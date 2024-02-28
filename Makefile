export GO111MODULE := on
export CGO_ENABLED := 0

BIN_DIR ?= bin

PROTO_SRCS += proto/controller/v1/controller.proto
PROTO_SRCS += proto/controller/v1/peer.proto

PROTO_OUTPUT += proto/gen/controller/v1/controller.pb.go
PROTO_OUTPUT += proto/gen/controller/v1/controller_grpc.pb.go
PROTO_OUTPUT += proto/gen/controller/v1/peer.pb.go

all: controller node


frontend:
	@docker-compose up --build

docker-controller:
	@docker-compose build controller
	@docker-compose up controller

controller: buf
	go build -o $(BIN_DIR)/controller cmd/controller/main.go

run-controller: controller
	$(BIN_DIR)/controller

node: buf
	go build -o $(BIN_DIR)/node cmd/node/main.go

buf: $(PROTO_OUTPUT)

$(PROTO_OUTPUT): $(PROTO_SRCS)
	@echo Generating proto...
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

.PHONY: all controller docker-controller deps frontend buf-lint clean node all

