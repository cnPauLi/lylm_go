# 分组（group）域 API

- Base URL：`/api/v1`
- 鉴权：除特别说明外，所有端点均需在请求头携带 `Authorization: Bearer sk-...`（有效 API 密钥）。缺失或无效返回 `401`。
- 统一响应信封：

```json
{ "code": 200, "message": "ok", "data": {} }
```

  - `code`：业务码，与 HTTP 状态码语义一致（成功 200）。
  - `message`：文本说明；失败时为错误原因。
  - `data`：业务数据；失败时为 `null`。

- 通用错误码：

| HTTP / code | 含义 |
|---|---|
| 400 | 请求参数不合法（JSON 语法错误、字段缺失/超长、枚举非法） |
| 401 | 未鉴权或密钥无效 |
| 500 | 服务器内部错误 |

- 分组对象（`group`）字段：

| 字段 | 类型 | 含义 |
|---|---|---|
| group_id | int | 分组 ID（主键） |
| group_name | string | 分组名称 |
| group_icon | string | 分组图标：纯 URL 已由服务端包裹为 `<img>` 标签，也可能是 SVG/HTML 片段 |
| group_order | int | 展示排序下标（从 0 开始） |
| group_status | int | 状态：1 启用 / 0 停用 |
| group_pwd | int | 关联的加密访问密码组 ID，0 表示不加密 |

---

## 1. 获取分组列表

- **用途**：返回全部分组（按 `group_order` 升序）、总数及各组链接数。
- **方法与路径**：`GET /groups`
- **鉴权**：需要。

### 入参

无。

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| total | int | 分组总数 |
| groups | array&lt;group&gt; | 分组列表（字段见上文） |
| link_count | object | 键为分组 ID，值为该组链接数量；无链接的分组可能不出现 |

### 错误码

401（未鉴权）、500（查询失败）。

### 请求示例

```http
GET /api/v1/groups
Authorization: Bearer sk-xxxx
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "total": 2,
    "groups": [
      {
        "group_id": 1,
        "group_name": "常用",
        "group_icon": "<img src=\"https://example.com/i.png\" class=\"lylme-group-img\" />",
        "group_order": 0,
        "group_status": 1,
        "group_pwd": 0
      },
      {
        "group_id": 2,
        "group_name": "工具",
        "group_icon": "<svg>...</svg>",
        "group_order": 1,
        "group_status": 0,
        "group_pwd": 0
      }
    ],
    "link_count": { "1": 12, "2": 5 }
  }
}
```

---

## 2. 新增分组

- **用途**：创建一个新分组，默认启用，排序自动追加到末尾。
- **方法与路径**：`POST /groups`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参（请求体）

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| name | body | string | 是 | 长度 1–30（trim 后不可为空白） | 分组名称 |
| icon | body | string | 否 | 最长 1000 | 图标；传纯 URL 时服务端自动包裹 `<img>`，传 SVG/HTML 片段则原样保存 |
| pwd | body | int | 否 | ≥ 0 | 加密访问密码组 ID，0（默认）表示不加密 |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| id | int | 新建分组的 ID |
| group | group | 新建后的完整分组对象 |

### 错误码

- 400：JSON 非法、`name` 缺失或为空白、字段超长。
- 401：未鉴权。
- 500：持久化失败。

### 请求示例

```http
POST /api/v1/groups
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "name": "常用", "icon": "https://example.com/i.png", "pwd": 0 }
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": 1,
    "group": {
      "group_id": 1,
      "group_name": "常用",
      "group_icon": "<img src=\"https://example.com/i.png\" class=\"lylme-group-img\" />",
      "group_order": 1,
      "group_status": 1,
      "group_pwd": 0
    }
  }
}
```

---

## 3. 修改分组

- **用途**：修改指定分组的名称、图标、密码组。
- **方法与路径**：`PUT /groups/{id}`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| id | path | int | 是 | 正整数 | 分组 ID |
| name | body | string | 是 | 长度 1–30（trim 后不可为空白） | 分组名称 |
| icon | body | string | 否 | 最长 1000 | 分组图标（规则同新增） |
| pwd | body | int | 否 | ≥ 0 | 加密访问密码组 ID，0 表示不加密 |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| id | int | 分组 ID |
| group | group | 更新后的完整分组对象 |

### 错误码

- 400：`id` 非正整数、JSON 非法、字段校验失败。
- 401：未鉴权。
- 404：分组不存在。
- 500：持久化失败。

### 请求示例

```http
PUT /api/v1/groups/1
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "name": "常用-改", "icon": "<svg></svg>", "pwd": 0 }
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": 1,
    "group": {
      "group_id": 1,
      "group_name": "常用-改",
      "group_icon": "<svg></svg>",
      "group_order": 1,
      "group_status": 1,
      "group_pwd": 0
    }
  }
}
```

---

## 4. 删除分组

- **用途**：删除指定分组；**组内全部链接会一并删除**。
- **方法与路径**：`DELETE /groups/{id}`
- **鉴权**：需要。

### 入参

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| id | path | int | 是 | 正整数 | 待删除分组 ID |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| id | int | 被删除的分组 ID |

### 错误码

- 400：`id` 非正整数。
- 401：未鉴权。
- 404：分组不存在。
- 500：删除失败。

### 请求示例

```http
DELETE /api/v1/groups/1
Authorization: Bearer sk-xxxx
```

### 响应示例

```json
{ "code": 200, "message": "ok", "data": { "id": 1 } }
```

---

## 5. 启用 / 停用分组

- **用途**：切换指定分组的启用状态。停用后该分组不在前台展示。
- **方法与路径**：`POST /groups/{id}/status`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参

| 名 | 位置 | 类型 | 必填 | 枚举 | 含义 |
|---|---|---|---|---|---|
| id | path | int | 是 | 正整数 | 分组 ID |
| status | body | int | 是 | `0`、`1` | 1 启用 / 0 停用 |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| id | int | 分组 ID |
| status | int | 写入后的状态 |

### 错误码

- 400：`id` 非正整数、`status` 非 0/1、JSON 非法。
- 401：未鉴权。
- 404：分组不存在。
- 500：更新失败。

### 请求示例

```http
POST /api/v1/groups/1/status
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "status": 0 }
```

### 响应示例

```json
{ "code": 200, "message": "ok", "data": { "id": 1, "status": 0 } }
```

---

## 6. 分组排序

- **用途**：按传入 ID 数组的顺序，从下标 0 开始依次重写各分组的 `group_order`。
- **方法与路径**：`POST /groups/sort`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参（请求体）

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| ids | body | array&lt;int&gt; | 是 | 1–200 个元素，每个为正整数 | 按新展示顺序排列的分组 ID |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| ids | array&lt;int&gt; | 本次写入顺序的分组 ID 数组 |

### 错误码

- 400：`ids` 缺失/为空、元素非正整数、数量超过 200、JSON 非法。
- 401：未鉴权。
- 500：更新失败。

### 请求示例

```http
POST /api/v1/groups/sort
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [2, 1] }
```

### 响应示例

```json
{ "code": 200, "message": "ok", "data": { "ids": [2, 1] } }
```
