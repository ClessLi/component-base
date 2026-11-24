# sort_map：并发安全的有序映射与稀疏列表

## 库定位与适用场景

`pkg/sort_map` 提供三个泛型容器：

- **SortMap[K, V]**：具体结构体，键按自然序（`cmp.Ordered`）排序的并发安全映射，支持 JSON/YAML 序列化与
  Go 1.23 迭代器（`iter.Seq`/`iter.Seq2`）；
- **MapIndexes[K]**：排序键索引（SortMap 的内部构件，也可独立使用）；
- **MapList[V]**：具体结构体，以非负 int 为索引的稀疏列表，支持"插入第一个空位"与"追加到末尾"。

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

- **数据与索引分离**：`SortMap` 内部为 `map[K]V` + `MapIndexes[K]`（有序键切片，二分查找定位），
  所有公开方法经 `sync.RWMutex` 保护；其**零值即空容器**，首次写入或反序列化时惰性初始化。
- **仅保留具体类型**：本包不提供行为接口，`SortMap`/`MapList` 即直接使用的具体 struct。
  业务结构体字段需要 JSON/YAML 反序列化时**必须直接使用 `SortMap`/`MapList`（或其指针）**，
  接口类型字段无法被 yaml.v3 分发（详见注意事项 5）。
- **构造器返回值类型**：`Map()`/`List()` 返回具体值类型 `SortMap[K, V]`/`MapList[V]`；
  `Set`/`Remove` 等指针接收者方法要求可寻址变量（`m := Map[string, int](); m.Set(...)`），
  不要对不可寻址的临时值调用写方法。
- **迭代器持锁遍历**：`Keys`/`Values`/`Range` 等返回的 `iter.Seq` 在迭代期间持读锁，
  迭代体内不要再调用同一容器的写方法（会死锁）。
- **MapList 复用 SortMap**：`MapList[V]` 内嵌值类型 `SortMap[int, V]`，同名方法（含值接收者的
  `MarshalJSON`/`MarshalYAML`）自动提升，索引即键；
  `InsertToEmpty` 二分查找第一个空洞，`Append` 取最大索引 +1，二者全程持写锁完成
  "计算空闲索引 + 写入"，并发下不会重复分配索引。

## 关键类型与函数清单

### SortMap（sort_map.go，具体类型）

| 签名                                                    | 说明                   |
|-------------------------------------------------------|----------------------|
| `func Map[K cmp.Ordered, V any]() SortMap[K, V]`      | 构造空 SortMap（返回具体值类型） |
| `Set(key K, value V) (oldValue V, present bool)`      | 写入并返回旧值              |
| `Get(key K) (value V, present bool)`                  | 读取                   |
| `Contains(key K) (present bool)`                      | 键存在性                 |
| `RemoveByKey(key K)`                                  | 删除键                  |
| `Clear()`                                             | 清空                   |
| `Len() int`                                           | 元素数                  |
| `Keys() iter.Seq[K]`                                  | 按键升序迭代键              |
| `Values() iter.Seq[V]`                                | 按键升序迭代值              |
| `Range() iter.Seq2[K, V]`                             | 按键升序迭代键值对            |
| `Indexes() iter.Seq[int]`                             | 迭代序号 0..Len-1        |
| `RangeWithIndex() iter.Seq2[int, V]`                  | 迭代 (序号, 值)           |
| `MarshalJSON/UnmarshalJSON/MarshalYAML/UnmarshalYAML` | 序列化；反序列化后重建排序索引      |

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

### MapList（map_list.go，具体类型）

| 签名                                                      | 说明                         |
|---------------------------------------------------------|----------------------------|
| `func List[V any]() MapList[V]`                         | 构造空列表（返回具体值类型）             |
| `Set(index int, value V) (oldValue V, present bool)`    | 负索引静默忽略                    |
| `InsertToEmpty(value V) int`                            | 插入第一个空洞索引并返回该索引            |
| `Append(value V) int`                                   | 追加到最大索引 +1 并返回索引           |
| `Get(index int) (value V, present bool)`                | 负索引返回零值/false              |
| `Contains(index int) (present bool)`                    | 索引存在性（继承自 SortMap）         |
| `Remove(index int)`                                     | 删除后留下空洞                    |
| `Clear()` / `Len() int`                                 | 清空 / 元素数                   |
| `Range() iter.Seq2[int, V]` / `Indexes() iter.Seq[int]` | 按索引升序迭代                    |
| `MarshalJSON/UnmarshalJSON/MarshalYAML/UnmarshalYAML`   | 序列化；反序列化拒绝负索引（错误信息含具体非法索引） |

YAML 反序列化节点支持：mapping（索引→值）、sequence（展开为 0..n-1）、null/空（重置为空）。

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

### JSON/YAML 配置结构（嵌套字段反序列化）

```go
type Config struct {
// 直接使用具体类型（值或指针），yaml.v3 / encoding/json 才能对字段
// 分发 UnmarshalYAML / UnmarshalJSON。
Backends sort_map.SortMap[string, string] `json:"backends" yaml:"backends"`
Slots    *sort_map.MapList[string]       `json:"slots" yaml:"slots"`
}

var cfg Config
if err := yaml.Unmarshal(data, &cfg); err != nil {
// ...
}
// 零值字段无需预初始化即可反序列化；读方法对零值安全（返回空结果，不 panic）。
// Marshal 输出标准对象（encoding/json 与 yaml.v3 均按键排序；注意 JSON 数值键按字典序）。
// 若字段可能接收 null，请声明为指针类型（null → nil 指针）；值类型结构体字段遇 null 时
// yaml.v3 不会调用 UnmarshalYAML，也不报错，字段保持原值不变。
```

注意：**接口类型的结构体字段不能被 yaml.v3 反序列化**（本包已不提供行为接口；若业务侧自行
定义接口包装这些容器，同样无法反序列化）——无论是否预初始化，yaml.v3 的分发机制都不会触达
其 `UnmarshalYAML`，且会因向接口字段回填通用 map 而直接 panic。仅 `encoding/json` 在字段
已用 `Map()`/`List()` 预初始化时可用。
本版调整（breaking change）：原 `SortMap`/`MapList` **接口已从本包删除**；
原 `BaseSortMap`/`BaseMapList` 具体类型更名为 `SortMap`/`MapList`，`Map()`/`List()` 改为返回具体值类型。
存量代码迁移方式：把字段类型直接改为具体 struct `SortMap`/`MapList`（或其指针）即可，其余调用代码不变。

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
   返回错误，错误信息包含具体的非法索引值。
5. **接口字段反序列化限制（重要）**：本包不再提供行为接口，容器字段应直接声明具体类型
   `SortMap[K, V]` / `MapList[V]` 或其指针（零值即可用）。任何接口类型的结构体字段
   （含业务自定义的包装接口）**yaml.v3 均无法反序列化**（其 `prepare()` 只解引用指针、不解引用接口，
   导致 `UnmarshalYAML` 永不触发，并向字段回填通用 map 引发 reflect panic；这是 yaml.v3 v3.0.1 的固有限制）。
   `encoding/json` 对"已预初始化的接口字段"可用，但不推荐依赖。
6. **null 与空节点语义**：
    - 空 flow mapping `{}` 会正常触发 `UnmarshalYAML`，容器被重置为**空容器**（内部 map 非 nil，后续 `Set` 安全）；直接以
      null 节点调用 `UnmarshalYAML` 同样重置为空容器。
    - 反序列化 `field:` / `field: null`（YAML null）时，yaml.v3 在 `prepare()` 中对 null 提前返回、**不调用**
      `UnmarshalYAML`：指针字段被置为 **nil**；值类型结构体字段则**保持原值不变且无错误**（yaml.v3 的 `d.null` 只清零
      Interface/Ptr/Map/Slice，不动 struct）。因此需要「null 清空/置空」语义时请使用指针字段。
7. **MapIndexes 非并发安全**：独立使用时需自行加锁。