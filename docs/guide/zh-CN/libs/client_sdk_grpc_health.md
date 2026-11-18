# client-sdk/grpc_health/v1：gRPC 健康检查客户端

## 库定位与适用场景

`pkg/client-sdk/grpc_health/v1` 封装标准 gRPC 健康检查协议
（`grpc.health.v1.Health`）的客户端，提供一次性的 `Check` 与持续订阅的
`Watch` 两种查询方式，支持直连地址与 Consul 服务发现两种构造入口。

适用场景：

- 探测 gRPC 服务端某个已注册服务的健康状态（如启动等待、探活巡检）；
- 订阅健康状态变化流（`Watch`），配合上下文取消管理订阅生命周期。

本仓库内部的典型消费方：[generic_server/server](generic_server_server.md)
的 gRPC 启动就绪探测（ping）逐服务调用 `Check` 直至全部 `SERVING`。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import grpchealthv1 "github.com/ClessLi/component-base/pkg/client-sdk/grpc_health/v1"
```

## 核心概念

- **HealthStatus**：`int32` 枚举，与 gRPC 官方健康协议取值一一对应：

  | 常量 | 值 | `StatusString` 输出 |
    |---|---|---|
  | `UNKNOWN` | 0 | `"unknown"` |
  | `SERVING` | 1 | `"serving"` |
  | `NOT_SERVING` | 2 | `"not serving"` |
  | `SERVICE_UNKNOWN` | 3 | `"service unknown"` |

- **service 参数**：健康检查的服务名。对 generic_server gRPC 服务端而言，
  传入的是 RA 注册名（`ServiceRegistration.Name`），空字符串查询整体状态。
- **Client 组合结构**：`Client` 内嵌 `*grpc.ClientConn` 与私有 `*endpoints`，
  因此同时具备 `Check`/`Watch`（来自 Service 语义）与 `Close()`（来自 conn）。

## 关键类型与函数清单

### service.go

| 签名                                                                                                                                                                     | 说明            |
|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------|
| `type HealthStatus int32`                                                                                                                                              | 健康状态枚举（常量见上表） |
| `type Service interface { Check(ctx context.Context, service string) (HealthStatus, error); Watch(ctx context.Context, service string) (<-chan HealthStatus, error) }` | 健康查询接口        |
| `func StatusString(status HealthStatus) string`                                                                                                                        | 枚举转小写描述字符串    |

### client.go

| 签名                                                                                                         | 说明                                  |
|------------------------------------------------------------------------------------------------------------|-------------------------------------|
| `type Client struct { *grpc.ClientConn; *endpoints }`                                                      | 组合连接与端点，直接实现 Service                |
| `func NewClient(svrAddr string, opts ...grpc.DialOption) (*Client, error)`                                 | 直连构造，内部用 `grpc.NewClient` 建 conn    |
| `func NewClientFromConsul(consulHost string, consulPort uint16, opts ...grpc.DialOption) (*Client, error)` | 经 skirnir Consul 发现构造（服务名硬编码，见注意事项） |

### endpoint.go

| 签名                                                                                            | 说明                                                                 |
|-----------------------------------------------------------------------------------------------|--------------------------------------------------------------------|
| `func (e *endpoints) Check(ctx context.Context, service string) (HealthStatus, error)`        | go-kit client 调用 `grpc.health.v1.Health/Check`；失败返回 `UNKNOWN` + 错误 |
| `func (e *endpoints) Watch(ctx context.Context, service string) (<-chan HealthStatus, error)` | 用 pb 生成的 `HealthClient.Watch` 开流，后台 goroutine 转发状态到 channel        |
| `func makeEndpoints(conn *grpc.ClientConn) *endpoints`                                        | 私有；组装 Check/Watch 两个端点                                             |

### transport.go

私有类型 `healthCheckRequest{Service string}` / `healthCheckResponse{Status HealthStatus}`
及编解码函数 `encodeClientRequest` / `decodeClientResponse`：在领域结构与
`grpc_health_v1.HealthCheckRequest/Response` 之间转换，类型不符时报错。

## 配置项说明

无 Options 结构。可配置点仅为透传给 `grpc.NewClient` 的
`opts ...grpc.DialOption`（如 insecure 凭据、拦截器）。连接生命周期由调用方
通过 `Client.Close()` 管理。

## 最小可运行示例（Check）

```go
client, err := grpchealthv1.NewClient("127.0.0.1:8081",
grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil { /* ... */ }
defer client.Close()

status, err := client.Check(ctx, "grpcserverpb.Demo")
if err != nil { /* ... */ }
fmt.Println(grpchealthv1.StatusString(status)) // "serving"
```

## 进阶示例（Watch 订阅）

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel() // 取消后 channel 被关闭，goroutine 退出并 CloseSend

statusC, err := client.Watch(ctx, "grpcserverpb.Demo")
if err != nil { /* ... */ }

for status := range statusC {
logV1.Infof("health status changed: %s", grpchealthv1.StatusString(status))
if status == grpchealthv1.NOT_SERVING {
// 触发告警或摘流逻辑
}
}
```

注意：`range` 只有在**调用方取消 ctx** 时才会正常结束；服务端主动关闭流时
channel 不会被关闭（见注意事项第 1 条）。

## 与其他库的协作关系

- **[generic_server/server](generic_server_server.md)**：gRPC 服务端启动时以
  本客户端逐服务 `Check`，1s 起指数退避至 8s，全部 `SERVING` 才标记 Running。
- **[framework/grpc service_register](generic_server_framework_grpc.md)**：
  `GenerateServiceRegisters` 先置 `NOT_SERVING` 再挂载后置 `SERVING`，
  本客户端查询到的状态即由此驱动。
- **[log/v1](log.md)**：Watch 的连接关闭/错误经 logV1 记录；Consul 发现日志
  用 `logV1.ZapLogger()` 适配 go-kit zap logger。
- **外部依赖**：服务发现使用 `github.com/ClessLi/skirnir/pkg/discover`；
  错误构造使用 `github.com/marmotedu/errors`（**非本仓库 pkg/errors**，
  错误码语义不互通，见注意事项第 4 条）。

## 注意事项、限制与常见问题

1. **Watch 的 channel 关闭语义不对称**：仅在 `ctx.Done()` 分支执行
   `close(statusC)` 并 `CloseSend()`；服务端返回 `io.EOF` 或其他错误时，
   后台 goroutine 只记日志后 `return`，**channel 不关闭**。消费方若用
   `for range` 且服务端断流，将永久阻塞。建议消费侧自行叠加超时/取消，
   或用 `select` 同时监听 ctx 与 channel。
2. **NewClientFromConsul 的发现服务名硬编码**为
   `"com.github.ClessLi.api.bifrost"`，不可配置；连接其他 Consul 服务名时
   该入口不适用，需改用 `NewClient` 直连（待确认后续是否参数化）。
3. **Watch 的 Recv 在 default 分支阻塞**：ctx 取消的检查发生在每轮循环开头，
   若 `Recv` 正阻塞等待下一条状态，需等 ctx 取消传导至流本身使 `Recv` 返回
   错误后才会走到退出路径。
4. **错误包混用**：client.go/transport.go 用 `marmotedu/errors`，endpoint.go
   用标准库 `errors`；返回的错误均不带本仓库 errors 包的错误码，上层如需
   码值判断应另行包装。
5. **Check 失败返回 `UNKNOWN`**：网络错误与真实 `UNKNOWN` 状态需靠 err 是否
   非 nil 区分。