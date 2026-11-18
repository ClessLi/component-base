# sort_map：并发安全的有序映射与稀疏列表

## 库定位与适用场景

`pkg/sort_map` 提供三个泛型容器：

- **SortMap[K, V]**：键按自然序（`cmp.Ordered`）排序的并发安全映射，支持 JSON/YAML 序列化与
  Go 1.23 迭代器（`iter.Seq`/`iter.Seq2`）；
- **MapIndexes[K]**：排序键索引（SortMap 的内部构件，也可独立使用）；
- **MapList[V]**：以非负 int 为索引的稀疏列表，支持"插入第一个空位"与"追加到末尾"。

适用场景：

- 需要确定性遍历顺序（按键排序）的配置表、注册表，且要求并发安全；
- 配置文件（JSON/YAML）中的映射结构需要稳定输出顺序；
- 需要可删除、可复用空位的索引式列表（如连接槽位、任务槽位）。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import "github.com/ClessLi/component-base/pkg/sort_map"
```

## 核心概念

- **数据与索引分离**：`sortMap` 内部为 `map[K]V` + `MapIndexes[K]`（有序键切片，二分查找定位），
  所有公开方法经 `sync.RWMutex` 保护。
- **迭代器持锁遍历**：`Keys`/`Values`/`Range` 等返回的 `iter.Seq` 在迭代期间持读锁，
  迭代体内不要再调用同一容器的写方法（会死锁）。
- **MapList 复用 SortMap**：`mapList[V]` 内嵌 `*sortMap[int, V]`，索引即键；
  `InsertToEmpty` 二分查找第一个空洞，`Append` 取最大索引 +1。
- **Mock**：`mock_sort_map.go` 由 mockgen 生成（`go:generate` 指令见 `sort_map.go` 头部），
  提供 `MockSortMap`/`MockMapList`/`MockMapIndexes` 供单元测试注入。

## 关键类型与函数清单

### SortMap（sort_map.go）

| 签名                                                    | 说明              |
|-------------------------------------------------------|-----------------|
| `func Map[K cmp.Ordered, V any]() SortMap[K, V]`      | 构造空 SortMap     |
| `Set(key K, value V) (oldValue V, present bool)`      | 写入并返回旧值         |
| `Get(key K) (value V, present bool)`                  | 读取              |
| `Contains(key K) (present bool)`                      | 键存在性            |
| `RemoveByKey(key K)`                                  | 删除键             |
| `Clear()`                                             | 清空              |
| `Len() int`                                           | 元素数             |
| `Keys() iter.Seq[K]`                                  | 按键升序迭代键         |
| `Values() iter.Seq[V]`                                | 按键升序迭代值         |
| `Range() iter.Seq2[K, V]`                             | 按键升序迭代键值对       |
| `Indexes() iter.Seq[int]`                             | 迭代序号 0..Len-1   |
| `RangeWithIndex() iter.Seq2[int, V]`                  | 迭代 (序号, 值)      |
| `MarshalJSON/UnmarshalJSON/MarshalYAML/UnmarshalYAML` | 序列化；反序列化后重建排序索引 |

### MapIndexes（map_indexes.go）

| 签名                                                   | 说明             |
|------------------------------------------------------|----------------|
| `func NewMapIndexes[K cmp.Ordered]() MapIndexes[K]`  | 构造空索引          |
| `Insert(key K)`                                      | 二分定位插入，重复键幂等   |
| `GetKey(idx int) K`                                  | 越界返回零值         |
| `IndexOf(key K) int`                                 | 不存在返回 -1       |
| `Contains(key K) (present bool)`                     | 存在性            |
| `Remove(key K)`                                      | 删除             |
| `Len() int`                                          | 长度             |
| `Keys() iter.Seq[int]` / `Range() iter.Seq2[int, K]` | 迭代序号 / (序号, 键) |

注意：MapIndexes 本身**不加锁**，并发安全由外层 SortMap 保证。

### MapList（map_list.go）

| 签名                                                      | 说明                 |
|---------------------------------------------------------|--------------------|
| `func List[V any]() MapList[V]`                         | 构造空列表              |
| `Set(index int, value V) (oldValue V, present bool)`    | 负索引静默忽略            |
| `InsertToEmpty(value V) int`                            | 插入第一个空洞索引并返回该索引    |
| `Append(value V) int`                                   | 追加到最大索引 +1 并返回索引   |
| `Get(index int) (value V, present bool)`                | 负索引返回零值/false      |
| `Contains(index int) (present bool)`                    | 索引存在性（继承自 SortMap） |
| `Remove(index int)`                                     | 删除后留下空洞            |
| `Clear()` / `Len() int`                                 | 清空 / 元素数           |
| `Range() iter.Seq2[int, V]` / `Indexes() iter.Seq[int]` | 按索引升序迭代            |
| `MarshalJSON/UnmarshalJSON/MarshalYAML/UnmarshalYAML`   | 序列化；反序列化拒绝负索引      |

## 配置项说明

本包无 Options 配置项，容器行为由泛型参数与构造器决定。

## 最小可运行示例

```go
package main

import (
	"fmt"

	"github.com/ClessLi/component-base/pkg/sort_map"
)

func main() {
	m := sort_map.Map[string, int]()
	m.Set("banana", 2)
	m.Set("apple", 1)
	m.Set("cherry", 3)

	for k, v := range m.Range() { // 按键升序：apple, banana, cherry
		fmt.Printf("%s=%d\n", k, v)
	}

	l := sort_map.List[string]()
	i0 := l.Append("a") // 0
	i1 := l.Append("b") // 1
	l.Remove(i0)        // 索引 0 成为空洞
	i2 := l.InsertToEmpty("c") // 复用空洞，返回 0
	fmt.Println(i0, i1, i2)    // 0 1 0
}
```

## 进阶示例

### JSON/YAML 配置结构

```go
type Config struct {
Backends sort_map.SortMap[string, string] `json:"backends" yaml:"backends"`
}

// 注意：接口字段需先实例化再反序列化
cfg := Config{Backends: sort_map.Map[string, string]()}
if err := json.Unmarshal(data, &cfg); err != nil {
// ...
}
// MarshalJSON 输出标准 JSON 对象（encoding/json 对 map 键本身按字典序输出）
```

### 单元测试中注入 Mock

```go
// go.uber.org/mock/gomock
ctrl := gomock.NewController(t)
m := sort_map.NewMockSortMap[string, int](ctrl)
m.EXPECT().Get("k").Return(1, true)
```

Mock 由 `go generate ./pkg/sort_map` 重新生成（依赖 `go.uber.org/mock` 的 mockgen）。

## 与其他库的协作关系

- **[errors](errors.md)**：YAML 反序列化空节点、MapList 负索引等错误经 `errors.New`/`errors.Errorf`
  返回。
- 当前仓库内无其他库直接依赖本包，属独立通用容器，可被业务项目直接引用。

## 注意事项、限制与常见问题

1. **键类型受限**：`K` 必须满足 `cmp.Ordered`（整型/浮点/字符串），不支持结构体键。
2. **迭代期间持读锁**：在 `Range` 等迭代体内调用同容器 `Set`/`RemoveByKey` 会死锁；
   先收集再修改。
3. **MarshalJSON 输出顺序**：`MarshalJSON` 直接序列化内部 map，`encoding/json` 按键字典序输出，
   与 SortMap 的排序一致（字符串键）；数值键的 JSON 对象键同样按字典序而非数值序。
4. **MapList 负索引语义**：`Set`/`Get`/`Remove` 对负索引静默忽略；JSON/YAML 反序列化遇到负索引
   返回错误。
5. **接口字段反序列化**：SortMap/MapList 是接口类型，作为结构体字段时必须先用 `Map()`/`List()`
   实例化，否则 `Unmarshal` 无法构造实现。
6. **MapIndexes 非并发安全**：独立使用时需自行加锁。