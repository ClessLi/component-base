# component-base

`github.com/ClessLi/component-base` 是一套 Go 通用组件基础库，为业务项目
提供从**命令行骨架、日志、错误码**到**HTTP/gRPC 服务端框架与客户端 SDK**
的整条工程化地基。设计目标：

- **分层解耦**：服务端与客户端均按 transport → endpoint → middleware →
  service → store 分层，业务代码只实现本层契约，装配与生命周期交给框架；
- **领域类型对外**：框架内部完成 pb/JSON 与领域类型的转换，业务接口不
  泄漏传输细节；
- **错误码工程化**：错误码以注释规范定义，由 codegen 生成注册代码、
  绑定 namespace 的 wrapper 与错误码文档，客户端自动还原服务端错误码；
- **泛型优先**：核心抽象（Endpoint、Client、有序 map 等）基于 Go 泛型，
  避免 interface{} 断言散落业务代码。

## 核心特性

- 带 HTTP 状态映射与调用堆栈的错误类型，兼容 go1.13 `Is/As/Unwrap`；
- zap 内核的统一日志门面，桥接 klog/logrus/distribution；
- 方法级出入参日志中间件（HTTP/gRPC 通用）；
- 通用服务端：BizHooks 装配、原子状态机、Consul 服务注册（skirnir）、
  健康探测、优雅停机、Prometheus metrics（HTTP）；
- gRPC 服务端流式分块发送与客户端透明聚合，调用方无流感知；
- cobra 命令行骨架与 YAML 配置模板生成工具；
- 错误码/文档代码生成器 codegen（AST + go/types 解析注释）。

## 环境要求

- Go 1.25.0 及以上（见 [go.mod](go.mod)）；
- 可选：protoc + protoc-gen-go（gRPC 示例 pb 生成）、Consul（服务注册发现）。

## 目录结构

```text
component-base/
├── pkg/
│   ├── app/               # CLI 应用骨架
│   ├── client-sdk/        # 客户端 SDK：grpc/v1、grpc_health/v1、http/v1
│   ├── core/              # HTTP 请求/响应核心工具
│   ├── errors/            # 错误码错误类型
│   ├── generic_server/    # 服务端：framework/{http,grpc}、server、options、logger
│   ├── genyaml/           # YAML 配置模板生成骨架
│   ├── log/               # 日志门面 log/v1 及 klog/logrus/distribution 桥接
│   ├── methodlog/         # 方法级日志中间件 methodlog/v1
│   ├── prevent_cycles/    # 循环引用检测
│   ├── sort_map/          # 泛型有序 map
│   └── utils/             # 反射判空、字符串/切片转换
├── tools/codegen/         # 错误码代码/文档生成器
├── examples/              # 5 个可运行示例（见下文）
└── docs/                  # 中文文档（guide 使用指南、devel 开发规范）
```

## 快速开始

```bash
go get github.com/ClessLi/component-base
```

最小示例：用通用服务端框架跑起一个 HTTP API 服务（完整分层实现见
[examples/apiserver](examples/apiserver/doc.go)）：

```go
package main

import (
	"log"

	"github.com/ClessLi/component-base/examples/apiserver"
)

func main() {
	if err := apiserver.Run(apiserver.NewOptions()); err != nil {
		log.Fatal(err)
	}
}
```

默认监听 `127.0.0.1:8080`，阻塞至 SIGINT/SIGTERM 优雅停机。验证：

```bash
curl http://127.0.0.1:8080/healthz
curl -X POST http://127.0.0.1:8080/v1/users -d '{"name":"alice"}'
curl http://127.0.0.1:8080/v1/users
```

在业务项目中落地时，参考 examples 的分层组织复制各层契约，替换 store 与
service 为真实实现即可（各示例 doc.go 写有逐步复用指引）。

## 通用库能力总览

| 库                         | 用途                                      | 文档                                                                                         |
|---------------------------|-----------------------------------------|--------------------------------------------------------------------------------------------|
| errors                    | 错误码/HTTP 状态映射、聚合、堆栈                     | [errors.md](docs/guide/zh-CN/libs/errors.md)                                               |
| log/v1                    | 统一日志门面（zap），klog/logrus/distribution 桥接 | [log.md](docs/guide/zh-CN/libs/log.md)                                                     |
| core                      | 统一响应写出、ErrResponse、请求读取                 | [core.md](docs/guide/zh-CN/libs/core.md)                                                   |
| utils                     | IsNil 反射判空、字符串/切片转换                     | [utils.md](docs/guide/zh-CN/libs/utils.md)                                                 |
| sort_map                  | 保持插入序的泛型有序 map                          | [sort_map.md](docs/guide/zh-CN/libs/sort_map.md)                                           |
| prevent_cycles            | 结构体循环引用检测                               | [prevent_cycles.md](docs/guide/zh-CN/libs/prevent_cycles.md)                               |
| app                       | cobra 命令、选项/配置装配、优雅停机                   | [app.md](docs/guide/zh-CN/libs/app.md)                                                     |
| methodlog/v1              | 方法级出入参日志中间件                             | [methodlog.md](docs/guide/zh-CN/libs/methodlog.md)                                         |
| genyaml                   | Options → YAML 配置模板工具骨架                 | [genyaml.md](docs/guide/zh-CN/libs/genyaml.md)                                             |
| framework/http            | HTTP 五层框架（gin）                          | [generic_server_framework_http.md](docs/guide/zh-CN/libs/generic_server_framework_http.md) |
| framework/grpc            | gRPC 五层框架（一元/流式、服务注册编排）                 | [generic_server_framework_grpc.md](docs/guide/zh-CN/libs/generic_server_framework_grpc.md) |
| generic_server/server     | 服务端生命周期、BizHooks、RA、健康探测                | [generic_server_server.md](docs/guide/zh-CN/libs/generic_server_server.md)                 |
| client-sdk/grpc/v1        | 领域类型 gRPC 客户端、流式透明聚合                    | [client_sdk_grpc.md](docs/guide/zh-CN/libs/client_sdk_grpc.md)                             |
| client-sdk/grpc_health/v1 | gRPC 健康检查 Check/Watch                   | [client_sdk_grpc_health.md](docs/guide/zh-CN/libs/client_sdk_grpc_health.md)               |
| client-sdk/http/v1        | 领域类型 HTTP 客户端、NilBody、路径参数              | [client_sdk_http.md](docs/guide/zh-CN/libs/client_sdk_http.md)                             |

完整索引与阅读路线见 [docs/guide/zh-CN/libs/index.md](docs/guide/zh-CN/libs/index.md)。

## codegen 工具

`tools/codegen` 从整型常量注释生成三类产物：错误码注册代码
（`code_generated.go`）、绑定 namespace 的 errors wrapper
（`errors_generated.go`）与错误码 Markdown 文档。注释规范：

```go
// ErrUserNotFound - 404: User not found.
const ErrUserNotFound int = 100101
```

安装与 go:generate 用法：

```bash
go install github.com/ClessLi/component-base/tools/codegen@latest
```

参数全表、注释解析规则与已知限制见
[tools/codegen.md](docs/guide/zh-CN/tools/codegen.md)；落地示例见
[examples/code](examples/code/doc.go)。

## examples 运行指引

| 示例         | 演示                           | 测试命令                                |
|------------|------------------------------|-------------------------------------|
| apiserver  | HTTP 服务端（User CRUD）          | `go test ./examples/apiserver/...`  |
| grpcserver | gRPC 服务端（Echo + 流式 Download） | `go test ./examples/grpcserver/...` |
| grpcclient | gRPC 客户端 SDK（自动拉起对端）         | `go test ./examples/grpcclient/...` |
| httpclient | HTTP 客户端 SDK（自动拉起对端）         | `go test ./examples/httpclient/...` |
| code       | 错误码定义 + codegen 产物           | `go test ./examples/code/...`       |

各示例的分层结构、运行方式与复用步骤见
[examples 总览](docs/guide/zh-CN/examples/index.md)。

## 文档导航

- [使用指南（zh-CN）](docs/guide/zh-CN/libs/index.md)：全部通用库文档；
- [examples 总览](docs/guide/zh-CN/examples/index.md)：示例运行/测试指引；
- [codegen 工具](docs/guide/zh-CN/tools/codegen.md)：代码生成器手册；
- [开发规范](docs/devel/zh-CN/scope.md)：提交 scope 规范。

## 贡献

提交信息遵循 Conventional Commits，scope 取值以
[docs/devel/zh-CN/scope.md](docs/devel/zh-CN/scope.md) 为准（如
`feat(generic_server): ...`、`docs(log): ...`）。欢迎通过 Issue 与 Pull
Request 参与改进。

## 许可证

[MIT License](LICENSE)