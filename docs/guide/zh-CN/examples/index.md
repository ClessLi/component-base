# examples 总览

`examples/` 目录包含 5 个可运行、带测试的示例，覆盖 component-base 的核心
使用路径：服务端（HTTP/gRPC）、客户端 SDK（HTTP/gRPC）与错误码工程化。
每个示例的 `doc.go` 都写有分层说明、复用步骤与运行指引，本文是其汇总索引。

## 示例清单

| 示例                                                   | 演示内容                                                      | 主要依赖库                                                                                                                                                                          | 文档                                    |
|------------------------------------------------------|-----------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------|
| [apiserver](../../../../examples/apiserver/doc.go)   | 基于 GenericAPIServer 的 HTTP API 服务端（内存 User CRUD）          | [generic_server/server](../libs/generic_server_server.md)、[framework/http](../libs/generic_server_framework_http.md)、[core](../libs/core.md)、[methodlog](../libs/methodlog.md) | 见下文                                   |
| [grpcserver](../../../../examples/grpcserver/doc.go) | 基于 GenericGRPCServer 的 gRPC 服务端（一元 Echo + 服务端流式 Download） | [generic_server/server](../libs/generic_server_server.md)、[framework/grpc](../libs/generic_server_framework_grpc.md)                                                           | 见下文                                   |
| [grpcclient](../../../../examples/grpcclient/doc.go) | 分层 gRPC 客户端 SDK（grpcserver 的对端）                           | [client-sdk/grpc](../libs/client_sdk_grpc.md)                                                                                                                                  | 见下文                                   |
| [httpclient](../../../../examples/httpclient/doc.go) | 分层 HTTP 客户端 SDK（apiserver 的对端）                            | [client-sdk/http](../libs/client_sdk_http.md)                                                                                                                                  | 见下文                                   |
| [code](../../../../examples/code/doc.go)             | 错误码定义规范 + codegen 生成产物                                    | [errors](../libs/errors.md)、[tools/codegen](../tools/codegen.md)                                                                                                               | [错误码表（生成产物）](error_code_generated.md) |

对端关系：`httpclient <-> apiserver`、`grpcclient <-> grpcserver`；客户端
示例的测试会自动在空闲端口拉起对端服务端做回环验证，无需手工启动。

## apiserver：HTTP 服务端

五层结构（单包多文件）：

```text
examples/apiserver/
├── api/v1/types.go     # 领域类型 User/CreateUserRequest（与 httpclient 共享）
├── controller.go       # 传输/控制器层（gin handler）
├── middleware.go       # 中间件层（装饰 ServiceFactory）
├── service.go          # 业务服务层
├── store.go            # 内存存储层
├── hooks.go            # APIServerBizHooks 实现（各 Init 只委托本层构造器）
├── server.go           # Options -> Config -> NewAPIServer 装配链 + Run
└── *_test.go           # httptest 驱动 gin 路由，无需网络端口
```

运行（默认监听 `127.0.0.1:8080`，`Run` 阻塞至 SIGINT/SIGTERM 优雅停机）：

```go
func main() {
if err := apiserver.Run(apiserver.NewOptions()); err != nil {
log.Fatal(err)
}
}
```

验证端点：

```bash
curl http://127.0.0.1:8080/healthz
curl -X POST http://127.0.0.1:8080/v1/users -d '{"name":"alice"}'
curl http://127.0.0.1:8080/v1/users/1
curl http://127.0.0.1:8080/v1/users
```

测试：`go test ./examples/apiserver/...`

## grpcserver：gRPC 服务端

五层结构 + pb 协议与生成命令：

```text
examples/grpcserver/
├── api/protobuf-spec/grpcserverpb/v1/   # .proto 与生成的 pb 代码
├── cmd/genpb/                           # go:generate protoc 命令
├── transport.go / handler.go            # pb 服务实现 + 一元/流式 handler
├── decoder.go / encoder.go              # pb <-> 领域类型编解码
├── endpoint.go / middleware.go          # 端点层 / 中间件层
├── service.go / store.go                # 业务层 / 内存存储层
├── register.go                          # NewServiceRegistration 声明
├── hooks.go / server.go                 # GRPCServerBizHooks + 装配链
└── *_test.go
```

装配链（与 [doc.go](../../../../examples/grpcserver/doc.go) 一致）：
`Options → CreateConfigFromOptions → GRPCServerConfig → cfg.Complete().NewGRPCServer(ctx, hooks) → PrepareRun() → Run()`。

Demo 服务两种 RPC 形态：

- **Echo**（一元）：decode -> endpoint -> encode 管线；
- **Download**（服务端流式）：`transport.StreamHandler` 封装「计算 + 分块
  发送」，ChunkSize 由 generic server 经 `InitTransport` 钩子注入
  （来源 `GRPCServing.ChunkSize`，默认 1024）。

运行（默认监听 `127.0.0.1:8081`，insecure）：

```go
func main() {
if err := grpcserver.Run(grpcserver.NewOptions()); err != nil {
log.Fatal(err)
}
}
```

测试：`go test ./examples/grpcserver/...`

### pb 代码生成（genpb）

[cmd/genpb/gen_grpcserver_protobuf.go](../../../../examples/grpcserver/cmd/genpb/gen_grpcserver_protobuf.go)
中的 go:generate 命令：

```bash
go generate ./examples/grpcserver/cmd/genpb/...
```

- 原工具链要求：protoc v3.13.0 + protoc-gen-go v1.23.0（内置 grpc 插件，
  `--go_out=plugins=grpc`）；
- protoc-gen-go v1.20+ 移除了 `plugins=grpc`，需另装 protoc-gen-go-grpc，
  改用两条命令（文件注释中已给出）：

```bash
protoc -I=../.. --go_out=../.. --go_opt=paths=source_relative ../../api/protobuf-spec/grpcserverpb/v1/grpcserver.proto
protoc -I=../.. --go-grpc_out=../.. --go-grpc_opt=paths=source_relative ../../api/protobuf-spec/grpcserverpb/v1/grpcserver.proto
```

（在 `examples/grpcserver/cmd/genpb` 目录内执行。）

## grpcclient：gRPC 客户端 SDK

分层结构（transport 内含编解码子包，client 持有连接）：

```text
examples/grpcclient/
├── transport/          # 每 RPC 一个 grpcclientv1.Client；持有 *grpc.ClientConn
│   ├── encoder/        # Encoder[REQ, ProtoReq] 工厂
│   └── decoder/        # Decoder / MsgExtractor / StreamDecoder 工厂
├── endpoint/           # Client -> 领域类型 Endpoint 适配
├── service/            # 组装 GRPCRequest 的业务方法
├── client.go           # transport.New -> endpoint.New -> service.New 装配
├── config.go           # Config（默认 127.0.0.1:8081，insecure）
└── *_test.go           # 自动拉起 grpcserver 回环验证
```

使用：

```go
client, err := grpcclient.New("127.0.0.1:8081",
grpc.WithTransportCredentials(insecure.NewCredentials()))
// 或 Config 驱动：grpcclient.NewConfig().Complete().NewClient()
defer client.Close()

resp, err := client.Demo().Echo("hello") // -> "echo: hello"
data, err := client.Demo().Download("greeting") // 流式分块已被 SDK 聚合为完整 []byte
```

测试：`go test ./examples/grpcclient/...`

## httpclient：HTTP 客户端 SDK

四层结构（域类型与 apiserver 共享 `examples/apiserver/api/v1`）：

```text
examples/httpclient/
├── transport/          # 每操作一个 httpclientv1.Client；持有 *http.Client 与 baseURL
├── endpoint/           # Client -> 领域类型 Endpoint 适配
├── service/            # 组装 HTTPRequest 的业务方法（Get 用 NilBody + PathVars）
├── client.go           # NewTransport -> NewEndpoint -> NewService -> NewClient
└── *_test.go           # 自动拉起 apiserver，验证 Create -> Get -> List 回环
```

使用（先启动 apiserver）：

```go
c := httpclient.NewClient(context.Background(), "http://127.0.0.1:8080", nil)
defer c.Close() // no-op，保持 API 对称

u, err := c.Users().Create(&apiv1.CreateUserRequest{Name: "alice"})
got, err := c.Users().Get(u.ID)
list, err := c.Users().List()
```

`NewClient(ctx, baseURL, httpClient)` 第三参传 nil 时使用新建的
`&http.Client{}`；ctx 作为每个请求的种子上下文。

测试：`go test ./examples/httpclient/...`

## code：错误码工程化

```text
examples/code/
├── 0base.go                 # 手写：错误码常量 + 注释规范 + go:generate 指令
├── code_generated.go        # 生成：init() 注册错误码（勿手工编辑）
├── errors_generated.go      # 生成：绑定 namespace 的 errors wrapper（勿手工编辑）
├── usage_example.go         # 手写：错误码使用示例
└── *_test.go
```

错误码采用 SSSMMM 六位数模式（SSS 服务 ID + MMM 模块 ID），注释格式
`// ErrXxx - <HTTP状态码>: <描述>.`。生成命令（需先安装
[codegen](../tools/codegen.md)）：

```bash
go generate ./examples/code/...
```

生成产物中的错误码文档见 [error_code_generated.md](error_code_generated.md)
（生成产物，不手工编辑）。

## 注意事项

1. **构造器命名不统一**：grpcclient 各层入口为 `New`，httpclient 为
   `NewXxx`（NewTransport/NewEndpoint/NewService/NewClient），以各自代码为准。
2. **code/doc.go 注释待修正**：其注释提到的 `1proxy_claim.go`、
   `2proxy_controller.go` 在当前目录中不存在，实际仅有 `0base.go`
   （服务 ID 10）；以实际文件为准。
3. 一键全量测试（由使用者自行执行）：

   ```bash
   go test -race ./examples/...
   ```