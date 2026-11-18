# methodlog/v1

方法执行日志中间件：以流式 API 包装业务方法，自动输出方法名、耗时、结果文案、
错误与 panic 记录，并按 info / error / panic 自动分级。支持 HTTP 与 gRPC 的
上下文字段提取（客户端 IP 等）。

## 基本用法

```go
err := methodlog.New(ctx, handler).
    Action(func(h Handler) error {
        return h.ServeHTTP(w, r)
    }).
    Result(func() string {
        return "request processed"
    }).
    Do()
```

带协议字段提取：

```go
err := methodlog.New(ctx, handler, methodlog.WithFieldExtractor(methodlog.HTTPFieldExtractor)).
    Action(func(h Handler) error {
        return h.ServeHTTP(w, r)
    }).
    Do()
```

## Result 闭包契约（重要）

`Result` / `ResultWithError` 的闭包在 `Do` 的 defer 阶段执行，**此时 `Do` 尚未返回**，
调用方 `err = ...Do()` 的赋值还没有发生。因此：

- `Result(func() string)` 闭包若捕获调用方变量，该变量**必须在 `Action` 内先赋值**，
  否则闭包恒读到旧值（通常是 nil）：

```go
var u *User
err := methodlog.New(ctx, getUser).
    Result(func() string {
        if u != nil {
            return fmt.Sprintf("user %s", u.Name)
        }
        return "not found"
    }).
    Action(func(m func(context.Context, int64) (*User, error)) error {
        var e error
        u, e = m(ctx, id) // 必须在 Action 内赋值，闭包才能看到
        return e
    }).
    Do()
```

- 需要按最终错误分支结果文案时，**推荐使用 `ResultWithError`**：它直接接收 `Do`
  自身的最终返回值，`Action` 内无需给外层变量赋值：

```go
err := methodlog.New(ctx, method).
    Action(func(m MethodType) error {
        return m(args) // 直接返回即可
    }).
    ResultWithError(func(err error) string {
        if err != nil {
            return fmt.Sprintf("request failed: %v", err)
        }
        return "request processed"
    }).
    Do()
```

`Result` 与 `ResultWithError` 共用同一结果槽位，多次设置时后者覆盖前者。
action panic 时 `ResultWithError` 收到 nil（panic 由 panic 级日志单独记录）。

## WithLogger：注入固定 logger

`WithLogger` 为本次 `Do` 注入固定 logger，覆盖默认的全局日志解析，
所有级别（含 panic 记录）都写入注入的 logger，常用于测试捕获：

```go
err := methodlog.New(ctx, handler, methodlog.WithLogger(captureLogger)).
    Action(func(h Handler) error {
        return h.ServeHTTP(w, r)
    }).
    Do()
```

## 日志输出目的地跟随宿主 log.Init

各级 logger 在**每次 `Do` 时惰性解析**（经 `log/v1` 的 `K()` / `V(-2)` / `V(-4)`），
不再于包加载时一次性绑定。宿主进程在启动阶段调用 `log.Init` 重新初始化日志
（例如把输出从 stdout/stderr 切换到文件）后，后续 `Do` 的日志会写入新目的地。

并发注意：`log/v1` 的 `Init` 持锁替换全局 logger，而 `K()` / `V()` 无锁读取，
这是 `log/v1` 的既有竞态，不在本包修复范围内。**宿主应在开始并发日志之前完成
`log.Init`**（例如在 main/启动阶段一次性初始化）。

## 调用位置：caller 与 callsite

日志后端（zap）自带的 `caller` 字段是**组件内部位置**：日志在 `Do` 的 defer
闭包中经多层包装写出，`caller` 落在本包内部（当前为 Do 的 defer 闭包，
builder.go；panic 即时记录路径则可能落在 runtime 帧），**不指向业务代码**。

为此，`New` 在构造时捕获业务调用点，每条日志（含 panic 即时记录与最终记录）
都携带 `callsite` 字段，短格式 `包目录/文件名:行号`（如
`middleware/middleware.go:42`），与 zap 的 `caller` 键不冲突：

```json
{"method":"...GetClaimDetail-fm","callsite":"middleware/middleware.go:42", ...}
```

定位业务代码请优先使用 `callsite` 或 `method`（方法全名）字段；panic 日志另带
`stacktrace` 字段可查完整调用栈。注意：若业务再封装一层 `New`，`callsite`
指向封装层的调用行（所有调用点捕获方案的共同局限）。宿主日志配置开启
`DisableCaller` 时输出仅保留 `callsite`，更干净。

## panic 语义

action panic 时，panic 记录经 panic 级 logger 立即输出。默认 panic 级 logger
为 zap PanicLevel，zap 在写完日志后会再次 panic（go-kit/zap 桥接的既有行为），
该二次 panic 会向外传播；本包保留此既有语义。使用 `WithLogger` 注入非 panic 级
logger 时，`Do` 吞掉 action panic 并返回 nil。

## 测试

```shell
go test -race ./pkg/methodlog/...
```