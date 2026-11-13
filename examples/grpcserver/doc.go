// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package grpcserver 演示如何基于 component-base 的通用 gRPC 服务端
// （pkg/generic_server/server）与 gRPC 五层框架（pkg/generic_server/framework/grpc）
// 搭建一个自包含的 gRPC 服务端，结构参考 bifrost 项目的服务端。
//
// 五层架构（以 go-kit 为基础）：
//
//	传输层  transport.go / handler.go / decoder.go / encoder.go  —— pb 服务实现、Handler、编解码
//	端点层  endpoint.go                                          —— Endpoint[DomainReq, DomainResp]
//	中间件层 middleware.go                                       —— Middlewares[ServiceFactory]
//	服务层  service.go                                           —— ServiceFactory / DemoService
//	存储层  store.go                                             —— StoreFactory / DemoStore（内存实现）
//
// 装配与生命周期：
//
//	Options -> CreateConfigFromOptions -> GRPCServerConfig
//	        -> NewServer(ctx, cfg) -> GenericGRPCServer
//	        -> PrepareRun() -> Run()
//
// bizHooks 实现 GRPCServerBizHooks[ServiceFactory, StoreFactory]，调用顺序为
// InitStore -> InitService -> RegisteredMiddlewares -> ServiceRegisters -> Startup -> Shutdown。
// 其中 ServiceRegisters 经 endpoint -> handler -> transport Factory -> register generator
// 产出 map[string]ServiceRegister，由通用服务端注册到 grpc.Server（及 RA，若配置）。
//
// Demo 服务含两种 RPC 形态：
//   - Echo：一元 RPC，演示 解码 -> 端点 -> 编码 链路。
//   - Download：服务器流式 RPC，由 transport.StreamHandler 封装“计算+分块下发”，pb-service 仅调 ServeGRPCStream。
//
// ChunkSize 经 NewBizHooks(chunkSize) 注入（ServiceRegisters hook 仅接收服务工厂），
// 来源于 server.GRPCServing.ChunkSize。component-base 已移除独立传输层 Options，
// chunkSize 在构造期注入 StreamHandler（NewDemoHandlers），底层经 ServeStream 调用 StreamSendMsg。
//
// 运行：
//
//	func main() {
//	    if err := grpcserver.Run(grpcserver.NewOptions()); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// 默认监听 127.0.0.1:8081（insecure）。pb 客户端调用示例见 grpcserver_test.go。
//
// 测试：
//
//	go test ./examples/grpcserver/...
//
// pb 代码由 cmd/genpb/gen_grpcserver_protobuf.go 的 go:generate 命令生成（参考 bifrost）。
package grpcserver
