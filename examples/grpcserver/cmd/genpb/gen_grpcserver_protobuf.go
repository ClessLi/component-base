package main

// This file mirrors bifrost's cmd/genbifrostpb/gen_bifrost_protobuf.go protocol
// generation command and generates the protobuf code for the gRPC server example.
//
// Generation target directory: examples/grpcserver/api/protobuf-spec/grpcserverpb/v1/
// How to run: execute `go generate` inside examples/grpcserver/cmd/genpb,
// or run `go generate ./examples/grpcserver/cmd/genpb/...` from the repo root.
//
// Toolchain requirements (same as bifrost): protoc v3.13.0 + protoc-gen-go v1.23.0 (built-in grpc plugin).
//go:generate protoc -I=../.. --go_out=plugins=grpc:../.. ../../api/protobuf-spec/grpcserverpb/v1/grpcserver.proto

// With official protoc-gen-go v1.20 or later (the plugins=grpc option was
// removed), use the two commands below instead (requires protoc-gen-go-grpc
// installed separately) to generate message code and gRPC service stubs:
//
//	protoc -I=../.. --go_out=../.. --go_opt=paths=source_relative ../../api/protobuf-spec/grpcserverpb/v1/grpcserver.proto
//	protoc -I=../.. --go-grpc_out=../.. --go-grpc_opt=paths=source_relative ../../api/protobuf-spec/grpcserverpb/v1/grpcserver.proto

func main() {
}
