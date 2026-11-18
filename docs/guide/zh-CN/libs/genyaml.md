# genyaml：cobra 命令 YAML 文档生成器

## 库定位与适用场景

`pkg/genyaml` 把 cobra 命令树（命令名、描述、flag 清单、示例、子命令关联）导出为 YAML 文档，
用于为 CLI 应用生成机器可读的命令参考（风格类似 kubectl 命令文档的 YAML 源）。

适用场景：

- 已有基于 cobra（或 [app](app.md) 骨架）的 CLI，需要批量产出命令说明 YAML；
- 文档流水线需要以 YAML 为中间格式再生成 Markdown/站点页面。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import (
"github.com/ClessLi/component-base/pkg/genyaml"
"github.com/ClessLi/component-base/pkg/genyaml/options"
"github.com/ClessLi/component-base/pkg/genyaml/server"
)
```

## 核心概念

- **入口**：`NewGenYamlApp(yamlname string, command *cobra.Command) *app.App` 创建一个
  [app](app.md) 应用（`WithNoConfig`，无配置文件依赖），运行后对传入的 cobra 命令及其
  **直接子命令**逐个生成 YAML。
- **配置链路**：`GenYamlOptions` →（`ApplyTo`）→ `server.Config` →（`Complete().NewServer()`）
  → `GenerateYamlServer` →（`GenYaml(command)`）→ 落盘。
- **输出文件命名**：无 ParentName 时 `<DocsDir>/<命令名>.yaml`；有 ParentName 时
  `<DocsDir>/<ParentName>_<命令名>.yaml`。
- **YAML 结构**（`server/config.go` 的 `cmdDoc`/`cmdOption`）：
  `name`、`synopsis`（Short）、`description`（Long）、`options`（本命令 flag）、
  `inherited_options`（继承 flag）、`example`、`see_also`（ParentName + 子命令名）。
  超过 60 字符且无换行的文本会追加换行（`forceMultiLine`，YAML 多行块）。

## 关键类型与函数清单

### 顶层（app.go、run.go、server.go）

| 签名                                                                                   | 说明                               |
|--------------------------------------------------------------------------------------|----------------------------------|
| `func NewGenYamlApp(yamlname string, command *cobra.Command) *app.App`               | 创建生成器应用；yamlname 作为 app basename |
| `func CreateConfigFromOptions(opts *options.GenYamlOptions) (*server.Config, error)` | Options → Config                 |
| `func Run(cfg *server.Config, command *cobra.Command) error`                         | 构建 server 并执行生成                  |

### options/gen_yaml_options.go

| 签名                                                                       | 说明                                                         |
|--------------------------------------------------------------------------|------------------------------------------------------------|
| `type GenYamlOptions struct { ParentName string; OutputDirPath string }` | 生成选项（json/mapstructure 标签 `parent-name`/`output-dir-path`） |
| `func NewGenYamlOptions() *GenYamlOptions`                               | 零值构造器                                                      |
| `func (g *GenYamlOptions) Flags() cliflag.NamedFlagSets`                 | 注册到 "generate yaml" 分组                                     |
| `func (g *GenYamlOptions) AddFlags(fs *pflag.FlagSet)`                   | 注册 `--parent-name`、`--output-dir-path`                     |
| `func (g *GenYamlOptions) Validate() []error`                            | 校验输出目录非空且为已存在目录                                            |
| `func (g *GenYamlOptions) ApplyTo(c *server.Config) error`               | 输出目录取绝对路径并 Clean，写入 YamlInfo                               |

### server/config.go、server/server.go

| 签名                                                                       | 说明                 |
|--------------------------------------------------------------------------|--------------------|
| `type YamlInfo struct { ParentName string; DocsDir string }`             | 输出信息               |
| `type Config struct { YamlInfo *YamlInfo }` / `func NewConfig() *Config` | 配置                 |
| `func (c *Config) Complete() CompletedConfig`                            | 补全（当前为直通包装）        |
| `func (c CompletedConfig) NewServer() (*GenerateYamlServer, error)`      | 构建生成器              |
| `type GenerateYamlServer struct { YamlInfo *YamlInfo }`                  | 生成器                |
| `func (g *GenerateYamlServer) GenYaml(command *cobra.Command) error`     | 生成命令自身与直接子命令的 YAML |

## 配置项说明

| flag                | 类型     | 默认值      | 说明                            |
|---------------------|--------|----------|-------------------------------|
| `--parent-name`     | string | `""`     | 生成文件的父级前缀，同时写入 `see_also`     |
| `--output-dir-path` | string | `""`（必填） | 输出目录，**必须已存在**，否则 Validate 失败 |

## 最小可运行示例

```go
package main

import (
	"github.com/spf13/cobra"

	"github.com/ClessLi/component-base/pkg/genyaml"
)

func main() {
	root := &cobra.Command{
		Use:   "myapp",
		Short: "My application",
		Long:  "A longer description of my application.",
	}
	root.Flags().String("addr", ":8080", "Listen address.")

	// 生成 myapp.yaml 到 ./docs 目录（目录需先存在）
	application := genyaml.NewGenYamlApp("myapp", root)
	application.Command().Flags().Set("output-dir-path", "./docs")
	application.Run()
}
```

更常见的方式是直接以命令行运行（app 骨架自动绑定 flag）：

```bash
myapp-gen --output-dir-path=./docs --parent-name=myapp
```

## 进阶示例

### 编程式调用（不经 app 骨架）

```go
opts := options.NewGenYamlOptions()
opts.OutputDirPath = "./docs"
opts.ParentName = "myapp"

cfg, err := genyaml.CreateConfigFromOptions(opts)
if err != nil {
// ...
}
if err := genyaml.Run(cfg, rootCmd); err != nil {
// ...
}
```

### 输出示例

对含 `--addr` flag 与子命令 `serve` 的 `myapp`，生成 `myapp_myapp.yaml`（ParentName=myapp）：

```yaml
name: myapp
synopsis: My application
description: A longer description of my application.
options:
    - name: addr
      default_value: ':8080'
      usage: Listen address.
see_also:
    - myapp
    - serve
```

## 与其他库的协作关系

- **[app](app.md)**：`NewGenYamlApp` 复用 app 骨架（`WithOptions`/`WithNoConfig`/
  `WithDefaultValidArgs`/`WithRunFunc`），flag 解析与校验由 app 流程驱动。
- **[errors](errors.md)**：Validate 错误经 `errors.Errorf` 构造。
- cobra/pflag/yaml.v3：命令树遍历、flag 读取与 YAML 序列化。

## 注意事项、限制与常见问题

1. **输出目录必须预先存在**：`Validate` 对不存在的目录报错，不会自动创建。
2. **只遍历一层子命令**：`GenYaml` 处理命令自身与 `command.Commands()` 直接子命令，
   不递归孙级命令；深层命令树需自行逐层调用。
3. **`server/config.go` 带构建标签 `!viper_yaml2`**：启用 `viper_yaml2` tag 编译时该文件被排除，
   仓库内未提供替代实现（待确认该 tag 的用途）。
4. **flag 快照时机**：生成读取的是调用时的 flag 定义（DefValue/Usage），应在所有 flag 注册
   完成后再生成。
5. **Validate 的边界缺陷**：`os.Stat` 出错且被判为"已存在"的极端分支可能对 nil FileInfo
   调用 `IsDir()`；常规路径（目录存在/不存在）行为正确。