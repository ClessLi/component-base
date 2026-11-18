# prevent_cycles：有向图循环依赖检测

## 库定位与适用场景

`pkg/prevent_cycles` 基于 [`github.com/dominikbraun/graph`](https://github.com/dominikbraun/graph)
封装"增量加边 + 环检测"能力：每次声明一条 `src → dst` 依赖边，若该边会引入环则拒绝并返回
包含拓扑排序信息的错误。

适用场景：

- 任务/模块/资源依赖关系建模，需要在建立依赖时即时阻止循环依赖；
- 配置校验（如路由转发链、插件加载顺序）中的环检测。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import "github.com/ClessLi/component-base/pkg/prevent_cycles"
```

## 核心概念

- **Preventer 接口**：只暴露 `CheckLoop(src, dst any) error`，语义为"尝试登记 src→dst 依赖，
  成环则报错"。
- **内部图**：`graph.New(hash, graph.PreventCycles(), graph.Directed())` 构建的有向防环图。
  `CheckLoop` 先幂等地添加两个顶点（容忍 `ErrVertexAlreadyExists`），再添加边；
  加边失败（如 `ErrEdgeWouldCreateCycle`）时执行拓扑排序，把当前依赖链与失败边拼入错误消息。
- **重复边幂等**：`ErrEdgeAlreadyExists` 被容忍，重复登记同一条边不报错。
- **泛型构造**：`New[K comparable, T any](hash graph.Hash[K, T])` 支持自定义顶点类型与哈希；
  `NewStringPreventer()` 是字符串顶点的便捷入口。

## 关键类型与函数清单

所在文件：`pkg/prevent_cycles/prevent_cycles.go`。

| 签名                                                               | 说明                           |
|------------------------------------------------------------------|------------------------------|
| `type Preventer interface { CheckLoop(any, any) error }`         | 环检测器接口                       |
| `func NewStringPreventer() Preventer`                            | 字符串顶点检测器（`graph.StringHash`） |
| `func New[K comparable, T any](hash graph.Hash[K, T]) Preventer` | 泛型构造器，hash 为 graph 库的顶点哈希函数  |

`CheckLoop` 行为细节：

- 顶点参数按 `T` 断言、边端点按 `K` 断言；`NewStringPreventer` 下两者均为 `string`，
  传入其他类型会 panic（类型断言失败）。
- 成环错误消息形如：`edge ... [Topological sort: aaa -> bbb -> ccc; Add edge failed: ccc -x-> aaa]`。

## 配置项说明

本包无 Options 配置项。图的特性（有向、防环）在构造时固定。

## 最小可运行示例

改写自 [prevent_cycles_test.go](../../../../pkg/prevent_cycles/prevent_cycles_test.go)：

```go
package main

import (
	"fmt"

	"github.com/ClessLi/component-base/pkg/prevent_cycles"
)

func main() {
	p := prevent_cycles.NewStringPreventer()

	fmt.Println(p.CheckLoop("aaa", "bbb")) // <nil>
	fmt.Println(p.CheckLoop("bbb", "ccc")) // <nil>
	fmt.Println(p.CheckLoop("aaa", "ccc")) // <nil>（捷径边，不成环）

	err := p.CheckLoop("ccc", "aaa") // 成环，拒绝
	fmt.Println(err)
	// 形如：<graph 加边错误> [Topological sort: aaa -> bbb -> ccc; Add edge failed: ccc -x-> aaa]
}
```

## 进阶示例

### 自定义顶点类型

```go
type Module struct {
Name string
}

func moduleHash(m Module) string { return m.Name }

p := prevent_cycles.New[string, Module](moduleHash)
// CheckLoop 的 src/dst 传 Module 值（顶点按 T=Module 断言，边端点按 K=string 断言）
```

注意：`New` 的 K/T 组合需与 `graph.Hash[K, T]` 的签名一致，且 `CheckLoop` 传入的值必须能
同时断言为 T（顶点）与 K（边端点）；K 与 T 不同类型时（如上例 `string`/`Module`），
`CheckLoop(src, dst any)` 的单一参数无法同时满足两种断言，**建议始终使用 K=T 的实例化**
（如 `NewStringPreventer`），其他组合待确认。

## 与其他库的协作关系

- **[errors](errors.md)**：`errors.Is` 用于容忍 graph 库的哨兵错误；成环错误经 `errors.Errorf`
  构造。
- 当前仓库内无其他库直接依赖本包，属独立工具，可被业务项目直接引用。

## 注意事项、限制与常见问题

1. **只增不删**：Preventer 不提供删边/删顶点接口，依赖关系登记后不可撤销；需要撤销语义请
   直接使用底层 graph 库。
2. **非并发安全**：内部图无锁保护，并发调用 `CheckLoop` 需外部同步。
3. **错误消息中的拓扑排序**：仅在加边失败时计算，用于展示当前依赖链；图本身仍保持加边前状态
   （失败边未写入）。
4. **类型断言 panic**：`CheckLoop` 参数类型与实例化的 K/T 不符时直接 panic，调用方需保证类型一致。