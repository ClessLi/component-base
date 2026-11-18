# generic_server：通用服务端骨架（server / options / logger）

## 库定位与适用场景

`pkg/generic_server` 的 `server`、`options`、`logger` 三个包构成配置驱动的通用服务端：

- **server**：`GenericAPIServer`（gin HTTP/HTTPS）与 `GenericGRPCServer`（gRPC），
  统一生命周期（PrepareRun → Run → Close）、优雅停机、健康检查、（gRPC）RA 服务注册；
- **options**：命令行 flag / 配置文件 → Options →（Validate/Complete/ApplyTo）→ Config；
- **logger**：info/error 双通道日志初始化（目录创建 + [log/v1](log.md) Init + Flush）。

适用场景：以 [app](app.md) 骨架承载的服务进程，希望存储/服务/中间件/端点/传输各层
初始化顺序、健康检查与停机清理全部由框架控制，业务只实现 hooks。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import (
genericserver "github.com/ClessLi/component-base/pkg/generic_server/server"
genericoptions "github.com/ClessLi/component-base/pkg/generic_server/options"
"github.com/ClessLi/component-base/pkg/generic_server/logger"
)
```

## 核心概念与生命周期

### 配置链路

```text
Options（flag/配置文件）→ Validate → Complete → ApplyTo
  → APIServerConfig / GRPCServerConfig → Complete()
  → completed.NewAPIServer(ctx, hooks) / completed.NewGRPCServer(ctx, hooks)
  → PrepareRun() → Run()（阻塞）→ Close()（信号或手动）
```

### 业务 hooks（hooks.go）

| 接口                                            | 方法（按框架调用顺序）                                                                                                                                                                                                                                                 |
|-----------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `APIServerBizHooks[Svc, Store any]`           | `InitStore() (Store, error)` → `InitService(Store) (Svc, error)` → `RegisteredMiddlewares() (mwv1.Middlewares[Svc], error)` → `InitController(router *gin.Engine, svc Svc) error`；`Startup() error`（Run 时、监听前）；`Shutdown() error`（停机清理）                     |
| `GRPCServerBizHooks[Txp, Ep, Svc, Store any]` | `InitStore` → `InitService` → `RegisteredMiddlewares` → `InitEndpoints(svc Svc) (Ep, error)` → `InitTransport(eps Ep, chunkSize int) (Txp, error)` → `ServiceRegistrations(txp Txp) ([]svcregisterv1.ServiceRegistration, error)`；`Startup` / `Shutdown` 同上 |

hooks 各方法只委托本层构造器，不做跨层装配；初始化失败的层由框架 defer 调用
`Shutdown` 清理（要求 Shutdown 幂等）。

### 状态机（const.go）

`ServerStateCreated(0) → ServerStateRunning(1) → ServerStateClosing(2) → ServerStateStopped(3)`，
atomic CAS 迁移；`Close` 幂等可并发；Stopped 后可再次 `Run`（gRPC 每轮重建
grpc.Server）；Run 前收到停机信号（`shutdownBeforeRun`）则拒绝启动。

## 关键类型与函数清单

### server/config.go、const.go

| 签名                                                                                                                                     | 说明                                                                                                                    |
|----------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------|
| `const RecommendedHomeDir = ".component"` / `RecommendedEnvPrefix = "COMPONENT"`                                                       | 推荐配置目录与环境变量前缀                                                                                                         |
| `type CertKey struct { CertFile, KeyFile string }`                                                                                     | PEM 证书/私钥文件                                                                                                           |
| `type SecureServingInfo struct { BindAddress string; BindPort int; CertKey CertKey }` / `func (s *SecureServingInfo) Address() string` | TLS 监听配置                                                                                                              |
| `type InsecureServingInfo struct { BindAddress string; BindPort int }` / `func (i *InsecureServingInfo) Address() string`              | 明文监听配置                                                                                                                |
| `func LoadConfig(cfg string, defaultName string, homeDir string, envPrefix string)`                                                    | viper 读配置：指定文件优先，否则依次搜 `.`、`~/<homeDir>`、`/etc/component`；yaml；环境变量前缀 + `.`/`-`→`_` 替换；**读取失败仅 Warn 不退出**（与 app 骨架不同） |
| `ServerStateCreated/Running/Closing/Stopped int32`                                                                                     | 状态常量                                                                                                                  |

### server/http.go

| 签名                                                                                                                                                                                                                                              | 说明                                                                |
|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------|
| `type APIServerConfig[Svc, Store any] struct { SecureServing; InsecureServing; Middlewares []string; Healthz bool; EnableProfiling bool; EnableMetrics bool; Mode string; HealthCheckPath string; PingTimeout, ShutdownTimeout time.Duration }` | HTTP 服务配置                                                         |
| `func NewAPIServerConfig[Svc, Store any]() *APIServerConfig[Svc, Store]`                                                                                                                                                                        | 默认：Healthz/EnableProfiling/EnableMetrics=true，Middlewares=[]，其余零值 |
| `func (c *APIServerConfig[Svc, Store]) Complete() CompletedAPIServerConfig[Svc, Store]`                                                                                                                                                         | 直通包装（不补默认值，默认值靠 Options.Complete）                                 |
| `func (c CompletedAPIServerConfig[Svc, Store]) NewAPIServer(ctx context.Context, bizHooks APIServerBizHooks[Svc, Store]) *GenericAPIServer[Svc, Store]`                                                                                         | hooks/ctx 为 nil 直接 Panicf；Mode 非空时 gin.SetMode                    |
| `func (s *GenericAPIServer[Svc, Store]) PrepareRun() PreparedGenericAPIServer`                                                                                                                                                                  | initRouter（gin.Default）→ initBiz → initLifecycle；失败 Panicf        |
| `type PreparedGenericAPIServer interface { Run() error; Close() error }`                                                                                                                                                                        | 就绪服务器                                                             |
| `func (s *GenericAPIServer[Svc, Store]) Close() error`                                                                                                                                                                                          | 幂等优雅停机                                                            |

initBiz 顺序：Store → Service → 中间件（内置 `gin.Recovery()` + 按名安装自定义）→
metrics（ginprometheus，`id`/`name` 路径参数归一为 `{id}`/`{name}`）→
healthz（`GET HealthCheckPath` 返回 `{"status":"ok"}`）→ profiling（**TODO 未实现**）→
`GET /version`（marmotedu version.Get）→ InitController。

Run/run：Startup hook → 启动 insecure HTTP（启动后 ping 健康检查：5s 客户端超时、
1s→8s 指数退避重试，超过 PingTimeout 失败并关停）→ 启动 secure HTTPS
（**无 ping**，源码 TODO）→ errgroup.Wait 阻塞。

Close：并行 `http.Server.Shutdown`（ShutdownTimeout 上限）→ cancel 内部 ctx →
hooks.Shutdown；错误聚合为 `errors.NewAggregate`。

### server/grpc.go

| 签名                                                                                                                                                                                                                                                                     | 说明                                                             |
|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------|
| `type RAInfo struct { Host string; Port int }`                                                                                                                                                                                                                         | RA（注册中心）地址                                                     |
| `type GRPCServingInfo struct { ChunkSize int; RecvTimeoutM int }` / `func NewGRPCServingInfo() *GRPCServingInfo`                                                                                                                                                       | 默认 ChunkSize=1024、RecvTimeoutM=1（分钟）                           |
| `type GRPCServerConfig[Txp, Ep, Svc, Store any] struct { Namespace string; SecureServing; InsecureServing; RA *RAInfo; GRPCServing *GRPCServingInfo; Middlewares []string; Healthz, EnableProfiling, EnableMetrics bool; PingTimeout, ShutdownTimeout time.Duration }` | gRPC 服务配置（携带 Txp/Ep 以支持流式构建）                                   |
| `func NewGRPCServerConfig[Txp, Ep, Svc, Store any]() *GRPCServerConfig[Txp, Ep, Svc, Store]`                                                                                                                                                                           | 默认：Healthz/Profiling/Metrics=true，Ping/Shutdown=30s            |
| `func (c *GRPCServerConfig[Txp, Ep, Svc, Store]) Complete() CompletedGRPCServerConfig[Txp, Ep, Svc, Store]`                                                                                                                                                            | 补 GRPCServing 默认值与 Ping/Shutdown=30s                           |
| `func (c CompletedGRPCServerConfig[Txp, Ep, Svc, Store]) NewGRPCServer(ctx context.Context, bizHooks GRPCServerBizHooks[Txp, Ep, Svc, Store]) *GenericGRPCServer[Txp, Ep, Svc, Store]`                                                                                 | 流式构建方法（与 HTTP 的 NewAPIServer 对齐）；hooks/ctx nil → Panicf        |
| `func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) PrepareRun() PreparedGenericGRPCServer`                                                                                                                                                                              | 无任何 serving → Panicf；installRAClient → initBiz → initLifecycle |
| `func (s *GenericGRPCServer[Txp, Ep, Svc, Store]) Close() error`                                                                                                                                                                                                       | 幂等：先 RA 摘流 → GracefulStop（超时强制 Stop）→ cancel → hooks.Shutdown  |

- **RA 注册**：`RAInfo` 非空且 Host/Port 有值时经 skirnir
  `discover.NewKitConsulRegistryClient` 连接 Consul；实例 ID 为
  `<服务名>-<uuid>`；每个服务对 insecure/secure 两个 serving 分别注册；
  注册/反注册失败仅 Warn。
- **initBiz**：Store → Service → 中间件 → metrics/profiling（**均 TODO 未实现**）→
  InitEndpoints → InitTransport(eps, ChunkSize) → ServiceRegistrations →
  [GenerateServiceRegisters](generic_server_framework_grpc.md)。
- **run**：Startup → buildServers（每轮新建 grpc.Server，TLS 证书每轮重载）→
  registerServices（health 服务 + 业务服务 + RA）→ 同步 `net.Listen`（端口冲突立即
  报错）→ `healthSvr.Resume()` → Serve → ping（用
  [client-sdk/grpc_health](client_sdk_grpc_health.md) 逐服务检查 SERVING，
  1s→8s 退避，PingTimeout 上限；无已注册服务时回退检查整体状态 `""`）。
- **ReceiveTimeout**：`RecvTimeoutM` 分钟转 `time.Duration` 存于服务器字段，
  当前源码未见消费点（待确认）。

### options/（6 文件）

所有 Options 均实现 `AddFlags(*pflag.FlagSet)`、`Validate() []error`、
`Complete() error`、`ApplyTo(...)`（Serving 类为 `ApplyToAPIConfig` 方法；
应用到 gRPC config 时为包级泛型函数 `ApplyInsecureServingToGRPCConfig` /
`ApplySecureServingToGRPCConfig`，Txp/Ep 从 config 实参推导）。

| 类型（文件）                                                                                                 | 构造器                                                             |
|--------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------|
| `APIServerRunOptions[Svc, Store]` / `GRPCServerRunOptions[Txp, Ep, Svc, Store]`（server_run_options.go） | `NewAPIServerRunOptions` / `NewGRPCServerRunOptions`            |
| `InsecureServingOptions[Svc, Store]`（insecure_serving_options.go，另有 `Addr() string`）                   | `NewInsecureServingOptions`                                     |
| `SecureServingOptions[Svc, Store]`（secure_serving_options.go，含 `CertKey`/`GeneratableKeyCert`）         | `NewSecureServingOptions`                                       |
| `GRPCServingOptions[Txp, Ep, Svc, Store]`（grpc_serving_options.go）                                     | `NewGRPCServingOptions`                                         |
| `RAOptions[Txp, Ep, Svc, Store]`（ra_options.go）                                                        | `NewRAOptions`                                                  |
| `LoggerOptions`（logger_options.go，非泛型；另有 `WithProjectName`/`WithLogBaseDir` 链式方法）                      | `NewLoggerOptions` / `NewLoggerOptionsWithProject(projectName)` |

### logger/（config.go、logger.go）

| 签名                                                                                           | 说明                                                                            |
|----------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------|
| `type Config struct { InfoLogOpts, ErrLogOpts *logV1.Options }` / `func NewConfig() *Config` | 双通道配置，默认各取 `logV1.NewOptions()`                                               |
| `func (c *Config) Complete() CompletedConfig`                                                | 直通包装                                                                          |
| `func (c CompletedConfig) NewLogger() (*Logger, error)`                                      | 构造延迟初始化的 Logger                                                               |
| `func (l *Logger) Init() error`                                                              | 创建输出目录（跳过 stdout/stderr，MkdirAll 0755）→ `logV1.Init(InfoLogOpts, ErrLogOpts)` |
| `func (l *Logger) Flush()`                                                                   | `logV1.Flush()`，进程退出前调用                                                       |

## 配置项与默认值

### APIServerRunOptions（flag 前缀 `server.`）

| flag                         | 类型        | 默认值        | 校验                         |
|------------------------------|-----------|------------|----------------------------|
| `--server.mode`              | string    | `debug`    | debug/test/prod 之一（空串合法）   |
| `--server.healthz`           | bool      | `true`     | 开启时 health-check-path 不得为空 |
| `--server.health-check-path` | string    | `/healthz` | 非空时必须以 `/` 开头              |
| `--server.ping-timeout`      | duration  | `30s`      | ≥0                         |
| `--server.shutdown-timeout`  | duration  | `30s`      | ≥0                         |
| `--server.middlewares`       | string 数组 | `[]`       | 名字必须在注册表中，否则初始化报错          |

### GRPCServerRunOptions

同上但无 mode/health-check-path；`--server.middlewares` 为 StringSlice（逗号分隔）。

### InsecureServingOptions / SecureServingOptions

| flag                                                       | 默认值                      | 说明                                                |
|------------------------------------------------------------|--------------------------|---------------------------------------------------|
| `--insecure.bind-address`                                  | `0.0.0.0`                | Complete 时补空值                                     |
| `--insecure.bind-port`                                     | `8080`                   | 0=禁用；范围 1-65535                                   |
| `--secure.bind-address`                                    | `0.0.0.0`                | 同上                                                |
| `--secure.bind-port`                                       | `0`（默认禁用 TLS）            | 0=禁用；启用后校验证书配置                                    |
| `--secure.tls.cert-file` / `--secure.tls.private-key-file` | `""`                     | 显式证书/私钥                                           |
| `--secure.tls.cert-dir`                                    | `/var/run/eir/apiserver` | 未显式给证书时必填                                         |
| `--secure.tls.pair-name`                                   | `eir-apiserver`          | 未显式给证书时必填；Complete 拼出 `<cert-dir>/<pair-name>.crt |.key` |

### GRPCServingOptions / RAOptions

| flag                     | 默认值         | 校验                        |
|--------------------------|-------------|---------------------------|
| `--grpc.chunksize`       | `1024`      | ≥100                      |
| `--grpc.receive-timeout` | `1`（分钟）     | >0                        |
| `--ra.host`              | `""`（禁用 RA） | 非空时 port 必须 1-65535       |
| `--ra.port`              | `0`         | ApplyTo 时 port=0 → RA=nil |

### LoggerOptions（flag 前缀 `log.`）

| flag                                                                                             | 默认值                       |
|--------------------------------------------------------------------------------------------------|---------------------------|
| `--log.log-base-dir`                                                                             | `logs`                    |
| `--log.info-level` / `--log.error-level`                                                         | `info` / `warn`（zap 级别文本） |
| `--log.format`                                                                                   | `console`（或 `json`）       |
| `--log.info-output-paths`                                                                        | `[logs/info.log]`         |
| `--log.error-output-paths`                                                                       | `[logs/error.log]`        |
| `--log.inner-error-output-paths`                                                                 | `[stderr]`（zap 内部错误输出）    |
| `--log.disable-caller` / `--log.disable-stacktrace` / `--log.enable-color` / `--log.development` | 均 `false`                 |

`NewLoggerOptionsWithProject("my-service")` 把默认路径改为
`logs/my-service.log` / `logs/my-service_error.log`；`WithProjectName` 只改项目名
不改已设路径，但 `Complete` 在路径为空时会按项目名补默认文件名。
ApplyTo 后 info 通道 Name=项目名、ErrorOutputPaths=inner-error 路径；
error 通道 ErrorOutputPaths=nil。

## 最小可运行示例（HTTP）

改写自 [examples/apiserver/server.go](../../../../examples/apiserver/server.go)：

```go
opts := &Options{
APIServerRunOptions:    genericoptions.NewAPIServerRunOptions[ServiceFactory, StoreFactory](),
InsecureServingOptions: genericoptions.NewInsecureServingOptions[ServiceFactory, StoreFactory](),
}
opts.APIServerRunOptions.Middlewares = []string{"logging"}

// Validate → Complete → ApplyTo
cfg := genericserver.NewAPIServerConfig[ServiceFactory, StoreFactory]()
// ...（各 opts.Validate()/Complete()/ApplyTo(cfg)，见示例源码）

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

srv := cfg.Complete().NewAPIServer(ctx, NewBizHooks()) // hooks 实现 APIServerBizHooks
if err := srv.PrepareRun().Run(); err != nil {
// 启动失败或异常退出
}
```

## 进阶示例（gRPC + RA + 日志）

改写自 [examples/grpcserver/server.go](../../../../examples/grpcserver/server.go)：

```go
runOpts := genericoptions.NewGRPCServerRunOptions[Factory, DemoEndpoints, ServiceFactory, StoreFactory]()
runOpts.Middlewares = []string{"logging"}
servingOpts := genericoptions.NewGRPCServingOptions[Factory, DemoEndpoints, ServiceFactory, StoreFactory]()
insecureOpts := genericoptions.NewInsecureServingOptions[ServiceFactory, StoreFactory]()
insecureOpts.BindAddress, insecureOpts.BindPort = "127.0.0.1", 8081

cfg := genericserver.NewGRPCServerConfig[Factory, DemoEndpoints, ServiceFactory, StoreFactory]()
// runOpts/servingOpts 依次 Validate/Complete/ApplyTo；
// insecure 用 genericoptions.ApplyInsecureServingToGRPCConfig(insecureOpts, cfg)

srv := cfg.Complete().NewGRPCServer(ctx, NewBizHooks()) // hooks 实现 GRPCServerBizHooks[Factory, DemoEndpoints, ...]
if err := srv.PrepareRun().Run(); err != nil {
// ...
}
```

日志初始化（app 启动早期）：

```go
logOpts := genericoptions.NewLoggerOptionsWithProject("my-service")
logConf := logger.NewConfig()
_ = logOpts.ApplyTo(logConf)
lg, _ := logConf.Complete().NewLogger()
if err := lg.Init(); err != nil {
// ...
}
defer lg.Flush()
```

完整 hooks 实现参考 [examples/apiserver/hooks.go](../../../../examples/apiserver/hooks.go)
与 [examples/grpcserver/hooks.go](../../../../examples/grpcserver/hooks.go)
（逐方法委托本层构造器 + `var _` 编译期接口断言）。

## 与其他库的协作关系

- **[app](app.md)**：Options 经 app 的 flag/viper 体系绑定；examples 以库形式组装，
  业务项目通常再包一层 app.NewApp。
- **[framework/http](generic_server_framework_http.md) /
  [framework/grpc](generic_server_framework_grpc.md)**：hooks 消费其契约类型；
  gRPC 注册控制流程由 service_register 包执行。
- **[log/v1](log.md) / [methodlog](methodlog.md)**：logger 包封装双通道 Init；
  框架自身日志全部走 log/v1 全局。
- **[core](core.md)**：healthz/version 端点用 `core.WriteResponse`。
- **[errors](errors.md)**：校验错误聚合 `NewAggregate`；停机错误聚合返回。
- **[client-sdk/grpc_health](client_sdk_grpc_health.md)**：gRPC 启动 ping 客户端。
- 外部依赖：gin、go-gin-prometheus、marmotedu/iam shutdown（POSIX 信号）、
  marmotedu/component-base（homedir/version）、skirnir discover（Consul RA）、
  satori/go.uuid、errgroup。

## 注意事项、限制与常见问题

1. **PrepareRun 失败即 panic**：initBiz/initLifecycle/RA 安装失败均 `logV1.Panicf`，
   不返回 error；`NewAPIServer`/`NewGRPCServer` 对 nil ctx/hooks 同样 panic。
2. **HTTP 默认值依赖 Options.Complete**：`APIServerConfig.Complete()` 是直通包装，
   直接手工构造 Config 而不经过 Options 时 Ping/ShutdownTimeout 为 0
   （ping 立即超时、Shutdown 无等待），务必设置。
3. **未实现能力（源码 TODO）**：HTTP/gRPC 的 profiling、gRPC 的 metrics 拦截器与
   内置 recovery 拦截器、HTTPS 的启动 ping、hooks 的 Uninstall 系列均未实现；
   `EnableProfiling`/`EnableMetrics`（gRPC）开关当前无实际效果。
4. **证书不自动生成**：`--secure.tls.cert-file` 帮助文本称可自动生成证书，但
   `Complete` 仅拼接 `<cert-dir>/<pair-name>.crt|.key` 路径，文件需预先存在，
   否则启动时加载失败。
5. **LoadConfig 与 app 行为不同**：配置文件缺失仅 Warn；app 骨架则 exit(1)。
6. **gRPC 停机顺序**：先 RA 反注册摘流，再 GracefulStop（ShutdownTimeout 后强制
   Stop，防长流挂死），最后 cancel ctx 与 hooks.Shutdown。
7. **RA 注册失败不阻断启动**：仅 Warn；服务仍正常监听。
8. **Namespace 字段未消费**：`GRPCServerConfig.Namespace` 当前源码未见使用（待确认）。
9. **ReceiveTimeout 未消费**：见上文 grpc.go 小节（待确认）。
10. **测试**：`go test -race ./examples/apiserver/... ./examples/grpcserver/...`
    （由使用者自行执行）。