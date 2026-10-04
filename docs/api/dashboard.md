# Dashboard / Files 域 API

- Base URL：`/api/v1`
- 鉴权：全部端点均需 `Authorization: Bearer sk-...`（API Key，后台「接口与 MCP」页管理）
- 统一响应信封：

```json
{ "code": 200, "message": "ok", "data": {} }
```

- 全局限流：默认每 IP 120 次/分钟，超限返回 `429`。

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/dashboard` | 仪表盘计数与版本信息 |
| GET | `/files/cleanup` | 扫描 files 目录下的未使用文件 |
| POST | `/files/cleanup` | 按 rel 列表删除未使用文件 |

---

## GET /dashboard

返回后台仪表盘汇总数据：链接/分组/待审申请计数与版本信息。

### 鉴权

需要 Bearer API Key。

### 入参

无（无 query、无 body）。

### 出参（data）

| 字段 | 类型 | 含义 |
|------|------|------|
| `links` | integer | 链接总数（`lylme_links` 行数） |
| `groups` | integer | 分组总数（`lylme_groups` 行数） |
| `apply_rows` | integer | 待审核收录申请数（`apply_status=0`） |
| `app_version` | string | 后端程序版本 |
| `db_version` | string | 数据库版本（配置键 `version`，未设置时为空串） |
| `build` | string | 建站时间（配置键 `build`，未设置时为空串） |

### 错误码

| HTTP / code | 触发条件 |
|------|------|
| 401 | 缺失/格式错误/无效的 API Key |
| 429 | 触发 IP 限流 |

### 示例

请求：

```bash
curl -H "Authorization: Bearer sk-xxxx" \
  https://example.com/api/v1/dashboard
```

响应：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "links": 128,
    "groups": 9,
    "apply_rows": 3,
    "app_version": "v2.0.0",
    "db_version": "1.6",
    "build": "2024-05-01"
  }
}
```

---

## GET /files/cleanup

扫描 `files/download` 与 `files/upload` 目录，与数据库图标引用
（`lylme_groups.group_icon`、`lylme_links.icon`）逐一比对：
文件名（大小写不敏感）出现在任一引用中即判定为已使用；仅未使用文件进入文件列表。

### 鉴权

需要 Bearer API Key。

### 入参

无（无 query、无 body）。

### 出参（data）

| 字段 | 类型 | 含义 |
|------|------|------|
| `total` | integer | 扫描到的文件总数（含已使用，不含子目录） |
| `used` | integer | 被图标引用的文件数 |
| `unused` | integer | 未使用文件数 |
| `total_size` | integer | 未使用文件总大小（字节） |
| `total_size_str` | string | 未使用文件总大小（人类可读，如 `1.01 KB`） |
| `dirs` | array | 分目录报告，固定顺序 `download`、`upload` |

`dirs[]`：

| 字段 | 类型 | 含义 |
|------|------|------|
| `name` | string | 目录名：`download` 或 `upload` |
| `unused` | integer | 该目录未使用文件数 |
| `files` | array | 未使用文件列表（无未使用文件时为 `[]`） |

`dirs[].files[]`：

| 字段 | 类型 | 含义 |
|------|------|------|
| `rel` | string | 相对 `files` 的路径，如 `download/a.png`；即 POST 删除时使用的标识 |
| `name` | string | 文件名 |
| `dir` | string | 所在目录：`download` / `upload` |
| `url` | string | 文件访问 URL：`/files/<rel>` |
| `ext` | string | 小写扩展名（不含点），无扩展名时为空串 |
| `size` | integer | 文件大小（字节） |
| `size_str` | string | 文件大小（人类可读，如 `12.34 KB`） |
| `is_img` | boolean | 是否为图片（png/jpg/jpeg/gif/ico/webp） |

> 说明：报告中的 `files` 只列出**未使用**文件；`total`/`used` 为含已使用文件的计数。
> 目录不存在时该目录按空统计，不构成错误。

### 错误码

| HTTP / code | 触发条件 |
|------|------|
| 401 | 缺失/格式错误/无效的 API Key |
| 429 | 触发 IP 限流 |
| 500 | 数据库引用查询失败 |

### 示例

请求：

```bash
curl -H "Authorization: Bearer sk-xxxx" \
  https://example.com/api/v1/files/cleanup
```

响应：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "total": 4,
    "used": 1,
    "unused": 3,
    "total_size": 1030,
    "total_size_str": "1.01 KB",
    "dirs": [
      {
        "name": "download",
        "unused": 2,
        "files": [
          {
            "rel": "download/unused1.png",
            "name": "unused1.png",
            "dir": "download",
            "url": "/files/download/unused1.png",
            "ext": "png",
            "size": 1,
            "size_str": "1 B",
            "is_img": true
          },
          {
            "rel": "download/unused2.txt",
            "name": "unused2.txt",
            "dir": "download",
            "url": "/files/download/unused2.txt",
            "ext": "txt",
            "size": 5,
            "size_str": "5 B",
            "is_img": false
          }
        ]
      },
      {
        "name": "upload",
        "unused": 1,
        "files": [
          {
            "rel": "upload/up.jpg",
            "name": "up.jpg",
            "dir": "upload",
            "url": "/files/upload/up.jpg",
            "ext": "jpg",
            "size": 1024,
            "size_str": "1.00 KB",
            "is_img": true
          }
        ]
      }
    ]
  }
}
```

---

## POST /files/cleanup

按 `rels` 列表删除文件。逐条校验，单条失败不阻断其余条目；非法路径不会产生任何文件操作。

合法 `rel` 规则：

- 正则 `^(download|upload)/[^/]+$`：仅允许 `download`/`upload` 单层目录下的单个文件；
- 不得包含目录穿越片段 `..`；
- 目标必须为已存在的普通文件。

### 鉴权

需要 Bearer API Key。

### 入参（JSON body）

| 字段 | 类型 | 必填 | 约束 | 含义 |
|------|------|------|------|------|
| `rels` | string array | 是 | 至少 1 项 | 待删除文件的 rel 列表（取自 GET 扫描结果的 `rel`） |

请求体示例：

```json
{ "rels": ["download/unused1.png", "upload/up.jpg"] }
```

### 出参（data）

| 字段 | 类型 | 含义 |
|------|------|------|
| `deleted` | integer | 成功删除数 |
| `failed` | integer | 失败数：非法路径、目录穿越、文件不存在或删除失败均计入 |

### 错误码

| HTTP / code | 触发条件 |
|------|------|
| 400 | 请求体不是合法 JSON；缺少 `rels`；`rels` 为空数组 |
| 401 | 缺失/格式错误/无效的 API Key |
| 429 | 触发 IP 限流 |

> 注意：`rels` 中个别元素非法**不**返回 4xx，而是正常返回 200 并计入 `failed`；
> 仅当整个 `rels` 列表缺失/为空或请求体无法解析时才返回 400。

### 示例

请求：

```bash
curl -X POST \
  -H "Authorization: Bearer sk-xxxx" \
  -H "Content-Type: application/json" \
  -d '{"rels":["download/unused1.png","../x"]}' \
  https://example.com/api/v1/files/cleanup
```

响应：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "deleted": 1,
    "failed": 1
  }
}
```

空列表被拒绝：

```bash
curl -X POST \
  -H "Authorization: Bearer sk-xxxx" \
  -H "Content-Type: application/json" \
  -d '{"rels":[]}' \
  https://example.com/api/v1/files/cleanup
```

```json
{
  "code": 400,
  "message": "请求体无效：rels 为必填项且至少包含 1 个路径",
  "data": null
}
```
