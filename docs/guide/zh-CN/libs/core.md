# core：HTTP 响应写出核心工具

## 库定位与适用场景

`pkg/core` 提供 API 服务端的核心响应写出能力：把业务返回的 `(data, error)` 统一转换为 HTTP JSON
响应。错误经 [errors](errors.md) 的 `ParseCoder` 解析为错误码元信息，成功则直接序列化数据。

适用场景：

- 基于 gin 的 HTTP API 控制器，需要统一的错误响应格式（namespace/code/message/reference）；
- 希望错误日志（含堆栈详情）与对外响应文案自动分离。

## 安装与导入路径

```bash
go get github.com/ClessLi/component-base
```

```go
import "github.com/ClessLi/component-base/pkg/core"
```

## 核心概念

- **ErrResponse**：对外错误响应体，字段与 `errors.Coder` 一一对应；`Reference` 为空时省略
  （`omitempty`）。
- **WriteResponse**：唯一的响应出口。`err != nil` 时先以 `log.Errorf("%#+v", err)` 记录 JSON
  格式错误详情（含堆栈），再按 Coder 的 HTTP 状态码写出 ErrResponse；`err == nil` 时以
  HTTP 200 写出 data。

## 关键类型与函数清单

所在文件：`pkg/core/core.go`。

| 签名                                                                                         | 说明                                                               |
|--------------------------------------------------------------------------------------------|------------------------------------------------------------------|
| `type ErrResponse struct { Namespace string; Code int; Message string; Reference string }` | 错误响应体；JSON 键为 `namespace`、`code`、`message`、`reference,omitempty` |
| `func WriteResponse(c *gin.Context, err error, data interface{})`                          | 按 err 是否为 nil 写出错误响应或 200 数据响应                                   |

ErrResponse 字段定义：

```go
type ErrResponse struct {
Namespace string `json:"namespace"`
Code      int    `json:"code"`
Message   string `json:"message"`
Reference string `json:"reference,omitempty"`
}
```

## 配置项说明

本包无 Options 配置项。错误码到 HTTP 状态码/文案的映射完全由 errors 包的注册表决定。

## 最小可运行示例

改写自 [examples/apiserver/controller.go](../../../../examples/apiserver/controller.go)：

```go
package apiserver

import (
	"github.com/gin-gonic/gin"

	"github.com/ClessLi/component-base/pkg/core"
)

type UserController struct {
	svc UserService
}

// Get handles GET /v1/users/:id.
func (ctl *UserController) Get(c *gin.Context) {
	u, err := ctl.svc.Get(c.Request.Context(), c.Param("id"))
	core.WriteResponse(c, err, u)
}

// List handles GET /v1/users.
func (ctl *UserController) List(c *gin.Context) {
	users, err := ctl.svc.List(c.Request.Context())
	core.WriteResponse(c, err, users)
}
```

错误时的响应示例（错误码已注册的前提下）：

```json
{
  "namespace": "codeexample",
  "code": 100006,
  "message": "Check the URL path and try again",
  "reference": "https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/examples/error_code_generated.md#L22"
}
```

## 进阶示例

完整的 gin 服务装配（路由、中间件、服务层、存储层）见
[examples/apiserver](../../../../examples/apiserver/)，其中控制器全部经 `core.WriteResponse`
出口，配合 [errors](errors.md) 错误码与 [log/v1](log.md) 日志。

## 与其他库的协作关系

- **[errors](errors.md)**：`ParseCoder` 提供错误码解析；未注册码/普通错误回退 unknownCoder
  （HTTP 500、code 1、命名空间 `base`）。
- **[log/v1](log.md)**：错误详情经 `log.Errorf("%#+v", err)` 输出为 JSON 数组日志。
- **[generic_server framework/http](generic_server_framework_http.md)**：HTTP 框架示例中的
  控制器响应写出即使用本包。

## 注意事项、限制与常见问题

1. **仅支持 gin**：`WriteResponse` 绑定 `*gin.Context`，不适用于 net/http 原生 handler 或 gRPC。
2. **未注册错误码的响应**：普通错误（`errors.New`/`fmt.Errorf`）会被解析为 unknownCoder，
   对外返回 HTTP 500 与通用文案，内部消息只进日志，不会泄露给调用方。
3. **成功响应固定 200**：需要 201/204 等状态码时须自行写出，本包不提供定制入口。
4. **关于 Copy 等工具能力**：当前 `core.go` 仅实现 `ErrResponse` 与 `WriteResponse`，
   不存在 Copy 等其他工具函数（待确认是否有后续扩展计划）。
5. **旧注释待修正**：`pkg/core/doc.go` 的 import 注释仍写作 `github.com/marmotedu/pkg/core`，
   实际导入路径以 go.mod 为准（本文档「安装与导入路径」一节）。