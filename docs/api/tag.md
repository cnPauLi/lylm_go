# 导航菜单（Tag）域 API

管理站点顶部/底部导航菜单（后台「导航菜单管理」对应能力），共 4 个端点。

## 通用约定

- Base URL：`/api/v1`
- 鉴权：全部端点需在请求头携带 `Authorization: Bearer sk-...`；缺失或无效返回 `401`。
- 统一响应信封：

```json
{ "code": 200, "message": "ok", "data": {} }
```

失败时 HTTP 状态码与业务 `code` 一致，`data` 为 `null`。
- 通用错误码：`400` 参数校验失败、`401` 未鉴权/密钥无效、`404` 记录不存在、`429` 触发限流、`500` 服务器内部错误。

## Tag 对象字段

| 字段 | 类型 | 含义 |
|------|------|------|
| tag_id | int | 导航菜单 ID |
| tag_name | string | 导航名称 |
| tag_link | string | 导航链接 |
| tag_target | int | 打开方式：`1` 新标签页，`0` 当前页 |
| sort | int | 排序值（升序；未显式指定时默认为 `10`） |

---

## 1. 获取导航菜单列表

- **用途**：返回全部导航菜单，按 `sort` 升序。
- **方法路径**：`GET /tags`
- **鉴权**：需要。

### 入参

无（无查询参数、无请求体）。

### 出参

`data` 为对象：

| 字段 | 类型 | 含义 |
|------|------|------|
| list | Tag[] | 导航菜单列表 |
| total | int | 记录总数 |

### 错误码

`401`、`429`、`500`。

### 请求示例

```http
GET /api/v1/tags
Authorization: Bearer sk-xxxxxxxx
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "list": [
      {
        "tag_id": 1,
        "tag_name": "首页",
        "tag_link": "/",
        "tag_target": 1,
        "sort": 1
      }
    ],
    "total": 1
  }
}
```

---

## 2. 新增导航菜单

- **用途**：新增一条导航菜单。
- **方法路径**：`POST /tags`
- **鉴权**：需要。

### 入参（请求体，JSON）

| 名 | 类型 | 必填 | 长度/取值 | 含义 |
|----|------|------|-----------|------|
| name | string | 是 | 1–30 字符 | 导航名称（trim 后不可为空） |
| link | string | 是 | 1–255 字符 | 导航链接（trim 后不可为空） |
| sort | int | 否 | ≥ 0；为 0 时取默认值 `10` | 排序值 |
| target | int | 否 | `0` 或 `1`；缺省为 `0` | 打开方式 |

### 出参

`data` 为创建后的 [Tag 对象](#tag-对象字段)。

### 错误码

| 码 | 触发条件 |
|----|----------|
| 400 | 缺少 name/link、超长、target 非 0/1、请求体非合法 JSON |
| 401 | 未鉴权/密钥无效 |
| 429 | 触发限流 |
| 500 | 服务器内部错误 |

### 请求示例

```http
POST /api/v1/tags
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json

{
  "name": "首页",
  "link": "/",
  "sort": 1,
  "target": 1
}
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "tag_id": 1,
    "tag_name": "首页",
    "tag_link": "/",
    "tag_target": 1,
    "sort": 1
  }
}
```

---

## 3. 修改导航菜单

- **用途**：按 ID 修改导航菜单全部字段（全量更新）。
- **方法路径**：`PUT /tags/{id}`
- **鉴权**：需要。

### 入参

路径参数：

| 名 | 位置 | 类型 | 必填 | 说明 |
|----|------|------|------|------|
| id | path | int | 是 | 导航菜单 ID，正整数 |

请求体：与 [新增](#2-新增导航菜单)完全一致（name/link 必填）。

### 出参

`data` 为修改后的 [Tag 对象](#tag-对象字段)。

### 错误码

| 码 | 触发条件 |
|----|----------|
| 400 | id 非正整数，或请求体校验失败（同新增） |
| 401 | 未鉴权/密钥无效 |
| 404 | 指定 id 的导航菜单不存在 |
| 429 | 触发限流 |
| 500 | 服务器内部错误 |

### 请求示例

```http
PUT /api/v1/tags/1
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json

{
  "name": "首页导航",
  "link": "/home",
  "sort": 5,
  "target": 0
}
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "tag_id": 1,
    "tag_name": "首页导航",
    "tag_link": "/home",
    "tag_target": 0,
    "sort": 5
  }
}
```

---

## 4. 删除导航菜单

- **用途**：按 ID 删除一条导航菜单。
- **方法路径**：`DELETE /tags/{id}`
- **鉴权**：需要。

### 入参

| 名 | 位置 | 类型 | 必填 | 说明 |
|----|------|------|------|------|
| id | path | int | 是 | 导航菜单 ID，正整数 |

无请求体。

### 出参

`data` 为对象：

| 字段 | 类型 | 含义 |
|------|------|------|
| id | int | 被删除的导航菜单 ID |

### 错误码

| 码 | 触发条件 |
|----|----------|
| 400 | id 非正整数 |
| 401 | 未鉴权/密钥无效 |
| 404 | 指定 id 的导航菜单不存在 |
| 429 | 触发限流 |
| 500 | 服务器内部错误 |

### 请求示例

```http
DELETE /api/v1/tags/1
Authorization: Bearer sk-xxxxxxxx
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": { "id": 1 }
}
```
