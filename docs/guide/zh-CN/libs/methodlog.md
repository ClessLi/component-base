# methodlog/v1：方法级执行日志中间件

## 库定位与适用场景

`pkg/methodlog/v1` 以流式（fluent）API 包装任意业务方法，自动输出方法名、耗时、结果文案、
错误与 panic 记录，并按 info / error / panic 自动分级；支持 HTTP 与 gRPC 的上下文字段提取
（客户端 IP 等）。

适用场景：

- 服务层/中间件层方法需要统一的"入参-结果-耗时-错误"结构化日志；
- 需要 panic 恢复与即时 panic 日志；
- 测试中需要捕获日志输出做断言（`WithLogger` 注入）。

包内 [README](../../../../pkg/methodlog/v1/README.md) 与
[CHANGELOG](../../../../pkg/methodlog/v1/CHANGELOG.md) 为最新中文资料，本文按统一结构整理并
补充 API 清单。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import methodlog "github.com/ClessLi/component-base/pkg/methodlog/v1"
```

## 核心概念

- **Builder[M]**：泛型流式构造器，`M` 为被包装方法的签名类型。`New(ctx, method, opts...)`
  创建时捕获业务调用点（`callsite` 字段）。
- **Action**：`func(method M) error`，在 `Do` 内执行，负责真正调用目标方法。
- **Result / ResultWithError**：结果文案闭包，在 `Do` 的 defer 阶段执行；两者共用同一结果槽位，
  后设置者覆盖前者。
- **AddInfos**：追加日志键值对闭包，多次调用按序组合（与 Result 的覆盖语义不同）。
- **自动分级**：成功 → info（`logV1.K()`）；返回错误 → error（`logV1.V(-2)`）；
  panic → panic（`logV1.V(-4)`），panic 记录即时输出，最终记录仍会 emit。
- **惰性 logger 解析**：各级 logger 在每次 `Do` 时解析，输出目的地跟随宿主 `log.Init`。

## 关键类型与函数清单

### builder.go

| 签名                                                                                 | 说明                                              |
|------------------------------------------------------------------------------------|-------------------------------------------------|
| `func New[M any](ctx context.Context, method M, opts ...Option) *Builder[M]`       | 创建构造器并捕获 callsite                               |
| `func (b *Builder[M]) Do() (err error)`                                            | 执行 action 并输出日志；未设置 Action 返回 `ErrActionNotSet` |
| `func (b *Builder[M]) Action(action Action[M]) *Builder[M]`                        | 设置执行闭包                                          |
| `type Action[M any] func(method M) error`                                          | 执行闭包签名                                          |
| `func (b *Builder[M]) Result(builder func() string) *Builder[M]`                   | 结果文案闭包（无参）                                      |
| `func (b *Builder[M]) ResultWithError(builder func(err error) string) *Builder[M]` | 结果文案闭包，接收 Do 的最终返回值                             |
| `func (b *Builder[M]) AddInfos(builder func() []interface{}) *Builder[M]`          | 追加键值对闭包，可多次组合                                   |
| `var ErrActionNotSet`                                                              | 未设置 Action 时 Do 返回的错误                           |

### option.go

| 签名                                                         | 说明                      |
|------------------------------------------------------------|-------------------------|
| `type Option func(*formatter)`                             | 构造器选项（内部作用于 formatter）  |
| `func WithFieldExtractor(extractor FieldExtractor) Option` | 注册上下文字段提取器              |
| `func WithLogger(logger kitlog.Logger) Option`             | 注入固定 logger，覆盖所有级别的惰性解析 |

### extractors.go

| 签名                                                                  | 说明                                                            |
|---------------------------------------------------------------------|---------------------------------------------------------------|
| `type FieldExtractor func(ctx context.Context) []interface{}`       | 字段提取器签名                                                       |
| `var HTTPFieldExtractor`                                            | 提取 `clientIp`（来自 `WithClientIP` 写入的 context 值，缺省 `"unknown"`） |
| `var GRPCFieldExtractor`                                            | 提取 `clientIp`（来自 gRPC `peer.FromContext`，缺省 `"unknown"`）      |
| `func WithClientIP(ctx context.Context, ip string) context.Context` | HTTP 中间件向 context 写入客户端 IP                                    |

### 日志字段（formatter.go）

每条日志携带：`method`（方法全名或类型名）、`callsite`（业务调用点，`包目录/文件.go:行号`）、
`took`（耗时，`time.Duration`）、提取器字段（如 `clientIp`）、`AddInfos` 字段、
`result`（结果文案）、`error`（错误）；panic 即时记录另带 `panic` 值。

## 配置项说明

无 Options 结构，行为由 `Option` 决定：

| 选项                   | 默认                                                     | 说明                         |
|----------------------|--------------------------------------------------------|----------------------------|
| `WithFieldExtractor` | 无提取器                                                   | 可多次传入，按序执行                 |
| `WithLogger`         | 惰性解析全局 log/v1                                          | 注入后所有级别（含 panic）写入该 logger |
| 结果文案                 | 成功且未设置 Result 时默认 `"operation completed successfully"` | 错误路径无默认文案                  |

## 最小可运行示例

```go
err := methodlog.New(ctx, handler).
Action(func (h Handler) error {
return h.ServeHTTP(w, r)
}).
Result(func () string {
return "request processed"
}).
Do()
```

带 HTTP 字段提取：

```go
err := methodlog.New(ctx, handler, methodlog.WithFieldExtractor(methodlog.HTTPFieldExtractor)).
Action(func (h Handler) error {
return h.ServeHTTP(w, r)
}).
Do()
```

## 进阶示例

### 服务层中间件（真实示例）

改写自 [examples/apiserver/middleware.go](../../../../examples/apiserver/middleware.go)：

```go
func (l *loggingUserService) Get(ctx context.Context, id string) (*apiv1.User, error) {
var u *apiv1.User
err := methodlog.New(ctx, l.next.Get, methodlog.WithFieldExtractor(methodlog.HTTPFieldExtractor)).
AddInfos(func () []interface{} {
return []interface{}{"id", id}
}).
Result(func () string {
if u != nil {
return fmt.Sprintf("user %s (%s)", u.Name, u.ID)
}
return "not found"
}).
Action(func (m func (context.Context, string) (*apiv1.User, error)) error {
var e error
u, e = m(ctx, id) // 必须在 Action 内赋值，Result 闭包才能看到
return e
}).
Do()
return u, err
}
```

HTTP 侧需先在 gin 中间件里传播客户端 IP（同文件 `clientIPMiddleware`）：

```go
ctx := methodlog.WithClientIP(c.Request.Context(), c.ClientIP())
c.Request = c.Request.WithContext(ctx)
```

gRPC 侧改用 `methodlog.GRPCFieldExtractor`，见
[examples/grpcserver/middleware.go](../../../../examples/grpcserver/middleware.go)。

### ResultWithError：按最终错误分支文案

```go
err := methodlog.New(ctx, method).
Action(func (m MethodType) error {
return m(args) // 直接返回即可，无需给外层变量赋值
}).
ResultWithError(func (err error) string {
if err != nil {
return fmt.Sprintf("request failed: %v", err)
}
return "request processed"
}).
Do()
```

### WithLogger：测试捕获

```go
err := methodlog.New(ctx, handler, methodlog.WithLogger(captureLogger)).
Action(func(h Handler) error { return h.ServeHTTP(w, r) }).
Do()
```

## 与其他库的协作关系

- **[log/v1](log.md)**：info/error/panic 分别经 `K()`/`V(-2)`/`V(-4)` 的 go-kit 形态输出；
  宿主 `log.Init` 后后续 `Do` 自动写入新目的地。
- **[generic_server framework](generic_server_framework_http.md)**：HTTP/gRPC 示例的
  logging 中间件均以本包实现。
- **[errors](errors.md)**：`ErrActionNotSet` 经 errors 包创建；action 返回的错误原样透传。

## 注意事项、限制与常见问题

1. **Result 闭包契约**：闭包在 `Do` 的 defer 中执行，此时 `err = ...Do()` 赋值尚未发生；
   捕获的调用方变量必须在 Action 内赋值，否则恒读旧值。按错误分支文案优先用 `ResultWithError`。
2. **panic 语义**：默认 panic 级 logger 为 zap PanicLevel，写完日志后 zap 会再次 panic 并向外
   传播（既有语义）；`WithLogger` 注入非 panic 级 logger 时，`Do` 吞掉 action panic 并返回 nil，
   `ResultWithError` 收到 nil。
3. **caller 与 callsite**：zap 自带 `caller` 指向本包内部（Do 的 defer 闭包），定位业务代码
   请用 `callsite` 或 `method` 字段；业务再封装一层 `New` 时 callsite 指向封装层调用行。
4. **初始化时机**：log/v1 的 `Init` 持锁写、`K()`/`V()` 无锁读，宿主应在并发日志开始前完成
   `log.Init`。
5. **Result 与 AddInfos 语义差异**：Result/ResultWithError 多次设置后者覆盖前者；
   AddInfos 多次调用按序组合全部执行。
6. **测试**：`go test -race ./pkg/methodlog/...`（由使用者自行执行）。