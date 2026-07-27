# 错误码

CodeExample错误码列表，由
`codegen -type=int -fullname=CodeExample -namespace=codeexample -doc-output=../../docs/guide/zh-CN/examples/error_code_generated.md -wrapper`
命令生成，不要对此文件做任何更改。

## 功能说明

如果返回结果中存在 `code` 字段，则表示调用 API 接口失败。例如：

```json
{
  "code": 100101,
  "message": "Database error"
}
```

上述返回中 `code` 表示错误码，`message` 表示该错误的具体信息。每个错误同时也对应一个 HTTP 状态码，比如上述错误码对应了 HTTP
状态码 500(Internal Server Error)。

## 错误码列表

CodeExample 系统支持的错误码列表如下：

| Identifier                 | Code   | HTTP Code | Description                                                                         | Reference                                                                                                                                                    |
|----------------------------|--------|-----------|-------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ErrSuccess                 | 100001 | 200       | OK                                                                                  | https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/errors/success.md                                                                       |
| ErrUnknown                 | 100002 | 500       | Contact system administrator with error details for investigation                   | Internal server error occurred, check logs for details (see details: https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/errors/unknown.md) |
| ErrBind                    | 100003 | 400       | Check request body and ensure it matches the expected format                        | https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/errors/bind.md                                                                          |
| ErrValidation              | 100004 | 400       | Check input parameters and ensure they meet validation requirements                 | Input validation failed, check parameter constraints                                                                                                         |
| ErrTokenInvalid            | 100005 | 401       | Provide a valid authentication token                                                |                                                                                                                                                              |
| ErrPageNotFound            | 100006 | 404       | Check the URL path and try again                                                    |                                                                                                                                                              |
| ErrRequestTimeout          | 100007 | 408       | Retry request or check network connection                                           | Request timeout, check network connectivity and retry                                                                                                        |
| ErrInvalidParameter        | 100008 | 400       | Check input parameters and ensure they are valid                                    |                                                                                                                                                              |
| ErrDatabase                | 100101 | 500       | Contact system administrator to resolve database connectivity issue                 | https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/errors/database.md                                                                      |
| ErrEncrypt                 | 100301 | 401       | Provide a valid password that can be encrypted                                      |                                                                                                                                                              |
| ErrSignatureInvalid        | 100302 | 401       | Provide a request with valid signature                                              |                                                                                                                                                              |
| ErrExpired                 | 100303 | 401       | Refresh authentication token and try again                                          |                                                                                                                                                              |
| ErrInvalidAuthHeader       | 100304 | 401       | Provide request with valid authorization header                                     |                                                                                                                                                              |
| ErrMissingHeader           | 100305 | 401       | Provide request with valid Authorization header                                     |                                                                                                                                                              |
| ErrUserOrPasswordIncorrect | 100306 | 401       | Provide valid username and password credentials                                     |                                                                                                                                                              |
| ErrPermissionDenied        | 100307 | 403       | Contact system administrator for required permissions                               |                                                                                                                                                              |
| ErrAuthnClientInitFailed   | 100308 | 500       | Contact system administrator to initialize authentication client                    |                                                                                                                                                              |
| ErrAuthClientNotInit       | 100309 | 500       | Contact system administrator to initialize authentication and authorization clients |                                                                                                                                                              |
| ErrConnToAuthServerFailed  | 100310 | 500       | Contact system administrator to check authentication server connectivity            |                                                                                                                                                              |
| ErrEncodingFailed          | 100401 | 500       | Check data format and ensure it can be properly encoded                             | https://github.com/ClessLi/component-base/blob/main/docs/guide/zh-CN/errors/encoding.md                                                                      |
| ErrDecodingFailed          | 100402 | 500       | Check data format and ensure it can be properly decoded                             |                                                                                                                                                              |
| ErrInvalidJSON             | 100403 | 500       | Provide data in valid JSON format                                                   |                                                                                                                                                              |
| ErrEncodingJSON            | 100404 | 500       | Check JSON data and ensure it can be properly encoded                               |                                                                                                                                                              |
| ErrDecodingJSON            | 100405 | 500       | Check JSON data and ensure it can be properly decoded                               |                                                                                                                                                              |
| ErrInvalidYaml             | 100406 | 500       | Provide data in valid YAML format                                                   |                                                                                                                                                              |
| ErrEncodingYaml            | 100407 | 500       | Check YAML data and ensure it can be properly encoded                               |                                                                                                                                                              |
| ErrDecodingYaml            | 100408 | 500       | Check YAML data and ensure it can be properly decoded                               |                                                                                                                                                              |

