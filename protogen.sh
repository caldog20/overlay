#!/bin/bash

#protoc --proto_path= --go_out=pb --go_opt=paths=source_relative pb/*.proto
echo "Generating protobuf code"
protoc --proto_path=msg --go_out=msg --go_opt=paths=source_relative --go-grpc_out=msg --go-grpc_opt=paths=source_relative msg/*.proto
