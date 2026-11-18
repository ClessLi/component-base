# Changelog — pkg/methodlog/v1

本包遵循语义化版本。以下变更按 **minor** 级别发布（行为修复 + 加法 API，无破坏性变更）。

## [Unreleased] (minor)

### Fixed

- **logger 陈旧问题（行为修复）**：移除包加载时（init）对
  `log/v1` 各级 logger 的一次性预绑定，改为每次 `Do` 的 defer 阶段惰性解析
  （`logV1.K()` / `logV1.V(-2).KitLogger()` / `logV1.V(-4).KitLogger()`，每个
  `Do` 解析一次）。宿主进程在 init 之后调用 `log.Init` 重新初始化日志（如把输出
  从 stdout/stderr 切换到文件）后，后续 `Do` 的日志将写入新目的地。
  **注意**：日志输出目的地将跟随宿主 `log.Init` 变化，这是期望的行为修复；
  依赖旧目的地（包加载时的默认输出）的部署需检查宿主日志配置。

### Added

- **`ResultWithError(builder func(err error) string) *Builder[M]`**：结果文案
  闭包直接接收 `Do` 自身的最终命名返回值 err（成功或 action panic 时为 nil），
  无需在 `Action` 内给外层捕获变量赋值。与 `Result` 共用同一结果槽位，
  多次设置后者覆盖前者。
- **`WithLogger(logger kitlog.Logger) Option`**：为本次 `Do` 注入固定 logger，
  覆盖惰性解析，所有级别（含 panic 记录）均写入注入 logger，用于测试捕获。
  nil logger 被忽略，仍走惰性解析。
- **`callsite` 日志字段（输出内容加法，minor）**：`New` 构造时捕获业务调用点，
  每条日志（含 panic 即时记录与最终记录）新增 `callsite` 字段，短格式
  `包目录/文件名:行号`，与 zap 自带的 `caller` 键不冲突。背景：后端 `caller`
  字段落在组件内部（Do 的 defer 闭包），不指向业务代码；定位业务请优先用
  `callsite` 或 `method` 字段。公开 API 与旧调用方零改动，仅日志输出多一个
  字段；若业务再封装一层 `New`，`callsite` 指向封装层调用行。

### Compatibility

- `New` / `Action` / `AddInfos` / `Result` / `Do` / `WithFieldExtractor` /
  `FieldExtractor` / `HTTPFieldExtractor` / `GRPCFieldExtractor` /
  `WithClientIP` / `ErrActionNotSet` 的签名与既有语义完全保留；
  现有 `Result(func() string)` 调用方零改动编译通过、行为不回归。
- panic 路径既有语义不变：panic 记录经 panic 级 logger 立即输出；默认
  zap PanicLevel logger 写完日志后自身再次 panic 的桥接行为原样保留。

### Documentation

- `Result` 的 doc comment 与包文档显式声明闭包契约：闭包在 `Do` 的 defer 阶段
  执行，捕获调用方变量时必须在 `Action` 内先赋值；推荐需要按最终错误分支文案
  时使用 `ResultWithError`。
- 新增 README.md（用法、契约、示例）。
- 包文档与 README 澄清 `caller` 语义：日志后端的 `caller` 字段为组件内部
  位置（当前为 Do 的 defer 闭包，builder.go；panic 即时记录路径可能为
  runtime 帧），不指向业务代码；业务定位使用 `callsite` / `method` /
  `stacktrace` 字段。

### Known Issues (out of scope)

- `log/v1` 的 `Init` 持锁写全局 logger，而 `K()` / `V()` 无锁读，为既有竞态，
  本次不扩大修复。**宿主应在开始并发日志之前完成 `log.Init`**。