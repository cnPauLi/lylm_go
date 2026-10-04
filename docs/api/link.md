# 链接（link）域 API

- Base URL：`/api/v1`
- 鉴权：除特别说明外，所有端点均需在请求头携带 `Authorization: Bearer sk-...`（有效 API 密钥）。缺失或无效返回 `401`。
- 统一响应信封：

```json
{ "code": 200, "message": "ok", "data": {} }
```

  - `code`：业务码，与 HTTP 状态码语义一致。
  - `message`：文本说明；失败时为错误原因。
  - `data`：业务数据；失败时为 `null`。

- 通用错误码：

| HTTP / code | 含义 |
|---|---|
| 400 | 请求参数不合法（JSON 语法错误、字段缺失/超长、枚举非法） |
| 401 | 未鉴权或密钥无效 |
| 404 | 链接不存在 |
| 409 | 链接 URL 已存在（创建冲突） |
| 500 | 服务器内部错误 |

- 链接对象（`link`）字段：

| 字段 | 类型 | 含义 |
|---|---|---|
| id | int | 链接 ID（主键） |
| name | string | 链接名称（可能含 `<font color="...">` 颜色包裹） |
| group_id | int | 所属分组 ID，0 为未分组 |
| url | string | 链接地址 |
| icon | string | 链接图标（URL / 类名 / SVG 等） |
| link_desc | string | 链接描述 |
| link_keywords | string | 检索关键词 |
| link_order | int | 组内排序下标 |
| link_status | int | 状态：1 启用 / 0 停用 |
| link_pwd | int | 访问密码组 ID，0 为无密码 |

- 链接简表（`brief`）字段：`id`（int）、`url`（string）、`name`（string）。

---

## 1. 分页查询链接

- **用途**：分页返回链接列表，支持按分组过滤与按关键词检索。
- **方法与路径**：`GET /links`
- **鉴权**：需要。

### 入参（query）

| 名 | 位置 | 类型 | 必填 | 默认 | 约束 | 含义 |
|---|---|---|---|---|---|---|
| page | query | int | 否 | 1 | ≥ 1 | 页码，从 1 开始 |
| page_size | query | int | 否 | 10 | 1–100（超出自动收敛到 100） | 每页条数 |
| group_id | query | int | 否 | 0（不过滤） | ≥ 0 | 仅返回该分组下的链接 |
| keyword | query | string | 否 | 空 | trim 后非空即生效 | 在名称、URL、描述、关键词中做包含匹配 |

> `keyword` 非空时走全站关键词检索，此时 `group_id` 不参与过滤。

### 出参 `data`（分页结构）

| 字段 | 类型 | 含义 |
|---|---|---|
| list | array&lt;link&gt; | 当前页链接（按 id 升序） |
| page | int | 当前页码 |
| page_size | int | 每页条数 |
| total | int | 满足条件的记录总数 |

### 错误码

401（未鉴权）、500（查询失败）。非数值的分页参数会回退为默认值，不报错。

### 请求示例

```http
GET /api/v1/links?page=1&page_size=10&group_id=1&keyword=导航
Authorization: Bearer sk-xxxx
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 1,
        "name": "六零导航",
        "group_id": 1,
        "url": "https://lylme.com",
        "icon": "",
        "link_desc": "导航站",
        "link_keywords": "导航,主页",
        "link_order": 1,
        "link_status": 1,
        "link_pwd": 0
      }
    ],
    "page": 1,
    "page_size": 10,
    "total": 1
  }
}
```

---

## 2. 获取全量链接简表

- **用途**：返回全站链接的精简列表（仅 id/url/name，按 id 升序），适合下拉选择、批量校验等场景。
- **方法与路径**：`GET /links/all`
- **鉴权**：需要。

### 入参

无。

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| count | int | 链接数量 |
| links | array&lt;brief&gt; | 简表数组 |

### 错误码

401（未鉴权）、500（查询失败）。

### 请求示例

```http
GET /api/v1/links/all
Authorization: Bearer sk-xxxx
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "count": 2,
    "links": [
      { "id": 1, "url": "https://lylme.com", "name": "六零导航" },
      { "id": 2, "url": "https://second.example", "name": "第二个" }
    ]
  }
}
```

---

## 3. 获取链接详情

- **用途**：按 ID 读取单条链接的完整字段。
- **方法与路径**：`GET /links/{id}`
- **鉴权**：需要。

### 入参

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| id | path | int | 是 | 正整数 | 链接 ID |

### 出参 `data`

链接对象（`link`，字段见上文）。

### 错误码

- 400：`id` 非正整数。
- 401：未鉴权。
- 404：链接不存在。

### 请求示例

```http
GET /api/v1/links/1
Authorization: Bearer sk-xxxx
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": 1,
    "name": "六零导航",
    "group_id": 0,
    "url": "https://lylme.com",
    "icon": "",
    "link_desc": "导航站",
    "link_keywords": "导航,主页",
    "link_order": 1,
    "link_status": 1,
    "link_pwd": 0
  }
}
```

---

## 4. 新增链接

- **用途**：创建一条链接，默认启用，排序追加到末尾。
- **方法与路径**：`POST /links`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参（请求体）

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| name | body | string | 是 | 最长 255（trim 后不可为空白） | 链接名称 |
| url | body | string | 是 | 最长 255（trim 后不可为空白） | 链接地址 |
| color | body | string | 否 | 最长 32 | 名称颜色；非空时名称被 `<font color="...">` 包裹 |
| icon | body | string | 否 | 最长 2000 | 链接图标 |
| group_id | body | int | 否 | ≥ 0 | 所属分组 ID，0（默认）为未分组 |
| link_desc | body | string | 否 | 最长 255 | 链接描述 |
| link_keywords | body | string | 否 | 最长 512 | 检索关键词 |
| pwd | body | int | 否 | ≥ 0 | 访问密码组 ID，0（默认）为无密码 |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| id | int | 新建链接的 ID |

### URL 重复（409）

服务端按去掉末尾 `/` 的 URL 判重（即 `https://a.com` 与 `https://a.com/` 视为同一地址）。命中已存在链接时：

- HTTP 状态码与业务码均为 **409**；
- `message` 固定为 **`链接已存在`**；
- `data` 为 `{ "id": <已存在链接 ID> }`。

### 错误码

- 400：JSON 非法、`name`/`url` 缺失或为空白、字段超长。
- 401：未鉴权。
- 409：URL 已存在。
- 500：持久化失败。

### 请求示例

```http
POST /api/v1/links
Authorization: Bearer sk-xxxx
Content-Type: application/json

{
  "name": "六零导航",
  "url": "https://lylme.com",
  "color": "",
  "icon": "",
  "group_id": 0,
  "link_desc": "导航站",
  "link_keywords": "导航,主页",
  "pwd": 0
}
```

### 响应示例（成功）

```json
{ "code": 200, "message": "ok", "data": { "id": 1 } }
```

### 响应示例（重复）

```json
{
  "code": 409,
  "message": "链接已存在",
  "data": { "id": 1 }
}
```

---

## 5. 修改链接

- **用途**：修改指定链接的全部字段（含访问密码组）。
- **方法与路径**：`PUT /links/{id}`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| id | path | int | 是 | 正整数 | 链接 ID |
| name | body | string | 是 | 最长 255（trim 后不可为空白） | 链接名称 |
| url | body | string | 是 | 最长 255（trim 后不可为空白） | 链接地址 |
| color | body | string | 否 | 最长 32 | 名称颜色 |
| icon | body | string | 否 | 最长 2000 | 链接图标 |
| group_id | body | int | 否 | ≥ 0 | 所属分组 ID |
| link_desc | body | string | 否 | 最长 255 | 链接描述 |
| link_keywords | body | string | 否 | 最长 512 | 检索关键词 |
| pwd | body | int | 否 | ≥ 0 | 访问密码组 ID |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| id | int | 链接 ID |
| link | link | 更新后的完整链接对象 |

### 错误码

- 400：`id` 非正整数、JSON 非法、字段校验失败。
- 401：未鉴权。
- 404：链接不存在。
- 500：持久化失败。

### 请求示例

```http
PUT /api/v1/links/1
Authorization: Bearer sk-xxxx
Content-Type: application/json

{
  "name": "六零导航-改",
  "url": "https://lylme.net",
  "color": "blue",
  "icon": "i",
  "group_id": 1,
  "link_desc": "新描述",
  "link_keywords": "新关键词",
  "pwd": 0
}
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": 1,
    "link": {
      "id": 1,
      "name": "<font color=\"blue\">六零导航-改</font>",
      "group_id": 1,
      "url": "https://lylme.net",
      "icon": "i",
      "link_desc": "新描述",
      "link_keywords": "新关键词",
      "link_order": 1,
      "link_status": 1,
      "link_pwd": 0
    }
  }
}
```

---

## 6. 批量删除链接

- **用途**：按 ID 数组批量删除链接。
- **方法与路径**：`DELETE /links`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参（请求体）

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| ids | body | array&lt;int&gt; | 是 | 1–500 个元素，每个为正整数 | 待删除链接 ID |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| ids | array&lt;int&gt; | 本次请求的 ID 数组 |

### 错误码

- 400：`ids` 缺失/为空、元素非正整数、数量超过 500、JSON 非法。
- 401：未鉴权。
- 500：删除失败。

> ID 不存在不会报错（数据库影响 0 行），不单独返回 404。

### 请求示例

```http
DELETE /api/v1/links
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [2, 3] }
```

### 响应示例

```json
{ "code": 200, "message": "ok", "data": { "ids": [2, 3] } }
```

---

## 7. 批量启用 / 停用链接

- **用途**：批量切换链接的启用状态。
- **方法与路径**：`POST /links/status`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参（请求体）

| 名 | 位置 | 类型 | 必填 | 约束 / 枚举 | 含义 |
|---|---|---|---|---|---|
| ids | body | array&lt;int&gt; | 是 | 1–500 个正整数 | 链接 ID |
| status | body | int | 是 | `0`、`1` | 1 启用 / 0 停用 |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| ids | array&lt;int&gt; | 链接 ID 数组 |
| status | int | 写入后的状态 |

### 错误码

400（参数非法、枚举非 0/1）、401（未鉴权）、500（更新失败）。

### 请求示例

```http
POST /api/v1/links/status
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [1, 2], "status": 0 }
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": { "ids": [1, 2], "status": 0 }
}
```

---

## 8. 批量移动分组

- **用途**：将多条链接移动到指定分组。
- **方法与路径**：`POST /links/group`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参（请求体）

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| ids | body | array&lt;int&gt; | 是 | 1–500 个正整数 | 待移动链接 ID |
| group_id | body | int | 是 | ≥ 1 | 目标分组 ID |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| ids | array&lt;int&gt; | 链接 ID 数组 |
| group_id | int | 目标分组 ID |

### 错误码

400（参数非法、`group_id` 缺失或 ≤ 0）、401（未鉴权）、500（更新失败）。

> 服务端不校验目标分组是否真实存在，调用方应自行保证。

### 请求示例

```http
POST /api/v1/links/group
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [1, 2], "group_id": 3 }
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": { "ids": [1, 2], "group_id": 3 }
}
```

---

## 9. 批量设置访问密码

- **用途**：为多条链接设置或取消访问密码组。
- **方法与路径**：`POST /links/pwd`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参（请求体）

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| ids | body | array&lt;int&gt; | 是 | 1–500 个正整数 | 链接 ID |
| pwd | body | int | 是 | ≥ 0 | 密码组 ID；0 表示取消密码 |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| ids | array&lt;int&gt; | 链接 ID 数组 |
| pwd | int | 密码组 ID |

### 错误码

400（参数非法）、401（未鉴权）、500（更新失败）。

### 请求示例

```http
POST /api/v1/links/pwd
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [1, 2], "pwd": 0 }
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": { "ids": [1, 2], "pwd": 0 }
}
```

---

## 10. 链接排序

- **用途**：按传入的 ID 字符串数组顺序，从下标 0 开始重写各链接的 `link_order`。
- **方法与路径**：`POST /links/sort`
- **鉴权**：需要。
- **Content-Type**：`application/json`

### 入参（请求体）

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| ordered | body | array&lt;string&gt; | 是 | 1–500 个非空元素，元素须为可解析的链接 ID 字符串 | 按新顺序排列的链接 ID（字符串形式） |

> 数组中无法解析为整数的元素会被服务端跳过，其余元素按下标赋值。

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| ordered | array&lt;string&gt; | 本次请求的顺序数组 |

### 错误码

400（`ordered` 缺失/为空、数量超过 500、JSON 非法）、401（未鉴权）、500（更新失败）。

### 请求示例

```http
POST /api/v1/links/sort
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ordered": ["2", "1"] }
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": { "ordered": ["2", "1"] }
}
```

---

## 11. 抓取目标站点 head

- **用途**：服务端抓取目标 URL 的站点标题、图标、描述、关键词等信息，供新增链接时自动填充。含 SSRF 防护（仅允许公网 http/https 地址）。
- **方法与路径**：`GET /links/fetch`
- **鉴权**：需要。

### 入参（query）

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| url | query | string | 是 | http/https 协议且必须包含主机名 | 待抓取的页面地址，需做 URL 编码 |

### 出参 `data`

| 字段 | 类型 | 含义 |
|---|---|---|
| title | string | 站点标题 |
| icon | string | 站点图标完整 URL |
| description | string | meta description |
| keywords | string | meta keywords |
| charset | string | 字符集，固定 `UTF-8` |
| url | string | 回显请求的 URL |
| code | int | 抓取层状态码（200 成功） |
| msg | string | 抓取层说明 |

### 错误码

- 400：`url` 缺失、格式非法（非 http/https、缺少主机名）。
- 401：未鉴权。
- 500：目标站点无法访问或抓取失败（标题与图标均为空）。

### 请求示例

```http
GET /api/v1/links/fetch?url=https%3A%2F%2Flylme.com
Authorization: Bearer sk-xxxx
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "charset": "UTF-8",
    "code": 200,
    "description": "六零导航页",
    "icon": "https://lylme.com/favicon.ico",
    "keywords": "导航",
    "msg": "请求成功",
    "title": "六零导航页",
    "url": "https://lylme.com"
  }
}
```
