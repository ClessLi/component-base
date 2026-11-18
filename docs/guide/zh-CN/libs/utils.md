# utils：反射与字符串转换小工具

## 库定位与适用场景

`pkg/utils` 收录跨库复用的轻量工具函数，当前包含两类：

- **反射判空/判型**（`reflect.go`）：`IsNil`、`IsPointer`、`IsPointerType`、`IsNullOrZeroValue`；
- **字符串列表转换**（`strlistconv.go`）：`ParseInt64List`。

适用场景：

- 泛型代码中需要对 `interface{}`/泛型参数做运行时 nil 判断（如 client-sdk 的 endpoint/encoder）；
- 把 URL 查询参数、命令行参数等字符串切片批量转换为 `[]int64`。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import "github.com/ClessLi/component-base/pkg/utils"
```

## 关键类型与函数清单

### reflect.go

| 签名                                               | 说明                                                                            |
|--------------------------------------------------|-------------------------------------------------------------------------------|
| `func IsNil(i interface{}) bool`                 | nil 接口返回 true；否则仅当值为**指针类型且为 nil** 时返回 true                                   |
| `func IsPointer(i interface{}) bool`             | 判断动态类型是否为指针（`reflect.Pointer`）                                                |
| `func IsPointerType[A any]() bool`               | 泛型版判型：判断类型参数 A 本身是否为指针类型                                                      |
| `func IsNullOrZeroValue(value interface{}) bool` | nil、空串、数值 0、false、空 `[]interface{}`/`map[string]interface{}` 及其他类型的零值均返回 true |

### strlistconv.go

| 签名                                                    | 说明                                                                       |
|-------------------------------------------------------|--------------------------------------------------------------------------|
| `func ParseInt64List(strs []string) ([]int64, error)` | 逐项 `strconv.ParseInt(str, 10, 64)`；失败时返回 nil 与带列表下标的包装错误（`errors.Wrapf`） |

## 配置项说明

本包无 Options 配置项，全部为纯函数。

## 最小可运行示例

```go
package main

import (
	"fmt"

	"github.com/ClessLi/component-base/pkg/utils"
)

type User struct{ Name string }

func main() {
	var u *User
	fmt.Println(utils.IsNil(u))        // true（typed nil 指针）
	fmt.Println(utils.IsNil(nil))      // true
	fmt.Println(utils.IsNil(User{}))   // false（非指针）

	fmt.Println(utils.IsPointer(&User{}))      // true
	fmt.Println(utils.IsPointerType[*User]())  // true
	fmt.Println(utils.IsPointerType[User]())   // false

	fmt.Println(utils.IsNullOrZeroValue("")) // true（空串视为零值）
	fmt.Println(utils.IsNullOrZeroValue(0))  // true

	ids, err := utils.ParseInt64List([]string{"1", "2", "3"})
	fmt.Println(ids, err) // [1 2 3] <nil>
}
```

## 进阶示例

client-sdk 中的真实用法（[pkg/client-sdk/http/v1/encoder.go](../../../../pkg/client-sdk/http/v1/encoder.go)）：

```go
// 泛型请求体编码：类型层面排除 NilBody 占位类型，值层面排除 nil 指针
shouldMarshalBody := !isNilBody[REQ]() && !utils.IsNil(req.Body)
```

endpoint 层对响应做同样的判空保护（[pkg/client-sdk/grpc/v1/endpoint.go](../../../../pkg/client-sdk/grpc/v1/endpoint.go)）：

```go
if utils.IsNil(resp) {
// 返回零值/占位响应，避免 nil 解引用
}
```

## 与其他库的协作关系

- **[client-sdk](client_sdk_http.md)**：HTTP/gRPC SDK 的 encoder 与 endpoint 依赖 `IsNil` 做
  运行时判空，配合 `NilBody` 占位类型处理空请求/响应体。
- **[errors](errors.md)**：`ParseInt64List` 的错误经 `errors.Wrapf` 包装，携带失败项下标。

## 注意事项、限制与常见问题

1. **`IsNil` 只覆盖指针**：`Kind() == reflect.Ptr && v.IsNil()`。typed nil 的
   map/slice/chan/func/interface **不会**被判为 nil；需要这些类型的判空请自行处理。
2. **`IsPointer(nil)` 会 panic**：`reflect.TypeOf(nil)` 返回 nil，调用 `.Kind()` 触发空指针
   解引用。传入前确保值非 nil 接口。
3. **`IsNullOrZeroValue` 的 default 分支用 `==` 比较零值**：对不可比较类型
   （如 `[]string`、`map[string]int`，注意不是 `[]interface{}`/`map[string]interface{}`）
   会 panic；此类类型请避免传入。
4. **`ParseInt64List` 失败即整体失败**：返回 nil 切片而非部分结果；错误消息含失败项的
   列表下标，便于定位。