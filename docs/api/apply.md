# Apply 收录申请域 API

收录申请（apply）域 REST 端点，对应后台「收录管理」的全部操作。

## 通用约定

- **Base URL**：`/api/v1`
- **鉴权**：全部端点均需 Bearer 密钥；请求头 `Authorization: Bearer sk-...`，缺失/无效返回 `401`。
- **统一响应信封**：

```json
{ "code": 200, "message": "ok", "data": {} }
```

- **分页结构**（`data`）：`list`（元素数组）、`page`、`page_size`、`total`。
- 请求体均为 `application/json`；参数校验失败返回 `400`，业务码同为 `400`。

### 申请元素对象 applyItem

| 字段 | 类型 | 含义 |
|------|------|------|
| id | int | 申请 ID（apply_id） |
| name | string | 申请名称 |
| url | string | 申请 URL |
| group_id | int | 所属分组 ID |
| group_name | string | 所属分组名（handler 经 GroupID→Name 映射组装；无匹配时为空串） |
| icon | string | 图标 |
| desc | string | 申请描述 |
| apply_time | string | 提交时间（RFC3339） |
| status | int | 审核状态：0=待审核 / 1=已通过 / 2=已拒绝 |

### 状态码总表

| HTTP / code | 触发条件 |
|-------------|----------|
| 400 | 参数缺失、超长、枚举非法、路径 id 非正整数 |
| 401 | 未携带或携带无效 API 密钥 |
| 404 | 指定申请记录不存在 |
| 429 | 触发 IP 限流 |
| 500 | 服务端内部错误 |

---

## 1. GET /apply —— 申请列表

分页返回收录申请，支持按审核状态过滤；列表按申请时间倒序。

### 鉴权

需要 Bearer 密钥。

### 入参（Query）

| 名称 | 类型 | 必填 | 规格 | 说明 |
|------|------|------|------|------|
| status | int | 否 | `oneof 0 1 2` | 缺省=全部；0=待审核，1=已通过，2=已拒绝 |
| page | int | 否 | ≥1，默认 1 | 页码 |
| page_size | int | 否 | 1–100，默认 10 | 每页条数 |

### 出参（data）

分页结构，`list` 元素为 [applyItem](#申请元素对象-applyitem)。

### 错误码

- `400`：status 不在 0/1/2，或分页参数非法。
- `401`：未鉴权。

### 示例

请求：

```http
GET /api/v1/apply?status=0&page=1&page_size=10
Authorization: Bearer sk-xxxx
```

响应：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 3,
        "name": "申请三",
        "url": "https://c.example.com",
        "group_id": 1,
        "group_name": "技术博客",
        "icon": "",
        "desc": "",
        "apply_time": "2026-10-04T10:00:00Z",
        "status": 0
      }
    ],
    "page": 1,
    "page_size": 10,
    "total": 3
  }
}
```

---

## 2. GET /apply/settings —— 读取收录设置

### 鉴权

需要 Bearer 密钥。

### 入参

无。

### 出参（data）

| 字段 | 类型 | 含义 |
|------|------|------|
| open | int | 收录开关：0=开启（需审核）/ 1=开启（免审核）/ 2=关闭；未配置时默认 0 |
| gg | string | 收录公告（apply_gg） |

### 示例

```json
{
  "code": 200,
  "message": "ok",
  "data": { "open": 0, "gg": "" }
}
```

---

## 3. PUT /apply/settings —— 更新收录设置

### 鉴权

需要 Bearer 密钥。

### 入参（JSON Body）

| 名称 | 类型 | 必填 | 规格 | 说明 |
|------|------|------|------|------|
| open | int | 是 | `oneof 0 1 2` | 收录开关 |
| gg | string | 否 | 最长 255 | 收录公告 |

### 出参（data）

返回保存后的收录设置（结构同 GET）。

### 错误码

- `400`：open 不在 0/1/2，或 gg 超长。
- `401`：未鉴权。

### 示例

请求：

```http
PUT /api/v1/apply/settings
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "open": 1, "gg": "欢迎提交收录" }
```

响应：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "open": 1, "gg": "欢迎提交收录" }
}
```

非法枚举（返回 400）：

```json
{ "code": 400, "message": "Key: 'applySettingsRequest.Open' ...", "data": null }
```

---

## 4. PUT /apply/:id —— 修改申请

修改指定申请的名称、URL、分组与图标。

### 鉴权

需要 Bearer 密钥。

### 入参

路径参数：

| 名称 | 类型 | 规格 |
|------|------|------|
| id | int | 正整数 |

JSON Body：

| 名称 | 类型 | 必填 | 规格 | 说明 |
|------|------|------|------|------|
| name | string | 是 | 最长 60，不可为空 | 申请名称 |
| url | string | 否 | 最长 255 | 申请 URL |
| group_id | int | 否 | — | 所属分组 ID |
| icon | string | 否 | — | 图标 |

### 出参（data）

| 字段 | 类型 | 含义 |
|------|------|------|
| id | int | 申请 ID |
| updated | bool | 固定为 `true` |
| row | object | 修改后的 [applyItem](#申请元素对象-applyitem) |

### 错误码

- `400`：id 非正整数；name 为空或超长；url 超长。
- `401`：未鉴权。
- `404`：申请记录不存在。

### 示例

```http
PUT /api/v1/apply/1
Authorization: Bearer sk-xxxx
Content-Type: application/json

{
  "name": "改名后的站点",
  "url": "https://renamed.example.com",
  "group_id": 1,
  "icon": "https://renamed.example.com/favicon.ico"
}
```

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "id": 1,
    "updated": true,
    "row": {
      "id": 1,
      "name": "改名后的站点",
      "url": "https://renamed.example.com",
      "group_id": 1,
      "group_name": "技术博客",
      "icon": "https://renamed.example.com/favicon.ico",
      "desc": "",
      "apply_time": "2026-10-04T10:00:00Z",
      "status": 0
    }
  }
}
```

---

## 5. POST /apply/:id/review —— 审核申请

对单条申请执行审核：**通过**时由 service 写入链接表（名称/URL 去除 HTML 标签，`link_order=max(id)+1`，状态启用）并将申请置为 1；**拒绝**仅将申请置为 2。

仅对 `status=0`（待审核）记录生效：对已处理记录重复调用为幂等操作（HTTP 200，不重复写链接、不改变状态）。

### 鉴权

需要 Bearer 密钥。

### 入参

路径参数：

| 名称 | 类型 | 规格 |
|------|------|------|
| id | int | 正整数 |

JSON Body：

| 名称 | 类型 | 必填 | 规格 | 说明 |
|------|------|------|------|------|
| status | int | 是 | `oneof 1 2` | 1=通过 / 2=拒绝 |

### 出参（data）

| 字段 | 类型 | 含义 |
|------|------|------|
| id | int | 申请 ID |
| status | int | 审核后申请状态（1 或 2；幂等调用时保持原状态） |

### 错误码

- `400`：status 不在 1/2；id 非正整数。
- `401`：未鉴权。
- `404`：申请记录不存在。

### 示例

审核通过：

```http
POST /api/v1/apply/1/review
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "status": 1 }
```

```json
{
  "code": 200,
  "message": "ok",
  "data": { "id": 1, "status": 1 }
}
```

> 此时链接表恰好新增 1 条链接，可经 `GET /api/v1/links?keyword=` 或 `GET /api/v1/links/all` 查询。

审核拒绝：`{ "status": 2 }` → `data: { "id": 2, "status": 2 }`，链接表无新增。

非法枚举：`{ "status": 3 }` → `400`，申请状态保持不变。

---

## 6. DELETE /apply —— 批量删除申请

按 ID 列表批量删除申请。

### 鉴权

需要 Bearer 密钥。

### 入参（JSON Body）

| 名称 | 类型 | 必填 | 规格 | 说明 |
|------|------|------|------|------|
| ids | int[] | 是 | 至少 1 个；每个元素为正整数 | 待删除申请 ID |

### 出参（data）

| 字段 | 类型 | 含义 |
|------|------|------|
| deleted | int | 本次提交删除的 ID 数量 |

### 错误码

- `400`：ids 为空数组/缺失，或包含非正整数。
- `401`：未鉴权。

### 示例

```http
DELETE /api/v1/apply
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [1, 2] }
```

```json
{
  "code": 200,
  "message": "ok",
  "data": { "deleted": 2 }
}
```
