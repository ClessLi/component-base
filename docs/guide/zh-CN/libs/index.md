# component-base 通用库文档索引

本目录是 `pkg/` 下全部通用库的简体中文文档。所有 API 签名、默认值均以
源码静态核实为准；示例优先改写自 [examples](../examples/index.md) 与测试中
已验证的调用。模块路径：`github.com/ClessLi/component-base`（Go 1.25.0）。

## 库清单

### 基础库

| 库              | 职责                                         | 导入路径                 | 文档                                     |
|----------------|--------------------------------------------|----------------------|----------------------------------------|
| errors         | 带错误码/HTTP 状态映射的错误类型、聚合、堆栈、go1.13 兼容        | `pkg/errors`         | [errors.md](errors.md)                 |
| log/v1         | 统一日志门面（zap 内核），klog/logrus/distribution 桥接 | `pkg/log/v1`         | [log.md](log.md)                       |
| core           | HTTP 请求/响应核心工具：统一响应写出、ErrResponse、请求读取     | `pkg/core`           | [core.md](core.md)                     |
| utils          | 反射判空（IsNil）与字符串/切片转换                       | `pkg/utils`          | [utils.md](utils.md)                   |
| sort_map       | 保持插入序的泛型有序 map、索引与列表                       | `pkg/sort_map`       | [sort_map.md](sort_map.md)             |
| prevent_cycles | 反射检测结构体循环引用，防止序列化死循环                       | `pkg/prevent_cycles` | [prevent_cycles.md](prevent_cycles.md) |

### 应用骨架与工具链

| 库             | 职责                             | 导入路径                 | 文档                                         |
|---------------|--------------------------------|----------------------|--------------------------------------------|
| app           | CLI 应用骨架：cobra 命令、选项/配置装配、优雅停机 | `pkg/app`            | [app.md](app.md)                           |
| methodlog/v1  | 方法级出入参日志中间件（HTTP/gRPC 通用）      | `pkg/methodlog/v1`   | [methodlog.md](methodlog.md)               |
| genyaml       | 从 Options 生成 YAML 配置模板的命令行工具骨架 | `pkg/genyaml`        | [genyaml.md](genyaml.md)                   |
| tools/codegen | 错误码代码/文档生成器（AST 解析注释）          | `tools/codegen`（命令行） | [../tools/codegen.md](../tools/codegen.md) |

### 服务端框架（generic_server）

| 库              | 职责                                                            | 导入路径                                                                                    | 文档                                                                   |
|----------------|---------------------------------------------------------------|-----------------------------------------------------------------------------------------|----------------------------------------------------------------------|
| framework/http | HTTP 五层框架：泛型 Endpoint/Middleware/Decoder/Encoder/Handler（gin） | `pkg/generic_server/framework/http/{endpoint,middleware,transport}/v1`                  | [generic_server_framework_http.md](generic_server_framework_http.md) |
| framework/grpc | gRPC 五层框架：一元/服务端流式 handler、服务注册与健康状态编排                        | `pkg/generic_server/framework/grpc/{endpoint,middleware,service_register,transport}/v1` | [generic_server_framework_grpc.md](generic_server_framework_grpc.md) |
| server         | 通用服务端生命周期：BizHooks、状态机、RA 注册、健康探测、优雅停机                        | `pkg/generic_server/server`（含 options、logger）                                           | [generic_server_server.md](generic_server_server.md)                 |

### 客户端 SDK（client-sdk）

| 库              | 职责                                    | 导入路径                            | 文档                                                     |
|----------------|---------------------------------------|---------------------------------|--------------------------------------------------------|
| grpc/v1        | 领域类型 gRPC 客户端：pb 编解码注入、流式透明聚合、中间件     | `pkg/client-sdk/grpc/v1`        | [client_sdk_grpc.md](client_sdk_grpc.md)               |
| grpc_health/v1 | gRPC 健康检查客户端：Check/Watch、Consul 发现入口  | `pkg/client-sdk/grpc_health/v1` | [client_sdk_grpc_health.md](client_sdk_grpc_health.md) |
| http/v1        | 领域类型 HTTP 客户端：JSON 内置编解码、NilBody、路径参数 | `pkg/client-sdk/http/v1`        | [client_sdk_http.md](client_sdk_http.md)               |

## 阅读路线建议

**路线 A：构建 HTTP 服务端**

1. [errors](errors.md) + [log](log.md)：错误码与日志是一切库的地基；
2. [core](core.md)：统一响应格式；
3. [generic_server_framework_http](generic_server_framework_http.md)：五层
   抽象与 gin handler 管线；
4. [generic_server_server](generic_server_server.md)：BizHooks 与生命周期；
5. [app](app.md)：命令行入口装配；
6. 对照 [examples/apiserver](../examples/index.md) 通读落地代码。

**路线 B：构建 gRPC 服务端**

路线 A 的 1、2 步后，改读
[generic_server_framework_grpc](generic_server_framework_grpc.md)（重点流式
分块协议与服务注册编排）→ [generic_server_server](generic_server_server.md)
（gRPC 侧 hooks/ping/RA）→ 对照 examples/grpcserver。

**路线 C：构建客户端 SDK**

1. 对端契约：先读路线 A/B 的 framework 篇，理解 Encoder/Decoder 对偶关系；
2. [client_sdk_http](client_sdk_http.md) 或 [client_sdk_grpc](client_sdk_grpc.md)；
3. 探活需求加读 [client_sdk_grpc_health](client_sdk_grpc_health.md)；
4. 对照 examples/httpclient、examples/grpcclient 的四/五层组织。

**路线 D：错误码工程化**

[errors](errors.md) → [../tools/codegen.md](../tools/codegen.md) →
[examples/code](../examples/index.md)（注释规范、go:generate、生成产物）。

## 相邻文档

- [examples 总览](../examples/index.md)：5 个可运行示例与测试指引；
- [tools/codegen](../tools/codegen.md)：命令行参数全表与注释解析规则；
- [错误码表（生成产物）](../examples/error_code_generated.md)；
- [提交 scope 规范](../../../devel/zh-CN/scope.md)：贡献前必读。

## 文档约定

- 每篇统一结构：定位 → 导入 → 概念/架构 → API 清单 → 配置项 → 示例 →
  协作关系 → 注意事项；
- 未实现/不确定能力标注「待确认」；与 pkg 内旧 README 冲突处以代码为准，
  并标注「旧 README 待修正」；
- 文档中的测试命令由使用者自行执行，编写过程仅做源码静态核实。