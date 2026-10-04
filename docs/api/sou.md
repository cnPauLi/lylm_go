# 搜索引擎（Sou）域 API

管理首页可选搜索引擎（后台「搜索引擎设置」对应能力），共 4 个端点。

## 通用约定

- Base URL：`/api/v1`
- 鉴权：全部端点需在请求头携带 `Authorization: Bearer sk-...`；缺失或无效返回 `401`。
- 统一响应信封：

```json
{ "code": 200, "message": "ok", "data": {} }
```

失败时 HTTP 状态码与业务 `code` 一致，`data` 为 `null`。
- 通用错误码：`400` 参数校验失败、`401` 未鉴权/密钥无效、`404` 记录不存在、`429` 触发限流、`500` 服务器内部错误。

## Sou 对象字段

| 字段 | 类型 | 含义 |
|------|------|------|
| sou_id | int | 搜索引擎 ID |
| sou_alias | string | 别名（前台搜索区 `data-alias` 使用） |
| sou_name | string | 名称 |
| sou_hint | string | 输入框提示语 |
| sou_color | string | 标识颜色（如 `#2932e1`） |
| sou_link | string | PC 端搜索链接模板 |
| sou_waplink | string | 移动端搜索链接模板 |
| sou_icon | string | 图标（URL 或 SVG 名称/内容） |
| sou_st | int | 启用状态：`1` 启用，`0` 停用 |
| sou_order | int | 排序值（升序） |

---

## 1. 获取搜索引擎列表

- **用途**：返回全部搜索引擎，按 `sou_order` 升序。
- **方法路径**：`GET /sou`
- **鉴权**：需要。

### 入参

无（无查询参数、无请求体）。

### 出参

`data` 为对象：

| 字段 | 类型 | 含义 |
|------|------|------|
| list | Sou[] | 搜索引擎列表 |
| total | int | 记录总数 |

### 错误码

`401`、`429`、`500`。

### 请求示例

```http
GET /api/v1/sou
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
        "sou_id": 1,
        "sou_alias": "baidu",
        "sou_name": "百度",
        "sou_hint": "请输入关键词",
        "sou_color": "#2932e1",
        "sou_link": "https://www.baidu.com/s?wd=",
        "sou_waplink": "",
        "sou_icon": "",
        "sou_st": 1,
        "sou_order": 1
      }
    ],
    "total": 1
  }
}
```

---

## 2. 新增搜索引擎

- **用途**：新增一个搜索引擎；`sou_order` 由服务端置为「当前记录数 + 1」。
- **方法路径**：`POST /sou`
- **鉴权**：需要。

### 入参（请求体，JSON）

| 名 | 类型 | 必填 | 长度/取值 | 含义 |
|----|------|------|-----------|------|
| alias | string | 是 | 1–30 字符 | 搜索引擎别名 |
| name | string | 是 | 1–30 字符 | 搜索引擎名称 |
| hint | string | 否 | ≤ 60 字符 | 输入框提示语 |
| color | string | 否 | ≤ 20 字符 | 标识颜色 |
| link | string | 否 | ≤ 255 字符 | PC 端搜索链接模板 |
| waplink | string | 否 | ≤ 255 字符 | 移动端搜索链接模板 |
| icon | string | 否 | ≤ 2000 字符 | 图标 |
| st | bool | 否 | `true`→启用(1)，`false`→停用(0)；缺省 `false` | 启用状态 |
| order | int | 否 | ≥ 0；新增时忽略（服务端自动赋值） | 排序值 |

> 说明：原版后台仅在六字段全部为空时报错；REST 契约有意收紧为 `alias`/`name` 必填。

### 出参

`data` 为创建后的 [Sou 对象](#sou-对象字段)。

### 错误码

| 码 | 触发条件 |
|----|----------|
| 400 | 缺少 alias/name、字段超长、请求体非合法 JSON |
| 401 | 未鉴权/密钥无效 |
| 429 | 触发限流 |
| 500 | 服务器内部错误 |

### 请求示例

```http
POST /api/v1/sou
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json

{
  "alias": "baidu",
  "name": "百度",
  "hint": "请输入关键词",
  "color": "#2932e1",
  "link": "https://www.baidu.com/s?wd=",
  "st": true
}
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "sou_id": 1,
    "sou_alias": "baidu",
    "sou_name": "百度",
    "sou_hint": "请输入关键词",
    "sou_color": "#2932e1",
    "sou_link": "https://www.baidu.com/s?wd=",
    "sou_waplink": "",
    "sou_icon": "",
    "sou_st": 1,
    "sou_order": 1
  }
}
```

---

## 3. 修改搜索引擎

- **用途**：按 ID 修改搜索引擎全部字段（全量更新，含 `sou_order`）。
- **方法路径**：`PUT /sou/{id}`
- **鉴权**：需要。

### 入参

路径参数：

| 名 | 位置 | 类型 | 必填 | 说明 |
|----|------|------|------|------|
| id | path | int | 是 | 搜索引擎 ID，正整数 |

请求体：与 [新增](#2-新增搜索引擎)一致；此端点 `order` 生效。

### 出参

`data` 为修改后的 [Sou 对象](#sou-对象字段)。

### 错误码

| 码 | 触发条件 |
|----|----------|
| 400 | id 非正整数，或请求体校验失败（同新增） |
| 401 | 未鉴权/密钥无效 |
| 404 | 指定 id 的搜索引擎不存在 |
| 429 | 触发限流 |
| 500 | 服务器内部错误 |

### 请求示例

```http
PUT /api/v1/sou/1
Authorization: Bearer sk-xxxxxxxx
Content-Type: application/json

{
  "alias": "baidu",
  "name": "百度搜索",
  "hint": "请输入关键词",
  "color": "#2932e1",
  "link": "https://www.baidu.com/s?wd=",
  "st": false,
  "order": 3
}
```

### 响应示例

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "sou_id": 1,
    "sou_alias": "baidu",
    "sou_name": "百度搜索",
    "sou_hint": "请输入关键词",
    "sou_color": "#2932e1",
    "sou_link": "https://www.baidu.com/s?wd=",
    "sou_waplink": "",
    "sou_icon": "",
    "sou_st": 0,
    "sou_order": 3
  }
}
```

---

## 4. 删除搜索引擎

- **用途**：按 ID 删除一个搜索引擎。
- **方法路径**：`DELETE /sou/{id}`
- **鉴权**：需要。

### 入参

| 名 | 位置 | 类型 | 必填 | 说明 |
|----|------|------|------|------|
| id | path | int | 是 | 搜索引擎 ID，正整数 |

无请求体。

### 出参

`data` 为对象：

| 字段 | 类型 | 含义 |
|------|------|------|
| id | int | 被删除的搜索引擎 ID |

### 错误码

| 码 | 触发条件 |
|----|----------|
| 400 | id 非正整数 |
| 401 | 未鉴权/密钥无效 |
| 404 | 指定 id 的搜索引擎不存在 |
| 429 | 触发限流 |
| 500 | 服务器内部错误 |

### 请求示例

```http
DELETE /api/v1/sou/1
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
