package main

// 本文件参考 bifrost cmd/genbifrostpb/gen_bifrost_protobuf.go 的协议生成命令，
// 用于为 gRPC 服务端示例生成 protobuf 代码。
//
// 生成目标目录：examples/grpcserver/api/protobuf-spec/grpcserverpb/v1/
// 执行方式：在 examples/grpcserver/cmd/genpb 目录下运行 `go generate`，
// 或在仓库根目录运行 `go generate ./examples/grpcserver/cmd/genpb/...`。
//
// 工具链要求（与 bifrost 一致）：protoc v3.13.0 + protoc-gen-go v1.23.0（内置 grpc 插件）。
//go:generate protoc -I=../.. --go_out=plugins=grpc:../.. ../../api/protobuf-spec/grpcserverpb/v1/grpcserver.proto

// 若使用较新的 protoc-gen-go（v1.20+ 已移除 plugins=grpc 选项），请改用下面两条命令
// （需额外安装 protoc-gen-go-grpc），分别生成消息代码与 gRPC 服务桩：
//
//	protoc -I=../.. --go_out=../.. --go_opt=paths=source_relative ../../api/protobuf-spec/grpcserverpb/v1/grpcserver.proto
//	protoc -I=../.. --go-grpc_out=../.. --go-grpc_opt=paths=source_relative ../../api/protobuf-spec/grpcserverpb/v1/grpcserver.proto

func main() {
}
