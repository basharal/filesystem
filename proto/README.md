# Protobuf

## Background

We use protobuf and gRPC to implment a client/server.

## Installation

Follow instructions to install protoc [here](https://developers.google.com/protocol-buffers) and the [Go/gRPC plugins](https://grpc.io/docs/languages/go/quickstart/).

## How to compile

`protoc --go_out=. --go-grpc_out=./pb_filesystem --go_opt=module=github.com/basharal/filesystem/proto --go-grpc_opt=paths=source_relative filesystem.proto`
