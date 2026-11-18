# app：命令行应用骨架

## 库定位与适用场景

`pkg/app` 基于 cobra + pflag + viper 封装命令行应用骨架：统一处理参数分组展示、配置文件加载、
环境变量绑定、版本信息、选项校验与启动回调，让业务只关注 `RunFunc`。

适用场景：

- 构建带 `--config`/`--version`/分组 `--help` 的服务端或工具类 CLI；
- 需要"命令行参数 + 配置文件 + 环境变量"三源合一的配置加载；
- 需要多级子命令（`Command`/`AddCommand`）。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import "github.com/ClessLi/component-base/pkg/app"
```

## 核心概念

- **App**：应用主结构，经 `NewApp(name, basename, opts...)` 创建，内部构建 cobra 根命令；
  `Run()` 执行并在出错时打印红色错误并 `os.Exit(1)`。
- **CliOptions**：业务选项接口（`Flags() cliflag.NamedFlagSets` + `Validate() []error`）。
  可选实现 `CompletableOptions`（`Complete() error`，补全默认值）、`PrintableOptions`
  （`String() string`，启动时打印配置）。启动流程依次执行 Complete → Validate → 打印 → RunFunc。
- **配置加载**（`config.go`）：`--config/-c` 指定文件（JSON/TOML/YAML/HCL/properties）；
  未指定时按 `./<basename>.<ext>` 与 `~/.<basename首段>/<basename>.<ext>` 搜索。
  环境变量前缀为 basename 大写（`-` 换 `_`），键中 `.`/`-` 映射为 `_`。
  参数经 `viper.BindPFlags` + `viper.Unmarshal` 回填到 CliOptions（依赖 `mapstructure` 标签）。
- **平台差异**（`cmd_linux.go`/`cmd_windows.go`）：`FormatBasename` 在 Linux 原样返回；
  Windows 转小写并去掉 `.exe` 后缀。
- **帮助与规范化**：分组 help/usage 输出（marmotedu cliflag）；`WordSepNormalizeFunc` 把
  flag 名中的 `_` 归一为 `-`；子命令自带 `--help/-H`。

## 关键类型与函数清单

### App 与选项（app.go）

| 签名                                                                         | 说明                                      |
|----------------------------------------------------------------------------|-----------------------------------------|
| `func NewApp(name, basename string, opts ...Option) *App`                  | 创建应用并构建 cobra 命令                        |
| `func (a *App) Run()`                                                      | 执行应用；失败打印错误并退出码 1                       |
| `func (a *App) Command() *cobra.Command`                                   | 取出内部 cobra 命令（供 genyaml 等复用）            |
| `func (a *App) AddCommand(cmd *Command)` / `AddCommands(cmds ...*Command)` | 挂载子命令（须在 NewApp 前经 Option 或创建后调用，见注意事项） |
| `type Option func(*App)`                                                   | 函数式选项                                   |
| `func WithOptions(opt CliOptions) Option`                                  | 启用命令行/配置文件选项                            |
| `func WithRunFunc(run RunFunc) Option`                                     | 设置启动回调                                  |
| `type RunFunc func(basename string) error`                                 | 启动回调签名                                  |
| `func WithDescription(desc string) Option`                                 | 应用描述（cobra Long）                        |
| `func WithSilence() Option`                                                | 静默模式：不打印启动/配置/版本信息                      |
| `func WithNoVersion() Option`                                              | 去掉 `--version`                          |
| `func WithNoConfig() Option`                                               | 去掉 `--config` 与配置文件加载                   |
| `func WithValidArgs(args cobra.PositionalArgs) Option`                     | 自定义位置参数校验                               |
| `func WithDefaultValidArgs() Option`                                       | 默认拒绝任何非 flag 位置参数                       |

### 选项接口（options.go）

| 签名                                                                                | 说明                   |
|-----------------------------------------------------------------------------------|----------------------|
| `type CliOptions interface { Flags() cliflag.NamedFlagSets; Validate() []error }` | 必需接口                 |
| `type CompletableOptions interface { Complete() error }`                          | 可选：补全                |
| `type PrintableOptions interface { String() string }`                             | 可选：打印配置              |
| `type ConfigurableOptions interface { ApplyFlags() []error }`                     | 已定义但当前启动流程未调用（待确认用途） |

### 子命令（cmd.go）

| 签名                                                                    | 说明      |
|-----------------------------------------------------------------------|---------|
| `func NewCommand(usage, desc string, opts ...CommandOption) *Command` | 创建子命令   |
| `func WithCommandOptions(opt CliOptions) CommandOption`               | 子命令选项   |
| `func WithCommandRunFunc(run RunCommandFunc) CommandOption`           | 子命令回调   |
| `type RunCommandFunc func(args []string) error`                       | 子命令回调签名 |
| `func (c *Command) AddCommand(cmd *Command)` / `AddCommands(...)`     | 子命令嵌套   |

### 其他（flag.go、help.go、config.go、cmd_*.go）

| 签名                                                                              | 说明            |
|---------------------------------------------------------------------------------|---------------|
| `func WordSepNormalizeFunc(_ *pflag.FlagSet, name string) pflag.NormalizedName` | `_` → `-` 归一化 |
| `func FormatBasename(basename string) string`                                   | 平台相关可执行名格式化   |

## 配置项说明

App 自身无 Options 结构，行为由函数式 Option 决定（默认值）：

| 行为                          | 默认 | 关闭方式                                   |
|-----------------------------|----|----------------------------------------|
| `--version` 版本 flag 与启动版本打印 | 启用 | `WithNoVersion()`                      |
| `--config/-c` 与配置文件加载       | 启用 | `WithNoConfig()`                       |
| 启动信息/配置打印                   | 启用 | `WithSilence()`                        |
| 位置参数校验                      | 无  | `WithValidArgs`/`WithDefaultValidArgs` |

内置全局 flag 来自 marmotedu/component-base 的 `globalflag`/`verflag`（如 `--version`）。

## 最小可运行示例

仓库内的真实使用者是 [pkg/genyaml/app.go](../../../../pkg/genyaml/app.go)：

```go
package main

import (
	"github.com/ClessLi/component-base/pkg/app"
	"github.com/ClessLi/component-base/pkg/genyaml/options"
)

func main() {
	opts := options.NewGenYamlOptions()
	application := app.NewApp("generate-yaml",
		"myapp",
		app.WithOptions(opts),
		app.WithDescription("An application used to generate yaml configurations."),
		app.WithDefaultValidArgs(),
		app.WithNoConfig(), // 工具类应用不需要配置文件
		app.WithRunFunc(func(basename string) error {
			// 业务启动逻辑，opts 已被命令行参数回填
			return nil
		}),
	)
	application.Run()
}
```

## 进阶示例

### 带配置文件的服务端选项

业务选项需实现 `CliOptions`，字段带 `mapstructure` 标签才能被 viper 回填：

```go
type ServerOptions struct {
Addr string `json:"addr" mapstructure:"addr"`
}

func (o *ServerOptions) Flags() (fss cliflag.NamedFlagSets) {
fs := fss.FlagSet("server")
fs.StringVar(&o.Addr, "addr", o.Addr, "Listen address.")
return fss
}

func (o *ServerOptions) Validate() []error { return nil }

// main:
opts := &ServerOptions{Addr: ":8080"}
app.NewApp("my server", "myserver",
app.WithOptions(opts),
app.WithRunFunc(func (basename string) error {
// 此处 opts.Addr 已合并 命令行 > 配置文件 > 环境变量 > 默认值
return nil
}),
).Run()
```

配合 `myserver.yaml`（内容 `addr: :9090`）或环境变量 `MYSERVER_ADDR=:9090` 运行。

### 子命令

`Command`/`NewCommand` 提供子命令描述结构，但 `NewApp` 构造时即完成 cobra 命令树构建，
且当前没有注入子命令的 Option，因此 `NewApp` 之后再 `AddCommand` 不会生效（见注意事项 2）。
需要子命令时可直接操作内部 cobra 命令：

```go
application := app.NewApp("tool", "tool", app.WithNoConfig())
application.Command().AddCommand(&cobra.Command{
Use:   "gen",
Short: "Generate something",
RunE: func (cmd *cobra.Command, args []string) error {
return nil
},
})
application.Run()
```

## 与其他库的协作关系

- **[log/v1](log.md)**：启动信息经 `log.Infof` 输出；业务通常在 `RunFunc` 内调用 `log.Init`。
- **[errors](errors.md)**：`Validate` 返回的多错误经 `errors.NewAggregate` 聚合上报。
- **[genyaml](genyaml.md)**：`NewGenYamlApp` 直接复用本包骨架，并取 `App.Command()` 生成命令文档。
- **[generic_server](generic_server_server.md)**：服务端示例的 options 体系遵循同样的
  Flags/Validate/Complete 模式。

## 注意事项、限制与常见问题

1. **配置文件缺失即退出**：未加 `WithNoConfig()` 时，`cobra.OnInitialize` 中
   `viper.ReadInConfig` 失败会打印错误并 `os.Exit(1)`——**默认必须存在可发现的配置文件**，
   纯命令行工具请加 `WithNoConfig()`。
2. **AddCommand 的时机**：`NewApp` 内部即 `buildCommand`，创建后再 `AddCommand` 不会自动
   重建 cobra 命令树；子命令应在 `NewApp` 之前准备好（当前无 WithCommands 选项，
   需要子命令时参考 genyaml 的用法或自行操作 `Command()`，待确认最佳实践）。
3. **viper 回填依赖 mapstructure 标签**：选项字段缺少标签时配置文件/环境变量值不会生效。
4. **`--help` 短选项为 `-H`**（help.go），子命令同样注册 `-H`。
5. **Windows basename**：`FormatBasename` 会转小写并去 `.exe`，影响配置文件搜索名与环境变量前缀。
6. **依赖外部 marmotedu/component-base**：cliflag/globalflag/term/version/homedir 来自
   `github.com/marmotedu/component-base`（go.mod 依赖），并非本仓库 pkg 内的实现。