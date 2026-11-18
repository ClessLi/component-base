# errors：命名空间隔离的错误处理库

## 库定位与适用场景

`pkg/errors` 提供带上下文、堆栈跟踪与**命名空间隔离错误码**的错误处理能力。它克隆自
[`github.com/marmotedu/errors`](https://github.com/marmotedu/errors) 并在本仓库内做了本地化扩展
（命名空间注册表、`DefaultCoder` 构造器、`IsCode`/`GetErrorNamespace` 等），用于减少外部依赖并支持
多项目错误码共存。

适用场景：

- 需要在错误链上附加上下文消息与调用堆栈，同时保留原始错误；
- 需要为 API 定义业务错误码，并映射到 HTTP 状态码与用户可见文案；
- 多个项目/服务共用一个进程或 SDK，错误码需要按命名空间隔离、互不冲突；
- 需要聚合多个 goroutine 或多个步骤产生的错误。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import "github.com/ClessLi/component-base/pkg/errors"
```

## 核心概念

- **错误链**：`New`/`Errorf` 创建带堆栈的基础错误；`Wrap`/`Wrapf`/`WithMessage`/`WithMessagef`/
  `WithStack` 在保留原因（cause）的前提下附加消息或堆栈；`Cause` 逐层取出根因。
- **Coder**：错误码元信息接口，包含整型码、HTTP 状态码、用户安全文案、参考文档链接、命名空间。
- **命名空间注册表**：全局 `namespace → (code → Coder)` 两级映射（`code.go` 中的 `nsCodes`），
  由 `Register` 写入（并发安全，`sync.RWMutex` 保护）。同一命名空间内码值唯一，跨命名空间可重复。
- **withCode 错误**：`WithCode`/`WrapC` 创建的错误携带 `(namespace, code)`，格式化输出时按注册表
  解析出用户安全文案；未注册的码回退到内置 `unknownCoder`（code=1，HTTP 500，命名空间 `base`）。
- **Go 1.13 兼容**：所有包装类型实现 `Unwrap`；`Is`/`As`/`Unwrap` 直接转发标准库（`go113.go`）。

## 关键类型与函数清单

### 错误创建与包装（errors.go）

| 签名                                                                                            | 说明                             |
|-----------------------------------------------------------------------------------------------|--------------------------------|
| `func New(message string) error`                                                              | 创建错误并记录调用点堆栈                   |
| `func Errorf(format string, args ...interface{}) error`                                       | 格式化创建错误并记录堆栈                   |
| `func WithStack(err error) error`                                                             | 附加堆栈；err 为 nil 时返回 nil         |
| `func Wrap(err error, message string) error`                                                  | 附加消息与堆栈；err 为 nil 时返回 nil      |
| `func Wrapf(err error, format string, args ...interface{}) error`                             | 附加格式化消息与堆栈                     |
| `func WithMessage(err error, message string) error`                                           | 仅附加消息，不记录堆栈                    |
| `func WithMessagef(err error, format string, args ...interface{}) error`                      | 仅附加格式化消息                       |
| `func WithCode(namespace string, code int, format string, args ...interface{}) error`         | 创建带命名空间与错误码的错误                 |
| `func WrapC(err error, namespace string, code int, format string, args ...interface{}) error` | 包装已有错误并附加错误码；err 为 nil 时返回 nil |
| `func Cause(err error) error`                                                                 | 沿 `Cause() error` 接口逐层取根因      |

### 错误码（code.go）

| 签名                                                                                                               | 说明                                                        |
|------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------|
| `type Coder interface { HTTPStatus() int; String() string; Reference() string; Code() int; Namespace() string }` | 错误码元信息接口                                                  |
| `func DefaultCoder(code int, httpStatus int, message string, ref string, namespace string) Coder`                | Coder 便捷构造器                                               |
| `func Register(coder Coder)`                                                                                     | 注册错误码；code 为 0、命名空间为空、同命名空间重复注册均 panic                    |
| `func ParseCoder(err error) Coder`                                                                               | 将任意错误解析为 Coder；nil 返回 nil，非 withCode 或未注册码返回 unknownCoder |
| `func IsCode(err error, namespace string, code int) bool`                                                        | 判断错误（沿 withCode 的 cause 链）是否命中指定命名空间与码                    |
| `func GetErrorNamespace(err error) string`                                                                       | 返回错误的命名空间；非 withCode 错误返回 `"unknown"`                     |

### Go 1.13 兼容（go113.go）

| 签名                                            | 说明                    |
|-----------------------------------------------|-----------------------|
| `func Is(err, target error) bool`             | 转发标准库 `errors.Is`     |
| `func As(err error, target interface{}) bool` | 转发标准库 `errors.As`     |
| `func Unwrap(err error) error`                | 转发标准库 `errors.Unwrap` |

### 错误聚合（aggregate.go）

| 签名                                                                     | 说明                                     |
|------------------------------------------------------------------------|----------------------------------------|
| `type Aggregate interface { error; Errors() []error; Is(error) bool }` | 多错误聚合接口，支持 `errors.Is`，不支持 `errors.As` |
| `func NewAggregate(errlist []error) Aggregate`                         | 聚合错误切片；自动剔除 nil，空列表返回 nil              |
| `type Matcher func(error) bool`                                        | 错误匹配函数                                 |
| `func FilterOut(err error, fns ...Matcher) error`                      | 递归剔除命中任一 Matcher 的错误                   |
| `func Flatten(agg Aggregate) Aggregate`                                | 递归展平嵌套聚合                               |
| `func CreateAggregateFromMessageCountMap(m MessageCountMap) Aggregate` | 由消息计数表构建聚合（重复消息附 `(repeated N times)`） |
| `func Reduce(err error) error`                                         | 聚合仅含 1 个错误时返回该错误，0 个返回 nil             |
| `func AggregateGoroutines(funcs ...func() error) Aggregate`            | 并行执行函数并聚合所有非 nil 错误                    |
| `type MessageCountMap map[string]int`                                  | 消息出现次数表                                |
| `var ErrPreconditionViolated`                                          | 前置条件违反哨兵错误（标准库 `errors.New` 创建）        |

### 堆栈与集合（stack.go、sets.go）

| 签名                                       | 说明                                       |
|------------------------------------------|------------------------------------------|
| `type Frame uintptr`                     | 单个调用帧，实现 `fmt.Formatter` 与 `MarshalText` |
| `type StackTrace []Frame`                | 调用栈，实现 `fmt.Formatter`                   |
| `type String map[string]Empty`           | 字符串集合（聚合去重内部使用，也可独立使用）                   |
| `func NewString(items ...string) String` | 创建字符串集合                                  |

`String` 集合还提供 `Insert`/`Delete`/`Has`/`HasAll`/`HasAny`/`Difference`/`Union`/`Intersection`/
`IsSuperset`/`Equal`/`List`（排序）/`UnsortedList`/`PopAny`/`Len` 等方法，见 `sets.go`。

## 配置项说明

本包没有 Options 配置对象。行为由以下因素决定：

- 运行时注册表内容（`Register` 的调用时机与内容）；
- 格式化动词（见下节）；
- 内置 `unknownCoder`：code=1、HTTP 500、文案 `"An internal server error occurred"`、命名空间 `base`，
  在包 `init()` 中自动注册，**业务码禁止使用 0**（`Register` 会 panic）。

## 格式化输出

withCode 错误实现 `fmt.Formatter`（`format.go`），支持：

| 动词              | 输出                                                         |
|-----------------|------------------------------------------------------------|
| `%s` / `%v`     | 用户安全文案（注册表中 Coder 的 `String()`；未注册时回退内部消息）                 |
| `%-v`           | 文案 + 调用点坐标 + `(命名空间:码)`                                    |
| `%+v`           | 沿错误链展开每一层的内部消息、坐标与 `(命名空间:码)`                              |
| `%#v`           | JSON 数组，仅含 `error` 字段                                      |
| `%#-v` / `%#+v` | JSON 数组，含 `namespace`、`code`、`message`、`error`、`caller` 字段 |

普通错误（`New`/`Wrap` 等）的 `%+v` 输出消息与完整堆栈帧。

## 最小可运行示例

### 基础错误与包装

改写自 [examples/code/usage_example.go](../../../../examples/code/usage_example.go)：

```go
package main

import (
	"fmt"

	"github.com/ClessLi/component-base/pkg/errors"
)

func main() {
	err := errors.New("database connection failed")
	err = errors.Wrap(err, "timeout after 30s")
	err = errors.Wrap(err, "service unavailable")

	fmt.Printf("Full error: %v\n", err)  // service unavailable
	fmt.Printf("Root cause: %v\n", errors.Cause(err)) // database connection failed
}
```

### 手动定义并注册错误码

```go
package main

import (
	"fmt"

	"github.com/ClessLi/component-base/pkg/errors"
)

const ErrNotFound = 1001

func init() {
	errors.Register(errors.DefaultCoder(
		ErrNotFound, 404,
		"Resource not found. Check the identifier and try again",
		"https://docs.example.com/errors/1001",
		"myproject",
	))
}

func main() {
	err := errors.WithCode("myproject", ErrNotFound, "user %s not found", "12345")
	fmt.Println(err)                          // Resource not found. Check the identifier and try again
	fmt.Printf("%+v\n", err)                  // user 12345 not found - #0 [...坐标...] (myproject:1001) Resource not found...
	fmt.Println(errors.IsCode(err, "myproject", ErrNotFound)) // true

	coder := errors.ParseCoder(err)
	fmt.Println(coder.HTTPStatus())           // 404
	fmt.Println(coder.Reference())            // https://docs.example.com/errors/1001
}
```

## 进阶示例

### 配合 codegen 生成的包装函数（推荐方式）

实际项目推荐用 [codegen 工具](../tools/codegen.md) 从注释生成注册代码与包装函数，业务侧无需手写
namespace。完整可运行示例见 [examples/code](../../../../examples/code/)：

```go
// examples/code 包内（Namespace 与包装函数由 codegen 生成）
err := WithCode(ErrPageNotFound, "user %s not found", "12345")
fmt.Println(err) // Check the URL path and try again
fmt.Printf("%+v\n", err) // user 12345 not found - #0 (codeexample:100006) Check the URL path and try again

if IsCode(err, ErrPageNotFound) {
// 命中本命名空间下的 ErrPageNotFound
}
```

生成的 `WithCode(code int, format string, args ...interface{})` 内部转发
`errors.WithCode(Namespace, code, format, args...)`，见
[errors_generated.go](../../../../examples/code/errors_generated.go)。

### 错误聚合与并行收集

```go
agg := errors.AggregateGoroutines(
func () error { return errors.WithCode("myproject", ErrNotFound, "a") },
func () error { return nil },
func () error { return errors.WithCode("myproject", ErrDatabase, "b") },
)
if agg != nil {
fmt.Println(agg) // [文案A, 文案B]
}

// 剔除已知可忽略的错误
filtered := errors.FilterOut(agg, func(err error) bool {
return errors.IsCode(err, "myproject", ErrNotFound)
})
```

### 标准库 Is/As 兼容

```go
// withCode.Is 按 (namespace, code) 判等
target := errors.WithCode("myproject", ErrNotFound, "")
errors.Is(err, target) // true（同命名空间同码）

// As 到 *errors.Coder 可直接取出注册的 Coder
var coder errors.Coder
if errors.As(err, &coder) {
fmt.Println(coder.HTTPStatus())
}
```

## 与其他库的协作关系

- **[core](core.md)**：`core.WriteResponse` 用 `ParseCoder` 把任意错误映射为 HTTP 状态码与
  `ErrResponse` JSON 体，是 HTTP API 错误响应的标准出口。
- **[codegen](../tools/codegen.md)**：从错误码常量注释生成 `Register` 调用、`Namespace` 常量与
  包装函数，是本包错误码的推荐生产方式。
- **generic_server / client-sdk**：服务端编码器与客户端解码器围绕 `(code, message)` 错误响应契约
  协作，详见对应文档。

## 注意事项、限制与常见问题

1. **`Register` 的 panic 条件**：code 为 0（保留给 unknownCoder）、命名空间为空、同命名空间重复
   注册。建议在 `init()` 中注册，尽早暴露冲突。
2. **`IsCode` 只识别 withCode 链**：它沿 `*withCode` 的 cause 递归，但若错误先被
   `WithMessage`/`Wrap` 包装成其他类型，`IsCode` 对外层类型直接返回 false。需要穿透任意包装时
   使用 `errors.Is` + withCode 目标值，或先 `Cause` 再判断。
3. **`%v` 输出的是注册文案而非传入消息**：`WithCode` 的 format 消息只在 `%-v`/`%+v`/JSON 详情中
   可见；对外展示默认使用 Coder 的用户安全文案。
4. **`WithStack`/`Wrap` 对 withCode 的已知局限**：源码 TODO 标注，包装 withCode 时可能丢失部分
   原始消息上下文（`errors.go` 中 `WithStack`、`Wrap` 的 TODO 注释），调试时优先使用 `%+v`。
5. **Aggregate 不支持 `errors.As`**：聚合可能包含多个同类型错误，语义上无法确定取哪一个；
   请用 `Errors()` 遍历。
6. **旧 README 待修正**（[pkg/errors/README.md](../../../../pkg/errors/README.md)，保留英文原文不动）：
    - API 清单中的 `MustRegister`、`GetCode` 在当前代码中**不存在**；
    - `WithCode` 实际签名带 `namespace` 首参，README 未体现；
    - 自定义 Coder 示例缺少接口必需的 `Namespace() string` 方法；
    - 格式化示例输出为 `(#100102)` 旧格式，当前实际输出为 `(命名空间:码)`，如 `(base:100102)`。