# log/v1：基于 zap 的日志抽象层

## 库定位与适用场景

`pkg/log/v1` 是基于 `go.uber.org/zap` 封装的生产级日志包，提供统一的 `Logger`/`InfoLogger` 接口、
全局 logger、V 分级、Context 传递，并桥接标准库 `log`、`klog`、`logrus`、go-kit log 等生态。

适用场景：

- 应用需要可配置（级别、格式、输出目的地、颜色）的结构化日志；
- 依赖库使用 klog/logrus/标准库 log，希望统一收敛到同一日志后端与格式；
- 需要在请求上下文中携带 logger 或 requestID/username 字段；
- 需要 go-kit 风格的 `kitlog.Logger`（如 methodlog、client-sdk 中间件）。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import log "github.com/ClessLi/component-base/pkg/log/v1"
```

子包（日志生态桥接）：

```go
import (
"github.com/ClessLi/component-base/pkg/log/v1/klog"         // klog → zap
"github.com/ClessLi/component-base/pkg/log/v1/logrus"       // logrus → zap（hook）
"github.com/ClessLi/component-base/pkg/log/v1/distribution" // 兼容 logrus/std log 风格 API
)
```

## 核心概念

- **Logger / InfoLogger 接口**（`log.go`）：`Logger` 内嵌 `InfoLogger`，提供
  Debug/Info/Warn/Error/Panic/Fatal 六个级别，每级别三种形态：
    - `Xxx(msg string, fields ...Field)`：强类型字段，性能最好；
    - `Xxxf(format string, v ...interface{})`：printf 风格；
    - `Xxxw(msg string, keysAndValues ...interface{})`：松散键值对（底层反射，低频场景使用）。
- **全局 std**：包加载时以默认 Options 创建；`Init(opts ...*Options)` 持锁整体替换。
- **多 Options tee 输出**：`New(opts...)`/`Init(opts...)` 传入多份 Options 时，第一份为主配置，
  其余各自构建 core 并以 `zapcore.NewTee` 合并，实现"全量日志 + 错误日志分文件"等分流。
- **V 分级**：`V(level)` 映射到 zap 级别 `-level`。`V(0)`=Info、`V(1)`=Debug；负数映射到更高级别
  （`V(-1)`=Warn、`V(-2)`=Error、`V(-4)`=Panic，methodlog 即依赖此映射）。级别未启用时返回
  no-op logger。
- **Context**：`WithContext`/`FromContext` 在 context 中存取 Logger；`L(ctx)` 额外提取
  `KeyRequestID`（`"requestID"`）与 `KeyUsername`（`"username"`）两个 context 值附加为字段。
- **生态桥接**：`New` 时自动调用 `klog.InitLogger` 并 `zap.RedirectStdLog`，klog 与标准库 log
  输出自动汇入 zap。

## 关键类型与函数清单

### 接口与全局管理（log.go）

| 签名                                                                                                                                                                                                                                          | 说明                                              |
|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------|
| `type InfoLogger interface { Info/Infof/Infow; KitLogger() kitlog.Logger; Enabled() bool }`                                                                                                                                                 | 非错误日志能力                                         |
| `type Logger interface { InfoLogger; Debug*/Warn*/Error*/Panic*/Fatal*; V(int) InfoLogger; Write([]byte) (int, error); WithValues(...interface{}) Logger; WithName(string) Logger; WithContext(context.Context) context.Context; Flush() }` | 完整日志能力                                          |
| `func Init(opts ...*Options)`                                                                                                                                                                                                               | 持锁重建并替换全局 logger                                |
| `func New(opts ...*Options) *zapLogger`                                                                                                                                                                                                     | 按 Options 创建 logger；多份 Options 走 tee；构建失败 panic |
| `func StdLogger() *zapLogger`                                                                                                                                                                                                               | 返回全局 logger                                     |
| `func NewLogger(l *zap.Logger) Logger`                                                                                                                                                                                                      | 用已有 zap.Logger 包装为 Logger                       |
| `func ZapLogger() *zap.Logger`                                                                                                                                                                                                              | 取出全局底层 zap.Logger（供桥接子包使用）                      |
| `func Flush()`                                                                                                                                                                                                                              | 刷出缓冲日志，进程退出前调用                                  |

### 包级日志函数（log.go，均作用于全局 std）

| 签名                                                                                       | 说明                                |
|------------------------------------------------------------------------------------------|-----------------------------------|
| `func Debug/Info/Warn/Error/Panic/Fatal(msg string, fields ...Field)`                    | 强类型字段日志                           |
| `func Debugf/Infof/Warnf/Errorf/Panicf/Fatalf(format string, v ...interface{})`          | 格式化日志                             |
| `func Debugw/Infow/Warnw/Errorw/Panicw/Fatalw(msg string, keysAndValues ...interface{})` | 键值对日志                             |
| `func V(level int) InfoLogger`                                                           | V 分级 InfoLogger                   |
| `func WithValues(keysAndValues ...interface{}) Logger`                                   | 派生带固定字段的子 logger                  |
| `func WithName(s string) Logger`                                                         | 派生带名字段（logger 键）的子 logger         |
| `func L(ctx context.Context) *zapLogger`                                                 | 派生带 requestID/username 字段的 logger |
| `func K() kitlog.Logger`                                                                 | 全局 logger 的 go-kit 形态（Info 级）     |
| `func StdErrLogger() *log.Logger` / `func StdInfoLogger() *log.Logger`                   | 标准库 log 形态（Error/Info 级）          |
| `func CheckIntLevel(level int32) bool`                                                   | klog 桥接用级别检查（<5 视为 Info，否则 Debug） |

### Context（context.go）

| 签名                                                      | 说明                                                     |
|---------------------------------------------------------|--------------------------------------------------------|
| `func WithContext(ctx context.Context) context.Context` | 将全局 logger 存入 ctx                                      |
| `func FromContext(ctx context.Context) Logger`          | 从 ctx 取 logger；不存在时返回名为 `Unknown-Context` 的全局派生 logger |

### 字段与级别（types.go）

- `type Field = zapcore.Field`、`type Level = zapcore.Level`（别名）；
- 级别变量：`DebugLevel`、`InfoLevel`、`WarnLevel`、`ErrorLevel`、`PanicLevel`、`FatalLevel`；
- 字段构造器别名：`Any`、`Bool`、`Int`/`Int32`/`Int64`、`String`、`Strings`、`Err`（=zap.Error）、
  `Duration`、`Time`、`Reflect`、`Stack` 等（完整清单见 `types.go`）；
- 常用 context 键常量：`KeyRequestID = "requestID"`、`KeyUsername = "username"`。

### 桥接子包

| 签名                                                                       | 所在文件                   | 说明                                                                                      |
|--------------------------------------------------------------------------|------------------------|-----------------------------------------------------------------------------------------|
| `func InitLogger(zapLogger *zap.Logger)`                                 | klog/logger.go         | 将 klog 各级输出重定向到 zap；`New` 时自动调用                                                         |
| `func NewLogger(zapLogger *zap.Logger) *logrus.Logger`                   | logrus/logger.go       | 创建 logrus logger，输出丢弃、经 hook 写入 zap                                                     |
| `type Logger struct{...}` / `func NewLogger(logger *zap.Logger) *Logger` | distribution/logger.go | 兼容 logrus/std log 的 Print/Trace/Debug/Info/Warn/Error/Fatal/Panic 及 ln/f 变体、`WithError` |

## 配置项说明（Options）

`Options` 定义于 `options.go`，默认值取自 `NewOptions()`：

| 字段                  | 类型         | 默认值          | 说明                                                 |
|---------------------|------------|--------------|----------------------------------------------------|
| `Level`             | `string`   | `"info"`     | 最低输出级别（zap 级别文本：debug/info/warn/error/panic/fatal） |
| `Format`            | `string`   | `"console"`  | `console`（文本）或 `json`                              |
| `EnableColor`       | `bool`     | `false`      | console 格式下启用 ANSI 颜色级别                            |
| `DisableCaller`     | `bool`     | `false`      | 关闭 caller 字段                                       |
| `DisableStacktrace` | `bool`     | `false`      | 关闭 panic 级以上的堆栈记录                                  |
| `Development`       | `bool`     | `false`      | zap 开发模式（DPanic 行为、更激进的堆栈）                         |
| `OutputPaths`       | `[]string` | `["stdout"]` | 日志输出：stdout/stderr/文件路径，可多个                        |
| `ErrorOutputPaths`  | `[]string` | `["stderr"]` | zap 内部错误输出                                         |
| `Name`              | `string`   | `""`         | logger 名（输出为 `logger` 键）                           |

Options 方法：

| 签名                                              | 说明                                                                                                                                                                                      |
|-------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `func NewOptions() *Options`                    | 默认配置构造器                                                                                                                                                                                 |
| `func (o *Options) Validate() []error`          | 校验 Level 与 Format                                                                                                                                                                       |
| `func (o *Options) AddFlags(fs *pflag.FlagSet)` | 注册 `--log.level`、`--log.format`、`--log.enable-color`、`--log.disable-caller`、`--log.disable-stacktrace`、`--log.output-paths`、`--log.error-output-paths`、`--log.development`、`--log.name` |
| `func (o *Options) Build() error`               | 直接构建并替换 zap 全局 logger（`zap.ReplaceGlobals`），与 `New` 的全局 std 是两条路径                                                                                                                       |
| `func (o *Options) String() string`             | JSON 序列化                                                                                                                                                                                |

输出编码固定为：时间 `2006-01-02 15:04:05.000`、时长毫秒浮点、短 caller；JSON 键为
`message`/`level`/`timestamp`/`logger`/`caller`/`stacktrace`（`encoder.go`、`log.go`）。
内置采样：每秒前 100 条全量、之后每 100 条记 1 条（zap SamplingConfig，代码内固定，不可配置）。

## 最小可运行示例

改写自 [pkg/log/v1/example/example.go](../../../../pkg/log/v1/example/example.go)：

```go
package main

import (
	"context"

	log "github.com/ClessLi/component-base/pkg/log/v1"
)

func main() {
	opts := log.NewOptions()
	opts.Level = "debug"
	opts.EnableColor = true
	log.Init(opts)
	defer log.Flush()

	log.Debug("This is a debug message")
	log.Info("This is a info message", log.Int32("int_key", 10))
	log.Warnf("This is a formatted %s message", "warn")
	log.Errorw("Message printed with Errorw", "X-Request-ID", "fbf54504-64da-4088-9b86-67824a7fb508")

	log.V(1).Info("This is a V level message") // debug 级
}
```

运行完整示例：

```bash
go run ./pkg/log/v1/example
```

## 进阶示例

### 多目的地分流（全量 + 错误分文件）

```go
stdOpts := &log.Options{
Level: "debug", Format: "console", EnableColor: true, DisableCaller: true,
OutputPaths:      []string{"test.log", "stdout"},
ErrorOutputPaths: []string{"error.log", "stderr"},
}
errOpts := &log.Options{
Level: "error", Format: "console", EnableColor: true, DisableCaller: true,
OutputPaths:      []string{"error2.log"},
ErrorOutputPaths: []string{},
}
log.Init(stdOpts, errOpts) // 第二份 Options 的 core 以 tee 方式并入
defer log.Flush()
```

### 子 logger 与 Context 传递

```go
lv := log.WithValues("X-Request-ID", "7a7b9f24-4cae-4b2a-9464-69088b45b904")
lv.Infow("Info message printed with [WithValues] logger")

ctx := lv.WithContext(context.Background())
lc := log.FromContext(ctx)
lc.Info("Message printed with [WithContext] logger")

ln := lv.WithName("test")
ln.Info("Message printed with [WithName] logger")
```

### 桥接第三方日志库

```go
// klog：log.New/Init 已自动桥接，直接使用 klog 即可
// logrus：
logrusLogger := logrusbridge.NewLogger(log.ZapLogger()) // import .../pkg/log/v1/logrus
logrusLogger.WithField("component", "worker").Info("hello")

// 兼容 logrus/std log 风格 API：
dist := distribution.NewLogger(log.ZapLogger())
dist.Printf("listening on %s", addr)
```

## 与其他库的协作关系

- **[app](app.md) / [generic_server](generic_server_server.md)**：Options 通过 `AddFlags` 挂到命令行，
  服务启动阶段调用 `log.Init` 完成初始化。
- **[methodlog](methodlog.md)**：每次 `Do` 经 `K()`/`V(-2)`/`V(-4)` 惰性解析 info/error/panic logger，
  输出目的地跟随宿主 `log.Init`。
- **[core](core.md)**：`WriteResponse` 用 `log.Errorf("%#+v", err)` 记录错误详情。
- **client-sdk / generic_server framework**：go-kit 中间件日志经 `K()` 或 `KitLogger()` 输出。

## 注意事项、限制与常见问题

1. **初始化时机与并发**：`Init` 持锁替换全局 std，但 `K()`/`V()`/包级日志函数无锁读取 std，
   并发运行中重初始化存在既有竞态。**宿主应在开始并发日志之前（main/启动阶段）一次性完成 `Init`**。
2. **`New` 构建失败会 panic**：输出路径不可写等配置错误在初始化期即崩溃，属快速失败设计。
3. **V 负数级别**：接口注释写明"小于 0 非法"，但实现实际支持负数并映射到 Warn/Error/Panic，
   methodlog 依赖该行为；业务代码建议仅使用非负 V 级别。
4. **`Xxxw` 键值对必须成对且键为 string**：奇数参数或非 string 键会触发 zap DPanic
   （开发模式 panic，生产模式记录错误日志）。
5. **采样**：内置 100/100 采样不可配置，高频重复日志可能被丢弃；审计类日志需注意。
6. **`Options.Build` 与 `Init` 的区别**：`Build` 替换的是 zap 全局（`zap.L()`），`Init` 替换的是
   本包全局 std；两者不等价，常规使用选 `Init`。
7. **旧 README 待修正**（[pkg/log/v1/README.md](../../../../pkg/log/v1/README.md)，保留原文不动）：
    - 导入路径写作 `github.com/ClessLi/ansible-role-manager/pkg/log/v2`，实际应为
      `github.com/ClessLi/component-base/pkg/log/v1`；
    - 示例链接 `../example/example_v2.go` 失效，实际文件为 `example/example.go`；
    - README 未提及 `EnableCaller` 字段（当前 Options 无此字段，只有 `DisableCaller`）。