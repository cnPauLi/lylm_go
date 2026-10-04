# 六零导航页 API v1 接口文档（REST + MCP）

> 版本：**v1**　｜　最后更新：2026-10-04
> 对应实现：`internal/api`（REST）、`internal/mcp`（MCP）、`internal/service`（业务逻辑）

## 目录

1. [概述](#1-概述)
2. [通用约定](#2-通用约定)
3. [REST 端点](#3-rest-端点)
   - 3.1 [自检 Ping](#31-自检-ping)
   - 3.2 [分组 Groups](#32-分组-groups)
   - 3.3 [链接 Links](#33-链接-links)
   - 3.4 [导航菜单 Tags](#34-导航菜单-tags)
   - 3.5 [搜索引擎 Sou](#35-搜索引擎-sou)
   - 3.6 [加密组 Pwd](#36-加密组-pwd)
   - 3.7 [设置域 Settings（含 about/account/system/wxplus/license/apikey）](#37-设置域-settings)
   - 3.8 [收录申请 Apply](#38-收录申请-apply)
   - 3.9 [主题 Themes](#39-主题-themes)
   - 3.10 [仪表盘与文件清理 Dashboard / Files](#310-仪表盘与文件清理-dashboard--files)
4. [MCP 服务](#4-mcp-服务)
5. [变更记录](#5-变更记录)

---

## 1. 概述

六零导航页 Go 版第二期将后台全部管理操作以标准化接口对外开放，包含两种等价接入方式：

- **REST API**：HTTP + JSON，前缀 `/api/v1`，覆盖分组、链接、导航菜单、搜索引擎、加密组、网站设置、收录申请、主题、仪表盘、文件清理等全部后台管理操作；
- **MCP 服务**：在 `/mcp` 提供 [Model Context Protocol](https://modelcontextprotocol.io/) Streamable HTTP（无状态）服务，**56 个工具与 REST 业务操作一一对应**，供 AI 客户端 / Agent 调用。

两种方式使用**同一把 `sk-` 密钥**鉴权，由后台「接口与 MCP」页面统一管理。

**规模统计**

| 项 | 数量 |
|---|---|
| REST 已注册路由 | **57**（含 1 个自检端点 `GET /ping`） |
| REST 业务端点 | 56 |
| MCP 工具 | **56** |

---

## 2. 通用约定

### 2.1 Base URL

- Base URL 为**相对站点根**的 `/api/v1`，例如站点为 `https://example.com`，则：

```
https://example.com/api/v1
```

- 本文所有端点路径均写为相对 `/api/v1` 的形式（如 `/groups` 表示 `/api/v1/groups`）。

### 2.2 鉴权

- `/api/v1/**` 与 `/mcp` 的**全部请求**强制鉴权（包括自检端点）。
- 凭证通过 HTTP 请求头提供：

```http
Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

- 密钥格式：`sk-` 前缀 + 48 个十六进制字符（随机部分 24 字节、192 bit 熵，`crypto/rand` 生成）。
- **密钥获取与管理**：登录后台，在「接口与 MCP」页面可：首次生成密钥、查看/复制当前密钥、重新生成密钥。全站仅**单一有效密钥**；**重新生成后旧密钥立即、永久失效**，所有客户端必须更换新密钥。
- 服务端使用恒定时间比较（`crypto/subtle`）校验密钥；密钥不会出现在任何日志中。
- **401 触发条件**：
  - 缺少 `Authorization` 头；
  - 不是 `Bearer` 方案（头值不以 `Bearer ` 开头）；
  - token 不以 `sk-` 开头；
  - 密钥不存在、不匹配，或已被重新生成操作作废。

### 2.3 统一响应信封

所有 REST 响应（无论成功或失败）均为如下信封：

```json
{
  "code": 200,
  "message": "ok",
  "data": {}
}
```

| 字段 | 类型 | 含义 |
|---|---|---|
| `code` | int | 业务码，与 HTTP 状态码一致；成功为 `200` |
| `message` | string | 文本说明；成功为 `ok`，失败为具体错误原因 |
| `data` | any / null | 业务数据；失败时固定为 `null` |

### 2.4 分页结构

分页列表端点的 `data` 统一为：

```json
{
  "list": [],
  "page": 1,
  "page_size": 10,
  "total": 0
}
```

| 字段 | 类型 | 含义 |
|---|---|---|
| `list` | array | 当前页元素 |
| `page` | int | 当前页码（从 1 开始） |
| `page_size` | int | 每页条数 |
| `total` | int64 | 满足条件的记录总数 |

### 2.5 错误码总表

| HTTP | `code` | 触发条件 |
|---|---|---|
| **400** | 400 | 请求体不是合法 JSON；参数绑定/校验失败（`required` 必填、`max` 长度、`min`、`oneof` 枚举、类型不符）；业务规则失败：旧密码错误（`ErrWrongPassword`）、未做出任何更改（`ErrNoChange`）等。`message` 含具体字段与原因 |
| **401** | 401 | 见 [2.2 鉴权](#22-鉴权)：缺少/格式错误的 `Authorization` 头，或密钥无效、已失效 |
| **404** | 404 | 路径 `id` 对应的记录不存在（分组 / 链接 / 导航菜单 / 搜索引擎 / 加密组 / 收录申请 / 主题） |
| **409** | 409 | 仅 `POST /links`：提交的 URL（忽略末尾斜杠）已存在；`message` 固定 `链接已存在`，`data` 为 `{"id": <已存在链接 ID>}` |
| **429** | 429 | 超过限流阈值（见 [2.6](#26-限流)）；`message` 为 `请求过于频繁，请稍后再试` |
| **500** | 500 | 未预期的内部错误（如持久化失败；`GET /links/fetch` 目标站标题与图标均为空等）；经统一错误映射的 500 响应 `message` 固定为 `internal error` |

> 不存在 403 返回：系统中无任何代码路径产生 403。

### 2.6 限流

- 默认阈值：**每个 IP 每分钟 120 次请求**（固定窗口，窗口 1 分钟）。
- REST（`/api/v1`）与 MCP（`/mcp`）共用同一 IP 计数桶；IP 经 `webutil.RealIP` 解析（支持反向代理场景）。
- 超过阈值的请求返回 HTTP **429**（信封见错误码总表）。
- `/api/v1`、`/mcp` 请求**不计入前台访问统计 PV**。

### 2.7 数据格式

- 请求体与响应体均使用 `Content-Type: application/json; charset=utf-8`。
- 带请求体的写操作（POST/PUT/DELETE body）必须为 JSON 对象；无请求体的端点不需要 `Content-Type`。
- 时间字段统一为 **RFC3339** 字符串（如 `2026-10-04T10:00:00+08:00`）。

### 2.8 通用对象结构

下文端点引用下列对象，字段名与实际 JSON 输出完全一致。

**Group 对象**（`lylme_groups`）

| 字段 | 类型 | 含义 |
|---|---|---|
| `group_id` | int | 分组 ID（主键） |
| `group_name` | string | 分组名称 |
| `group_icon` | string | 分组图标：纯 URL 已由服务端包裹为 `<img src="..." class="lylme-group-img" />`；也可能是 SVG/HTML 片段 |
| `group_order` | int | 展示排序下标（从 0 开始） |
| `group_status` | int | 状态：`1` 启用 / `0` 停用 |
| `group_pwd` | int | 关联的加密访问密码组 ID，`0` 表示不加密 |

**Link 对象**（`lylme_links`）

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 链接 ID（主键） |
| `name` | string | 链接名称（指定 `color` 时可能含 `<font color="...">` 包裹） |
| `group_id` | int | 所属分组 ID，`0` 为未分组 |
| `url` | string | 链接地址 |
| `icon` | string | 链接图标（URL / 类名 / SVG 等） |
| `link_desc` | string | 链接描述 |
| `link_keywords` | string | 检索关键词 |
| `link_order` | int | 组内排序下标 |
| `link_status` | int | 状态：`1` 启用 / `0` 停用 |
| `link_pwd` | int | 访问密码组 ID，`0` 为无密码 |

**LinkBrief 链接简表**

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 链接 ID |
| `url` | string | 链接地址 |
| `name` | string | 链接名称 |

**Tag 对象**（`lylme_tags`）

| 字段 | 类型 | 含义 |
|---|---|---|
| `tag_id` | int | 导航菜单 ID |
| `tag_name` | string | 导航名称 |
| `tag_link` | string | 导航链接 |
| `tag_target` | int | 打开方式：`1` 新标签页 / `0` 当前页 |
| `sort` | int | 排序值（升序；默认 `10`） |

**Sou 对象**（`lylme_sou`）

| 字段 | 类型 | 含义 |
|---|---|---|
| `sou_id` | int | 搜索引擎 ID |
| `sou_alias` | string | 别名（前台搜索区 `data-alias` 使用） |
| `sou_name` | string | 名称 |
| `sou_hint` | string | 输入框提示语 |
| `sou_color` | string | 标识颜色（如 `#2932e1`） |
| `sou_link` | string | PC 端搜索链接模板 |
| `sou_waplink` | string | 移动端搜索链接模板 |
| `sou_icon` | string | 图标（URL 或 SVG 名称/内容） |
| `sou_st` | int | 启用状态：`1` 启用 / `0` 停用 |
| `sou_order` | int | 排序值（升序） |

**Pwd 对象**（`lylme_pwd`）

| 字段 | 类型 | 含义 |
|---|---|---|
| `pwd_id` | int | 加密组 ID |
| `pwd_name` | string | 加密组名称 |
| `pwd_key` | string | 访问密码（链接的访问口令） |
| `pwd_ps` | string | 密码提示语 |

**ApplyItem 对象**（收录申请元素，handler 组装）

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 申请 ID（`apply_id`） |
| `name` | string | 申请名称 |
| `url` | string | 申请 URL |
| `group_id` | int | 所属分组 ID |
| `group_name` | string | 所属分组名（GroupID→Name 映射；无匹配为空串） |
| `icon` | string | 图标 |
| `desc` | string | 申请描述 |
| `apply_time` | string | 提交时间，RFC3339 |
| `status` | int | 审核状态：`0` 待审核 / `1` 已通过 / `2` 已拒绝 |

**ThemeInfo 对象**（主题列表项）

| 字段 | 类型 | 含义 |
|---|---|---|
| `dir` | string | 主题目录名（稳定标识） |
| `name` | string | 主题显示名 |
| `author` | string | 作者 |
| `version` | string | 版本 |
| `description` | string | 主题说明 |

**ThemeField 对象**（主题配置字段定义）

| 字段 | 类型 | 含义 |
|---|---|---|
| `type` | string | 表单类型：`text` / `textarea` / `select` / `checkbox` / `radio` / `color` / `switch` |
| `name` | string | 配置键（主题内唯一） |
| `title` | string | 字段标题 |
| `description` | string | 字段说明（可能含 HTML） |
| `verify` | string | 校验方式：`required` / `url` / `number` / `email` 等 |
| `placeholder` | string | 占位提示 |
| `value` | string \| string[] | 默认值（checkbox 为字符串数组） |
| `enum` | array | 枚举项（select/radio/checkbox 使用，其余为 `[]`）；每项 `{"value": string, "label": string}` |

---

## 3. REST 端点

> 除 Ping 外，端点按域组织。每个端点均需 Bearer 密钥鉴权，下文中不再重复说明 401 的全部触发条件（见 [2.2](#22-鉴权)），仅列「错误码」。

### 3.1 自检 Ping

#### GET /ping

- **用途**：验证密钥与连通性。
- **鉴权**：需要 Bearer 密钥。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `pong` | bool | 固定为 `true` |

- **错误码**：401、429。
- **请求示例**：

```http
GET /api/v1/ping
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "pong": true } }
```

---

### 3.2 分组 Groups

#### 1. GET /groups

- **用途**：返回全部分组（按 `group_order` 升序）、分组总数及各组链接计数。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `total` | int64 | 分组总数 |
| `groups` | Group[] | 分组列表（结构见 [Group 对象](#28-通用对象结构)） |
| `link_count` | object | 键为分组 ID（字符串形式）、值为该组链接数量；无链接的分组可能不出现 |

- **错误码**：401、429、500（分组查询失败）。
- **请求示例**：

```http
GET /api/v1/groups
Authorization: Bearer sk-xxxx
```

- **响应示例**：

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

#### 2. POST /groups

- **用途**：新增分组；默认启用，排序自动追加到末尾（`group_order = MAX(group_order)+1`）。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `name` | body | string | 是 | 最长 30；trim 后不可为空白 | 分组名称 |
| `icon` | body | string | 否 | 最长 1000 | 图标；纯 URL 自动包裹 `<img>`，SVG/HTML 片段原样保存 |
| `pwd` | body | int | 否 | ≥ 0（默认 0） | 关联加密访问密码组 ID，0 不加密 |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 新建分组 ID |
| `group` | Group | 新建后的完整分组对象 |

- **错误码**：400（JSON 非法、`name` 缺失/空白、字段超长）、401、429、500（持久化失败）。
- **请求示例**：

```http
POST /api/v1/groups
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "name": "常用", "icon": "https://example.com/i.png", "pwd": 0 }
```

- **响应示例**：

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

#### 3. PUT /groups/:id

- **用途**：修改指定分组的名称、图标、关联加密组。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 分组 ID |
| `name` | body | string | 是 | 最长 30；trim 后不可为空白 | 分组名称 |
| `icon` | body | string | 否 | 最长 1000 | 分组图标（规则同新增） |
| `pwd` | body | int | 否 | ≥ 0 | 加密访问密码组 ID，0 不加密 |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 分组 ID |
| `group` | Group | 更新后的完整分组对象 |

- **错误码**：400（`id` 非正整数、JSON 非法、字段校验失败）、401、404（分组不存在）、429、500。
- **请求示例**：

```http
PUT /api/v1/groups/1
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "name": "常用-改", "icon": "<svg></svg>", "pwd": 0 }
```

- **响应示例**：

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

#### 4. DELETE /groups/:id

- **用途**：删除指定分组；**组内全部链接同时被级联删除**。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 待删除分组 ID |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 被删除的分组 ID |

- **错误码**：400（`id` 非正整数）、401、404（分组不存在）、429、500（删除失败）。
- **请求示例**：

```http
DELETE /api/v1/groups/1
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "id": 1 } }
```

#### 5. POST /groups/:id/status

- **用途**：启用/停用分组；停用后该分组不在前台展示。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 枚举/约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 分组 ID |
| `status` | body | int | 是 | `0`、`1` | `1` 启用 / `0` 停用 |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 分组 ID |
| `status` | int | 写入后的状态 |

- **错误码**：400（`id` 非正整数、`status` 非 0/1、JSON 非法）、401、404（分组不存在）、429、500。
- **请求示例**：

```http
POST /api/v1/groups/1/status
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "status": 0 }
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "id": 1, "status": 0 } }
```

#### 6. POST /groups/sort

- **用途**：按 `ids` 数组顺序，从下标 0 开始依次重写各分组的 `group_order`。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `ids` | body | int[] | 是 | 1–200 个元素；每个为正整数 | 按新展示顺序排列的分组 ID |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `ids` | int[] | 本次写入顺序的分组 ID 数组 |

- **错误码**：400（`ids` 缺失/为空、元素非正整数、超过 200、JSON 非法）、401、429、500。
- **请求示例**：

```http
POST /api/v1/groups/sort
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [2, 1] }
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "ids": [2, 1] } }
```

---

### 3.3 链接 Links

#### 1. GET /links

- **用途**：分页查询链接，支持按分组过滤与按关键词检索。
- **鉴权**：需要。
- **入参（Query）**：

| 名 | 位置 | 类型 | 必填 | 默认 | 约束 | 含义 |
|---|---|---|---|---|---|---|
| `page` | query | int | 否 | 1 | 非法/负数回退 1 | 页码，从 1 开始 |
| `page_size` | query | int | 否 | 10 | 非法/负数回退 10；**大于 100 自动收敛为 100** | 每页条数 |
| `group_id` | query | int | 否 | 0 | 非法/负数回退 0 | 仅返回该分组链接，0 不过滤 |
| `keyword` | query | string | 否 | 空 | trim 后非空即生效；长度无显式上限 | 在名称、URL、描述、关键词中做子串匹配 |

> `keyword` 非空时走全站关键词检索，此时 `group_id` 不参与过滤。
> 与收录申请列表不同：本端点的非数值分页/分组参数**静默回退默认值**，不返回 400。

- **出参 `data`**：分页结构（见 [2.4](#24-分页结构)），`list` 元素为 [Link 对象](#28-通用对象结构)，按 `id` 升序。
- **错误码**：401、429、500（查询失败）。
- **请求示例**：

```http
GET /api/v1/links?page=1&page_size=10&group_id=1&keyword=导航
Authorization: Bearer sk-xxxx
```

- **响应示例**：

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

#### 2. GET /links/all

- **用途**：返回全站链接简表（仅 `id`/`url`/`name`，按 `id` 升序），适合下拉选择、批量校验。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `count` | int | 链接数量 |
| `links` | LinkBrief[] | 简表数组（结构见 [LinkBrief](#28-通用对象结构)） |

- **错误码**：401、429、500。
- **请求示例**：

```http
GET /api/v1/links/all
Authorization: Bearer sk-xxxx
```

- **响应示例**：

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

#### 3. GET /links/:id

- **用途**：按 ID 读取单条链接完整字段。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 链接 ID |

- **出参 `data`**：[Link 对象](#28-通用对象结构)。
- **错误码**：400（`id` 非正整数）、401、404（链接不存在）、429。
- **请求示例**：

```http
GET /api/v1/links/1
Authorization: Bearer sk-xxxx
```

- **响应示例**：

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

#### 4. POST /links

- **用途**：新增链接；默认启用，排序追加到末尾。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `name` | body | string | 是 | 最长 255；trim 后不可为空白 | 链接名称 |
| `url` | body | string | 是 | 最长 255；trim 后不可为空白 | 链接地址 |
| `color` | body | string | 否 | 最长 32 | 名称颜色（十六进制/颜色名）；非空时名称以 `<font color="...">` 包裹 |
| `icon` | body | string | 否 | 最长 2000 | 链接图标 |
| `group_id` | body | int | 否 | ≥ 0（默认 0） | 所属分组 ID，0 未分组 |
| `link_desc` | body | string | 否 | 最长 255 | 链接描述 |
| `link_keywords` | body | string | 否 | 最长 512 | 检索关键词 |
| `pwd` | body | int | 否 | ≥ 0（默认 0） | 访问密码组 ID，0 无密码 |

- **出参 `data`（成功）**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 新建链接 ID |

- **错误码**：
  - 400：JSON 非法、`name`/`url` 缺失或空白、字段超长；
  - 401、429；
  - **409**：URL 已存在（详见下）；
  - 500：持久化失败。
- **URL 查重规则**：服务端将 URL 去掉末尾全部 `/` 后判重（`https://a.com` 与 `https://a.com/` 视为同一地址）。命中已存在链接时 HTTP 状态码与业务码均为 **409**：

```json
{
  "code": 409,
  "message": "链接已存在",
  "data": { "id": 1 }
}
```

- **请求示例**：

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

- **响应示例（成功）**：

```json
{ "code": 200, "message": "ok", "data": { "id": 1 } }
```

#### 5. PUT /links/:id

- **用途**：修改指定链接的全部字段（含访问密码组）。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 链接 ID |
| `name` | body | string | 是 | 最长 255；trim 后不可为空白 | 链接名称 |
| `url` | body | string | 是 | 最长 255；trim 后不可为空白 | 链接地址 |
| `color` | body | string | 否 | 最长 32 | 名称颜色；非空时名称被 `<font>` 包裹 |
| `icon` | body | string | 否 | 最长 2000 | 链接图标 |
| `group_id` | body | int | 否 | ≥ 0 | 所属分组 ID |
| `link_desc` | body | string | 否 | 最长 255 | 链接描述 |
| `link_keywords` | body | string | 否 | 最长 512 | 检索关键词 |
| `pwd` | body | int | 否 | ≥ 0 | 访问密码组 ID |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 链接 ID |
| `link` | Link | 更新后的完整链接对象 |

- **错误码**：400（`id` 非正整数、JSON 非法、字段校验失败）、401、404（链接不存在）、429、500。
- **请求示例**：

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

- **响应示例**：

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

#### 6. DELETE /links

- **用途**：按 ID 数组批量删除链接。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `ids` | body | int[] | 是 | 1–500 个元素；每个为正整数 | 待删除链接 ID |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `ids` | int[] | 本次请求的 ID 数组 |

- **错误码**：400（`ids` 缺失/为空、元素非正整数、超过 500、JSON 非法）、401、429、500。
- ID 不存在不报错（影响 0 行），不单独返回 404。
- **请求示例**：

```http
DELETE /api/v1/links
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [2, 3] }
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "ids": [2, 3] } }
```

#### 7. POST /links/status

- **用途**：批量启用/停用链接。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束/枚举 | 含义 |
|---|---|---|---|---|---|
| `ids` | body | int[] | 是 | 1–500 个正整数 | 链接 ID |
| `status` | body | int | 是 | `0`、`1` | `1` 启用 / `0` 停用 |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `ids` | int[] | 链接 ID 数组 |
| `status` | int | 写入后的状态 |

- **错误码**：400（参数非法、`status` 非 0/1）、401、429、500。
- **请求示例**：

```http
POST /api/v1/links/status
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [1, 2], "status": 0 }
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "ids": [1, 2], "status": 0 }
}
```

#### 8. POST /links/group

- **用途**：将多条链接批量移动到指定分组。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `ids` | body | int[] | 是 | 1–500 个正整数 | 待移动链接 ID |
| `group_id` | body | int | 是 | ≥ 1 | 目标分组 ID |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `ids` | int[] | 链接 ID 数组 |
| `group_id` | int | 目标分组 ID |

- **错误码**：400（参数非法、`group_id` 缺失或 ≤ 0）、401、429、500。
- 服务端不校验目标分组是否真实存在，调用方应自行保证。
- **请求示例**：

```http
POST /api/v1/links/group
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [1, 2], "group_id": 3 }
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "ids": [1, 2], "group_id": 3 }
}
```

#### 9. POST /links/pwd

- **用途**：为多条链接批量设置或取消访问密码组。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `ids` | body | int[] | 是 | 1–500 个正整数 | 链接 ID |
| `pwd` | body | int | 是 | ≥ 0 | 密码组 ID；`0` 取消密码 |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `ids` | int[] | 链接 ID 数组 |
| `pwd` | int | 密码组 ID |

- **错误码**：400（参数非法）、401、429、500。
- **请求示例**：

```http
POST /api/v1/links/pwd
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [1, 2], "pwd": 0 }
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "ids": [1, 2], "pwd": 0 }
}
```

#### 10. POST /links/sort

- **用途**：按 `ordered` 数组顺序，从下标 0 开始重写各链接的 `link_order`。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `ordered` | body | string[] | 是 | 1–500 个**非空**元素；元素为链接 ID 字符串 | 按新顺序排列的链接 ID（字符串形式） |

> 数组中无法解析为整数的元素会被服务端跳过，其余元素按下标赋值。

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `ordered` | string[] | 本次请求的顺序数组 |

- **错误码**：400（`ordered` 缺失/为空、元素空、超过 500、JSON 非法）、401、429、500。
- **请求示例**：

```http
POST /api/v1/links/sort
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ordered": ["2", "1"] }
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "ordered": ["2", "1"] }
}
```

#### 11. GET /links/fetch

- **用途**：服务端抓取目标 URL 的站点标题、图标、描述、关键词等信息，供新增链接时自动填充。含 SSRF 防护（仅允许公网 http/https 地址，DNS rebinding 防护）；结果带服务端缓存（缓存目录 `cache`，TTL 7 天）。
- **鉴权**：需要。
- **入参（Query）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `url` | query | string | 是 | http/https 协议且必须包含主机名；需做 URL 编码 | 待抓取页面地址 |

- **出参 `data`（抓取结果对象）**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `code` | int | 抓取层状态码，成功为 `200` |
| `msg` | string | 抓取层说明（成功为 `请求成功`） |
| `title` | string | 站点标题 |
| `charset` | string | 字符集，固定 `UTF-8` |
| `icon` | string | 站点图标完整 URL（无则空串） |
| `description` | string | meta description |
| `keywords` | string | meta keywords |
| `url` | string | 回显请求的 URL |

- **错误码**：
  - 400：`url` 缺失，或格式非法（非 http/https、缺少主机名）；
  - 401、429；
  - **500**：目标站点无法访问/抓取失败——标题与图标均为空（`message` 为 `internal error`）。
- **请求示例**：

```http
GET /api/v1/links/fetch?url=https%3A%2F%2Flylme.com
Authorization: Bearer sk-xxxx
```

- **响应示例**：

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

---

### 3.4 导航菜单 Tags

#### 1. GET /tags

- **用途**：返回全部导航菜单，按 `sort` 升序。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `list` | Tag[] | 导航菜单列表（[Tag 对象](#28-通用对象结构)） |
| `total` | int | 记录总数 |

- **错误码**：401、429、500。
- **请求示例**：

```http
GET /api/v1/tags
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "list": [
      { "tag_id": 1, "tag_name": "首页", "tag_link": "/", "tag_target": 1, "sort": 1 }
    ],
    "total": 1
  }
}
```

#### 2. POST /tags

- **用途**：新增一条导航菜单；`sort` 为 0 时取默认值 `10`。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束/枚举 | 含义 |
|---|---|---|---|---|---|
| `name` | body | string | 是 | 最长 30；trim 后不可为空白 | 导航名称 |
| `link` | body | string | 是 | 最长 255；trim 后不可为空白 | 导航链接 |
| `sort` | body | int | 否 | ≥ 0；0 → 默认 10 | 排序值（升序） |
| `target` | body | int | 否 | `0`、`1`（默认 0） | `1` 新标签页 / `0` 当前页 |

- **出参 `data`**：创建后的 [Tag 对象](#28-通用对象结构)。
- **错误码**：400（缺少 name/link、超长、target 非 0/1、JSON 非法）、401、429、500。
- **请求示例**：

```http
POST /api/v1/tags
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "name": "首页", "link": "/", "sort": 1, "target": 1 }
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "tag_id": 1, "tag_name": "首页", "tag_link": "/", "tag_target": 1, "sort": 1 }
}
```

#### 3. PUT /tags/:id

- **用途**：按 ID 修改导航菜单全部字段（全量更新）。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 导航菜单 ID |
| `name` | body | string | 是 | 同新增 | 导航名称 |
| `link` | body | string | 是 | 同新增 | 导航链接 |
| `sort` | body | int | 否 | 同新增 | 排序值 |
| `target` | body | int | 否 | `0`、`1` | 打开方式 |

- **出参 `data`**：修改后的 [Tag 对象](#28-通用对象结构)。
- **错误码**：400（id 非正整数或请求体校验失败）、401、404（菜单不存在）、429、500。
- **请求示例**：

```http
PUT /api/v1/tags/1
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "name": "首页导航", "link": "/home", "sort": 5, "target": 0 }
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "tag_id": 1, "tag_name": "首页导航", "tag_link": "/home", "tag_target": 0, "sort": 5 }
}
```

#### 4. DELETE /tags/:id

- **用途**：按 ID 删除一条导航菜单。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 导航菜单 ID |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 被删除的导航菜单 ID |

- **错误码**：400（id 非正整数）、401、404（菜单不存在）、429、500。
- **请求示例**：

```http
DELETE /api/v1/tags/1
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "id": 1 } }
```

---

### 3.5 搜索引擎 Sou

> REST 契约有意收紧：`alias` / `name` 必填（原版后台仅六个字段全空才报错）。

#### 1. GET /sou

- **用途**：返回全部搜索引擎，按 `sou_order` 升序。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `list` | Sou[] | 搜索引擎列表（[Sou 对象](#28-通用对象结构)） |
| `total` | int | 记录总数 |

- **错误码**：401、429、500。
- **请求示例**：

```http
GET /api/v1/sou
Authorization: Bearer sk-xxxx
```

- **响应示例**：

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

#### 2. POST /sou

- **用途**：新增搜索引擎；`sou_order` 由服务端置为「当前记录数 + 1」。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `alias` | body | string | 是 | 最长 30 | 搜索引擎别名（前台 `data-alias`） |
| `name` | body | string | 是 | 最长 30 | 搜索引擎名称 |
| `hint` | body | string | 否 | 最长 60 | 输入框提示语 |
| `color` | body | string | 否 | 最长 20 | 标识颜色 |
| `link` | body | string | 否 | 最长 255 | PC 端搜索链接模板 |
| `waplink` | body | string | 否 | 最长 255 | 移动端搜索链接模板 |
| `icon` | body | string | 否 | 最长 2000 | 图标（URL/SVG 名称/内容） |
| `st` | body | bool | 否 | `true`→1 启用，`false`→0 停用；缺省 `false` | 启用状态 |
| `order` | body | int | 否 | ≥ 0；**新增时忽略**（服务端自动赋值） | 排序值 |

- **出参 `data`**：创建后的 [Sou 对象](#28-通用对象结构)。
- **错误码**：400（缺少 alias/name、字段超长、JSON 非法）、401、429、500。
- **请求示例**：

```http
POST /api/v1/sou
Authorization: Bearer sk-xxxx
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

- **响应示例**：

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

#### 3. PUT /sou/:id

- **用途**：按 ID 修改搜索引擎全部字段（全量更新，`order` 生效）。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 搜索引擎 ID |
| `alias` | body | string | 是 | 最长 30 | 别名 |
| `name` | body | string | 是 | 最长 30 | 名称 |
| `hint` | body | string | 否 | 最长 60 | 提示语 |
| `color` | body | string | 否 | 最长 20 | 颜色 |
| `link` | body | string | 否 | 最长 255 | PC 模板 |
| `waplink` | body | string | 否 | 最长 255 | 移动模板 |
| `icon` | body | string | 否 | 最长 2000 | 图标 |
| `st` | body | bool | 否 | true→1 / false→0；缺省 `false` | 启用状态 |
| `order` | body | int | 否 | ≥ 0 | 排序值（此端点写入 `sou_order`，不传为 0） |

- **出参 `data`**：修改后的 [Sou 对象](#28-通用对象结构)。
- **错误码**：400（id 非正整数或请求体校验失败）、401、404（引擎不存在）、429、500。
- **请求示例**：

```http
PUT /api/v1/sou/1
Authorization: Bearer sk-xxxx
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

- **响应示例**：

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

#### 4. DELETE /sou/:id

- **用途**：按 ID 删除一个搜索引擎。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 搜索引擎 ID |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 被删除的搜索引擎 ID |

- **错误码**：400（id 非正整数）、401、404（引擎不存在）、429、500。
- **请求示例**：

```http
DELETE /api/v1/sou/1
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "id": 1 } }
```

---

### 3.6 加密组 Pwd

#### 1. GET /pwd

- **用途**：返回全部加密访问密码组。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `list` | Pwd[] | 加密组列表（[Pwd 对象](#28-通用对象结构)） |
| `total` | int | 记录总数 |

- **错误码**：401、429、500。
- **请求示例**：

```http
GET /api/v1/pwd
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "list": [
      { "pwd_id": 1, "pwd_name": "私密分组", "pwd_key": "secret", "pwd_ps": "访问提示" }
    ],
    "total": 1
  }
}
```

#### 2. POST /pwd

- **用途**：新增一个加密访问密码组。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `name` | body | string | 是 | 最长 30；trim 后不可为空白 | 加密组名称 |
| `key` | body | string | 是 | 最长 60；trim 后不可为空白 | 访问密码（链接的访问口令） |
| `ps` | body | string | 否 | 最长 60 | 密码提示语 |

- **出参 `data`**：创建后的 [Pwd 对象](#28-通用对象结构)。
- **错误码**：400（缺少 name/key、字段超长、JSON 非法）、401、429、500。
- **请求示例**：

```http
POST /api/v1/pwd
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "name": "私密分组", "key": "secret", "ps": "访问提示" }
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "pwd_id": 1, "pwd_name": "私密分组", "pwd_key": "secret", "pwd_ps": "访问提示" }
}
```

#### 3. PUT /pwd/:id

- **用途**：按 ID 修改加密组全部字段（全量更新）。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 加密组 ID |
| `name` | body | string | 是 | 同新增 | 加密组名称 |
| `key` | body | string | 是 | 同新增 | 访问密码 |
| `ps` | body | string | 否 | 最长 60 | 密码提示语 |

- **出参 `data`**：修改后的 [Pwd 对象](#28-通用对象结构)。
- **错误码**：400（id 非正整数或请求体校验失败）、401、404（加密组不存在）、429、500。
- **请求示例**：

```http
PUT /api/v1/pwd/1
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "name": "私密分组2", "key": "secret2", "ps": "新提示" }
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "pwd_id": 1, "pwd_name": "私密分组2", "pwd_key": "secret2", "pwd_ps": "新提示" }
}
```

#### 4. DELETE /pwd/:id

- **用途**：按 ID 删除一个加密组。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 加密组 ID |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 被删除的加密组 ID |

- **错误码**：400（id 非正整数）、401、404（加密组不存在）、429、500。
- **请求示例**：

```http
DELETE /api/v1/pwd/1
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "id": 1 } }
```

---

### 3.7 设置域 Settings

本域对应 `internal/api/settings_handler.go`，共 14 个端点：网站设置、关于页、管理员账号、系统安全、微信推送、网站授权、API 密钥。路径上 `/wxplus`、`/license`、`/apikey` 为顶级路径（不在 `/settings` 前缀下）。

#### 1. GET /settings

- **用途**：读取全部站点配置项（`lylme_config` 键值）。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：JSON 对象，键为配置名、值均为字符串。**敏感键显式黑名单，永远不会出现**：

| 黑名单键 | 说明 |
|---|---|
| `admin_pwd` | 管理员密码哈希 |
| `api_key` | API/MCP 接口密钥明文 |

其余配置项（含 `api_key_created_at`、`admin_user`、微信推送密钥、授权码等）照常返回。

- **错误码**：401、429。
- **请求示例**：

```http
GET /api/v1/settings
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "title": "上网导航 - LyLme Spage",
    "logo": "./assets/img/logo.png",
    "background": "./assets/img/background.jpg",
    "wap_background": "",
    "keywords": "六零导航页,导航",
    "description": "简洁高效的上网导航",
    "mode": "1",
    "snapshot": "",
    "copyright": "Copyright ©2022-2026",
    "icp": "",
    "wztj": "",
    "cdnpublic": "",
    "yan": "true",
    "tq": "false",
    "template": "default",
    "api_key_created_at": "2026-10-04T10:00:00+08:00"
  }
}
```

#### 2. PUT /settings

- **用途**：批量更新网站基本设置。
- **鉴权**：需要。
- **入参（JSON Body；全部字段可空，空串为合法值，表示清空该项）**：

| 名 | 位置 | 类型 | 必填 | 落库键 | 含义 |
|---|---|---|---|---|---|
| `title` | body | string | 否 | `title` | 网站名称 |
| `logo` | body | string | 否 | `logo` | 网站 LOGO |
| `background` | body | string | 否 | `background` | PC 背景图片 |
| `wap_background` | body | string | 否 | `wap_background` | 手机背景图片 |
| `keywords` | body | string | 否 | `keywords` | 网站关键词 |
| `description` | body | string | 否 | `description` | 网站描述 |
| `mode` | body | string | 否 | `mode` | 运行模式（原版取值 `1`/`2`；服务端不强制枚举） |
| `snapshot` | body | string | 否 | `snapshot` | 详情页快照 API |
| `copyright` | body | string | 否 | `copyright` | 底部版权 |
| `icp` | body | string | 否 | `icp` | ICP 备案号 |
| `wztj` | body | string | 否 | `wztj` | 自定义 footer |
| `cdnpublic` | body | string | 否 | `cdnpublic` | 静态资源 CDN |
| `yan` | body | string | 否 | `yan` | 随机一言：**仅恰好为字符串 `"true"` 时落 `"true"`，其余任何取值（含 `"1"`、`"yes"`、空串）一律落 `"false"`** |

> 服务端同时强制写入 `tq="false"`；本端点入参不包含 `tq`。

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `updated` | bool | 固定为 `true` |

- **错误码**：400（JSON 格式错误）、401、429、500。
- **请求示例**：

```http
PUT /api/v1/settings
Authorization: Bearer sk-xxxx
Content-Type: application/json

{
  "title": "我的导航",
  "logo": "./logo.png",
  "background": "./bg.jpg",
  "wap_background": "",
  "keywords": "导航,搜索",
  "description": "这是我的导航站",
  "mode": "2",
  "snapshot": "https://snap.example.com/?url=",
  "copyright": "© 2026",
  "icp": "京ICP备00000000号",
  "wztj": "",
  "cdnpublic": "https://cdn.example.com/",
  "yan": "true"
}
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "updated": true } }
```

#### 3. GET /settings/about

- **用途**：读取关于页 HTML 内容；未配置时返回内置默认内容。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `content` | string | 关于页 HTML（未配置时为内置默认内容） |

- **错误码**：401、429。
- **请求示例**：

```http
GET /api/v1/settings/about
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "content": "<h3>关于本站</h3>\r\n<p>欢迎访问本站…</p>" }
}
```

#### 4. PUT /settings/about

- **用途**：更新关于页 HTML 内容（按原始提交值落库）。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `content` | body | string | 是 | 不得缺失/为空串；无长度上限 | 关于页 HTML |

- **出参 `data`**：`{ "updated": true }`。
- **错误码**：400（缺少 `content` 或为空串、JSON 非法）、401、429、500。
- **请求示例**：

```http
PUT /api/v1/settings/about
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "content": "<p>这是自定义关于页</p>" }
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "updated": true } }
```

#### 5. POST /settings/about/reset

- **用途**：将关于页内容恢复为内置默认内容。
- **鉴权**：需要。
- **入参**：无请求体。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `content` | string | 恢复后的内置默认 HTML 内容 |

- **错误码**：401、429、500。
- **请求示例**：

```http
POST /api/v1/settings/about/reset
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "content": "<h3>关于本站</h3>…" }
}
```

#### 6. PUT /settings/account

- **用途**：修改后台管理员用户名和/或密码。旧密码校验最先执行；业务分支与原后台 `user.php` 一致。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `old_password` | body | string | 是 | — | 当前管理员密码明文 |
| `username` | body | string | 否 | 服务端 trim；空串不改用户名 | 新用户名 |
| `new_password` | body | string | 否 | 空串不改密码 | 新密码 |
| `confirm_password` | body | string | 否 | 须与 `new_password` 一致 | 新密码确认值 |

- **出参 `data`（不回显新密码明文）**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `changed_user` | bool | 是否修改了用户名 |
| `changed_pwd` | bool | 是否修改了密码 |
| `username` | string | 修改后的用户名（未改名时为空串） |

- **错误码（业务错误均为 400）**：

| 触发条件 | message 要点 |
|---|---|
| `old_password` 与当前密码不符 | `config: 当前密码错误` |
| `new_password` 与 `confirm_password` 不一致 | `config: 两次新密码不一致` |
| `username` trim 后为空且 `new_password` 为空（无任何更改） | `config: 未做出更改` |

- 密码存储沿用 `md5("lylme" + 明文)`；本端点仅影响后台登录，不影响 API 密钥。
- **请求示例（仅改密）**：

```http
PUT /api/v1/settings/account
Authorization: Bearer sk-xxxx
Content-Type: application/json

{
  "username": "",
  "old_password": "oldpass1",
  "new_password": "newpass2",
  "confirm_password": "newpass2"
}
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "changed_user": false, "changed_pwd": true, "username": "" }
}
```

#### 7. GET /settings/system

- **用途**：读取系统安全设置。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `admin_path` | string | 后台目录名；未配置时回退应用配置（`.env` 的 `ADMIN_PATH`，再缺省为 `admin`） |
| `debug` | string | 调试模式，仅 `"0"`/`"1"`；未配置或非 `"1"` 一律返回 `"0"` |

- **错误码**：401、429。
- **请求示例**：

```http
GET /api/v1/settings/system
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "admin_path": "admin", "debug": "0" }
}
```

#### 8. PUT /settings/system

- **用途**：更新后台目录与调试模式；两项校验全部通过才落库，任一项非法不写任何配置。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `admin_path` | body | string | 是 | trim 后非空且匹配 `^[A-Za-z0-9_-]+$`（仅字母、数字、下划线、中划线） | 后台目录名 |
| `debug` | body | string | 是 | 枚举 `"0"`、`"1"` | 调试模式 |

- **出参 `data`**：`{ "updated": true }`。
- **错误码**：
  - 400：`admin_path` 缺失/为空/含非法字符（如 `a/b`、`a b`、`../x`、中文），或 `debug` 非 `"0"`/`"1"`；
  - 401、429、500。
- `admin_path` 落库后于**下次启动**注册路由时生效，需通过新路径访问后台。
- **请求示例**：

```http
PUT /api/v1/settings/system
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "admin_path": "panel-1", "debug": "1" }
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "updated": true } }
```

#### 9. GET /wxplus

- **用途**：读取微信推送配置。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 落库键 | 含义 |
|---|---|---|---|
| `key` | string | `wxplus` | 微信推送密钥；空串未开启 |
| `time` | string | `wxplustime` | 每日推送时间（如 `20:00`） |

- **错误码**：401、429。
- **请求示例**：

```http
GET /api/v1/wxplus
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "key": "", "time": "20:00" }
}
```

#### 10. PUT /wxplus

- **用途**：更新微信推送密钥与每日推送时间；空 key 表示关闭推送，按原始提交值落库。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 含义 |
|---|---|---|---|---|
| `key` | body | string | 否 | 微信推送密钥 |
| `time` | body | string | 否 | 每日推送时间 |

- **出参 `data`**：`{ "updated": true }`。
- **错误码**：400（JSON 格式错误）、401、429、500。
- **请求示例**：

```http
PUT /api/v1/wxplus
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "key": "WXSECRET", "time": "08:30" }
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "updated": true } }
```

#### 11. GET /license

- **用途**：读取网站授权配置。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 落库键 | 含义 |
|---|---|---|---|
| `code` | string | `c` | 程序授权码；未配置为空串 |
| `hostmode` | string | `hostmode` | 域名获取方式：`"1"` HTTP_HOST / `"2"` SERVER_NAME；缺省或其他值归一为 `"2"` |

- **错误码**：401、429。
- **请求示例**：

```http
GET /api/v1/license
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "code": "", "hostmode": "2" }
}
```

#### 12. PUT /license

- **用途**：更新网站授权码与域名获取方式，按原始提交值落库。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `code` | body | string | 否 | — | 程序授权码 |
| `hostmode` | body | string | 是 | 枚举 `"1"`、`"2"` | 域名获取方式 |

- **出参 `data`**：`{ "updated": true }`。
- **错误码**：400（`hostmode` 缺失或非 `"1"`/`"2"`）、401、429、500。
- **请求示例**：

```http
PUT /api/v1/license
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "code": "LIC-123", "hostmode": "1" }
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "updated": true } }
```

#### 13. GET /apikey

- **用途**：查询当前 API/MCP 密钥信息；**任何情况下都不回读完整明文**。
- **鉴权**：需要 Bearer 当前密钥（能通过本端点鉴权即证明密钥有效）。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `exists` | bool | 当前是否存在有效密钥 |
| `created_at` | string | 密钥创建时间，RFC3339；无法解析时为空串 |
| `masked` | string | 密钥掩码，形如 `sk-1a2b…`（保留 `sk-` 与其后 4 个字符，其余以 `…` 代替）；密钥不存在时为空串 |

- **错误码**：401、429。
- **请求示例**：

```http
GET /api/v1/apikey
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "exists": true,
    "created_at": "2026-10-04T10:00:00+08:00",
    "masked": "sk-1a2b…"
  }
}
```

#### 14. POST /apikey/regenerate

- **用途**：重新生成 API/MCP 密钥。
- **鉴权**：Bearer **当前（即将被作废）**密钥；中间件在本请求内仍以旧密钥鉴权，service 处理时完成轮换。
- **入参**：无请求体。
- **出参 `data`（本端点是唯一会返回完整明文密钥的时机）**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `key` | string | 新密钥完整明文（`sk-` + 48 个十六进制字符）；仅此一次展示，请立即妥善保存 |
| `created_at` | string | 新密钥创建时间，RFC3339 |

> ## ⚠️ 旧密钥立即失效
>
> 重新生成成功的瞬间，旧密钥立即、永久失效，无法恢复。所有使用旧密钥的 REST / MCP 客户端、脚本与 CI 配置必须立即更换新密钥，否则随后的请求一律返回 **401**。

- **错误码**：401（旧密钥在调用前已失效）、429、500。
- **轮换后验证时序**：① 用新密钥请求任意端点 → 200；② 用旧密钥请求任意端点 → 401。
- **请求示例**：

```http
POST /api/v1/apikey/regenerate
Authorization: Bearer sk-旧密钥
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "key": "sk-9f8e7d6c5b4a000000000000000000000000000000000000",
    "created_at": "2026-10-04T11:30:00+08:00"
  }
}
```

---

### 3.8 收录申请 Apply

#### 1. GET /apply

- **用途**：分页返回收录申请，按 `apply_time` 倒序，支持按审核状态过滤。分页切片在 handler 层完成。
- **鉴权**：需要。
- **入参（Query）**：

| 名 | 位置 | 类型 | 必填 | 默认 | 约束 | 含义 |
|---|---|---|---|---|---|---|
| `status` | query | int | 否 | 缺省=全部 | 枚举 `0`、`1`、`2` | `0` 待审核 / `1` 已通过 / `2` 已拒绝 |
| `page` | query | int | 否 | 1 | ≥ 1 | 页码 |
| `page_size` | query | int | 否 | 10 | 1–100 | 每页条数 |

> 注意：与 `GET /links` 不同，本端点参数经绑定校验，`page`/`page_size`/`status` 非法（含非数值、越界、错误枚举）返回 **400**，不做静默回退。

- **出参 `data`**：分页结构（见 [2.4](#24-分页结构)），`list` 元素为 [ApplyItem](#28-通用对象结构)。
- **错误码**：400（参数非法）、401、429、500。
- **请求示例**：

```http
GET /api/v1/apply?status=0&page=1&page_size=10
Authorization: Bearer sk-xxxx
```

- **响应示例**：

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

#### 2. GET /apply/settings

- **用途**：读取收录开关与公告。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `open` | int | 收录开关：`0` 开启（需审核）/ `1` 开启（免审核）/ `2` 关闭；未配置默认 `0` |
| `gg` | string | 收录公告（配置键 `apply_gg`） |

- **错误码**：401、429。
- **请求示例**：

```http
GET /api/v1/apply/settings
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "open": 0, "gg": "" } }
```

#### 3. PUT /apply/settings

- **用途**：更新收录开关与公告。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `open` | body | int | 是 | 枚举 `0`、`1`、`2` | 收录开关 |
| `gg` | body | string | 否 | 最长 255 | 收录公告 |

- **出参 `data`**：保存后的收录设置（结构同 GET）。
- **错误码**：400（`open` 非 0/1/2、`gg` 超长、JSON 非法）、401、429、500。
- **请求示例**：

```http
PUT /api/v1/apply/settings
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "open": 1, "gg": "欢迎提交收录" }
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "open": 1, "gg": "欢迎提交收录" }
}
```

#### 4. PUT /apply/:id

- **用途**：修改指定收录申请的名称、URL、分组、图标。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 申请 ID |
| `name` | body | string | 是 | 最长 60；不可为空 | 申请名称 |
| `url` | body | string | 否 | 最长 255 | 申请 URL |
| `group_id` | body | int | 否 | 无枚举校验 | 所属分组 ID |
| `icon` | body | string | 否 | 无长度上限 | 图标 |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 申请 ID |
| `updated` | bool | 固定为 `true` |
| `row` | ApplyItem | 修改后的申请元素 |

- **错误码**：400（id 非正整数；name 为空/超长；url 超长）、401、404（申请不存在）、429、500。
- **请求示例**：

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

- **响应示例**：

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

#### 5. POST /apply/:id/review

- **用途**：审核单条收录申请。**通过（status=1）**：由 service 写入链接表（名称/URL 去除 HTML 标签，`link_order=MAX(id)+1`，启用），申请置 1；**拒绝（status=2）**：仅将申请置 2。
- **鉴权**：需要。
- **入参**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `id` | path | int | 是 | 正整数 | 申请 ID |
| `status` | body | int | 是 | 枚举 `1`、`2` | `1` 通过 / `2` 拒绝 |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | int | 申请 ID |
| `status` | int | 审核后申请状态 |

- **错误码**：400（status 非 1/2、id 非正整数）、401、404（申请不存在）、429、500。
- **幂等**：仅对 `status=0`（待审核）记录生效；对已处理记录重复调用返回 200，不重复写链接、不改变状态。
- 审核通过时链接表恰好新增 1 条链接，可经 `GET /links/all` 查询；拒绝不新增链接。
- **请求示例（通过）**：

```http
POST /api/v1/apply/1/review
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "status": 1 }
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "id": 1, "status": 1 } }
```

- 拒绝示例：`{ "status": 2 }` → `{ "id": 2, "status": 2 }`；非法枚举 `{ "status": 3 }` → 400。

#### 6. DELETE /apply

- **用途**：按 ID 列表批量删除收录申请。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `ids` | body | int[] | 是 | 至少 1 个；每个为正整数；无数量上限 | 待删除申请 ID |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `deleted` | int | 本次提交删除的 ID 数量 |

- **错误码**：400（ids 缺失/空数组、含非正整数）、401、429、500。
- **请求示例**：

```http
DELETE /api/v1/apply
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "ids": [1, 2] }
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "deleted": 2 } }
```

---

### 3.9 主题 Themes

#### 1. GET /themes

- **用途**：返回全部已发现主题（`template/*` 下含 `theme.ini` 的目录）及当前启用主题目录名。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `current` | string | 当前主题目录名；配置缺失或指向不存在主题时回退 `default` |
| `themes` | ThemeInfo[] | 主题列表（结构见 [ThemeInfo](#28-通用对象结构)） |

- **错误码**：401、429。
- **请求示例**：

```http
GET /api/v1/themes
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "current": "default",
    "themes": [
      { "dir": "5iux", "name": "五零优享", "author": "lylme", "version": "1.0", "description": "简洁主题" },
      { "dir": "default", "name": "default", "author": "六零二开", "version": "2.0", "description": "默认主题" }
    ]
  }
}
```

#### 2. PUT /themes/current

- **用途**：切换当前主题（写配置键 `template`），前台首页随即使用新主题。
- **鉴权**：需要。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `name` | body | string | 是 | trim 后不能为空 | 目标主题目录名（如 `default`） |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `current` | string | 切换后的当前主题目录名 |

- **错误码**：400（`name` 缺失/空白、JSON 非法）、401、404（目标主题不存在，`message` 为 `theme: 主题不存在`）、429、500。
- **请求示例**：

```http
PUT /api/v1/themes/current
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "name": "default" }
```

- **响应示例**：

```json
{ "code": 200, "message": "ok", "data": { "current": "default" } }
```

#### 3. GET /themes/config

- **用途**：返回当前主题的配置表单字段定义（解析自主题 `config.php`）与已保存的自定义配置（配置键 `theme_config_<tpl>`）。
- **鉴权**：需要。
- **入参**：无（始终针对当前主题）。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `theme` | string | 配置所属主题目录名 |
| `fields` | ThemeField[] | 字段定义列表（结构见 [ThemeField](#28-通用对象结构)） |
| `values` | object | 已保存自定义配置键值 map；从未保存过时为 `{}`（不为 null） |

- **错误码**：401、429。
- **请求示例**：

```http
GET /api/v1/themes/config
Authorization: Bearer sk-xxxx
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "theme": "default",
    "fields": [
      {
        "type": "textarea",
        "name": "home_title",
        "title": "首页提示",
        "description": "显示在首页的提示内容，<code>支持HTML代码</code>",
        "verify": "",
        "placeholder": "",
        "value": "",
        "enum": []
      },
      {
        "type": "select",
        "name": "lytoday",
        "title": "今日热榜",
        "description": "LyToday-JS插件显示位置",
        "verify": "",
        "placeholder": "",
        "value": "0",
        "enum": [
          { "value": "0", "label": "关闭" },
          { "value": "1", "label": "搜索栏下方" },
          { "value": "2", "label": "底部" }
        ]
      }
    ],
    "values": {}
  }
}
```

#### 4. PUT /themes/config

- **用途**：保存主题自定义配置，JSON 序列化后写配置键 `theme_config_<tpl>`；未提交的 `switch` 字段自动补 `0`，未提交（或提交空串）的 `checkbox` 字段自动补空数组。
- **鉴权**：需要。
- **入参（JSON Body）**：请求体必须为 JSON 对象，支持**三种等价形式**：

| 名 | 位置 | 类型 | 必填 | 含义 |
|---|---|---|---|---|
| `theme` | body | string | 否 | 主题目录名；为空或主题不存在时回退当前主题 |
| `values` | body | object | 否 | 配置键值对象（推荐形式） |
| `data` | body | object | 否 | `values` 的别名，二者择一 |

三种等价形式：

```json
{ "theme": "default", "values": { "lytoday": "1" } }
```

```json
{ "theme": "default", "data": { "lytoday": "1" } }
```

```json
{ "theme": "", "lytoday": "1", "gonganbei": "" }
```

> 第三种为平铺形式：除 `theme` 外的键全部视为配置键。
> **全量替换语义**：未出现在本次请求中的既有配置键会被覆盖移除（字段定义层面的 switch/checkbox 默认值补齐除外）。

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `theme` | string | 实际生效的主题目录名 |
| `values` | object | 保存后重新读回的配置值（即随后 GET /themes/config 的 `values`） |

- **错误码**：400（请求体非 JSON 对象；`values`/`data` 存在但不是对象）、401、429、500。
- **请求示例**：

```http
PUT /api/v1/themes/config
Authorization: Bearer sk-xxxx
Content-Type: application/json

{ "theme": "default", "values": { "lytoday": "1" } }
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "theme": "default",
    "values": { "lytoday": "1" }
  }
}
```

> `values` 仅反映 `theme_config_<tpl>` 中本次提交并保存的内容（switch/checkbox 缺省补齐除外），不与字段默认值合并；字段默认值见 GET 返回的 `fields[].value`。

---

### 3.10 仪表盘与文件清理 Dashboard / Files

#### 1. GET /dashboard

- **用途**：返回后台仪表盘汇总：链接/分组/待审申请计数与版本信息。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `links` | int64 | 链接总数（`lylme_links` 行数） |
| `groups` | int64 | 分组总数（`lylme_groups` 行数） |
| `apply_rows` | int64 | 待审核收录申请数（`apply_status=0`） |
| `app_version` | string | 后端程序版本 |
| `db_version` | string | 数据库版本（配置键 `version`，未设置为空串） |
| `build` | string | 建站时间（配置键 `build`，未设置为空串） |

- **错误码**：401、429（该端点处理器无错误返回路径，不产生 400/404/500）。
- **请求示例**：

```bash
curl -H "Authorization: Bearer sk-xxxx" \
  https://example.com/api/v1/dashboard
```

- **响应示例**：

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

#### 2. GET /files/cleanup

- **用途**：扫描 `files/download` 与 `files/upload` 目录，与数据库图标引用（`lylme_groups.group_icon`、`lylme_links.icon`）逐一比对：文件名（大小写不敏感）出现在任一引用中即判定为已使用；仅未使用文件进入列表。
- **鉴权**：需要。
- **入参**：无。
- **出参 `data`（扫描报告）**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `total` | int | 扫描到的文件总数（含已使用，不含子目录） |
| `used` | int | 被图标引用的文件数 |
| `unused` | int | 未使用文件数 |
| `total_size` | int64 | 未使用文件总大小（字节） |
| `total_size_str` | string | 未使用文件总大小（人类可读，如 `1.01 KB`） |
| `dirs` | array | 分目录报告，固定顺序 `download`、`upload` |

`dirs[]`：

| 字段 | 类型 | 含义 |
|---|---|---|
| `name` | string | `download` 或 `upload` |
| `unused` | int | 该目录未使用文件数 |
| `files` | array | 未使用文件列表（无则为 `[]`） |

`dirs[].files[]`：

| 字段 | 类型 | 含义 |
|---|---|---|
| `rel` | string | 相对 `files` 的路径，如 `download/a.png`；POST 删除时使用此标识 |
| `name` | string | 文件名 |
| `dir` | string | 所在目录：`download` / `upload` |
| `url` | string | 访问 URL：`/files/<rel>` |
| `ext` | string | 小写扩展名（不含点），无则空串 |
| `size` | int64 | 文件大小（字节） |
| `size_str` | string | 人类可读大小（如 `12.34 KB`） |
| `is_img` | bool | 是否图片（png/jpg/jpeg/gif/ico/webp） |

> 目录不存在时该目录按空统计，不构成错误。

- **错误码**：401、429、500（数据库引用查询失败）。
- **请求示例**：

```bash
curl -H "Authorization: Bearer sk-xxxx" \
  https://example.com/api/v1/files/cleanup
```

- **响应示例**：

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
          { "rel": "download/unused1.png", "name": "unused1.png", "dir": "download", "url": "/files/download/unused1.png", "ext": "png", "size": 1, "size_str": "1 B", "is_img": true },
          { "rel": "download/unused2.txt", "name": "unused2.txt", "dir": "download", "url": "/files/download/unused2.txt", "ext": "txt", "size": 5, "size_str": "5 B", "is_img": false }
        ]
      },
      {
        "name": "upload",
        "unused": 1,
        "files": [
          { "rel": "upload/up.jpg", "name": "up.jpg", "dir": "upload", "url": "/files/upload/up.jpg", "ext": "jpg", "size": 1024, "size_str": "1.00 KB", "is_img": true }
        ]
      }
    ]
  }
}
```

#### 3. POST /files/cleanup

- **用途**：按 `rels` 列表逐条校验并删除未使用文件；单条失败不阻断其余条目。
- **鉴权**：需要。
- **合法 `rel` 规则**：
  - 正则 `^(download|upload)/[^/]+$`：仅允许 `download`/`upload` 单层目录下的单个文件；
  - 不得包含目录穿越片段 `..`（含反斜杠会先归一为 `/` 再校验）；
  - 目标必须为已存在的普通文件。
- **入参（JSON Body）**：

| 名 | 位置 | 类型 | 必填 | 约束 | 含义 |
|---|---|---|---|---|---|
| `rels` | body | string[] | 是 | 至少 1 项 | 待删除文件 rel 列表（取自 GET 扫描结果） |

- **出参 `data`**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `deleted` | int | 成功删除数 |
| `failed` | int | 失败数：非法路径、目录穿越、文件不存在或删除失败均计入 |

- **错误码**：
  - 400：请求体非合法 JSON；缺少 `rels`；`rels` 为空数组（`message` 为 `请求体无效：rels 为必填项且至少包含 1 个路径`）；
  - 401、429。
- 个别元素非法**不**返回 4xx，正常返回 200 并计入 `failed`。
- **请求示例**：

```bash
curl -X POST \
  -H "Authorization: Bearer sk-xxxx" \
  -H "Content-Type: application/json" \
  -d '{"rels":["download/unused1.png","../x"]}' \
  https://example.com/api/v1/files/cleanup
```

- **响应示例**：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "deleted": 1, "failed": 1 }
}
```

---

## 4. MCP 服务

### 4.1 Endpoint 与传输

- **Endpoint**：`POST <站点根>/mcp`（同时挂载 `/mcp` 与 `/mcp/`；支持 GET/POST/DELETE 方法）。
- **传输**：MCP **Streamable HTTP，无状态（stateless）**：每个请求独立处理，不依赖会话粘滞，服务端不返回 `Mcp-Session-Id`，无服务端主动通知。
- 请求体/响应遵循 JSON-RPC 2.0；响应可能为普通 JSON 或 `text/event-stream`（SSE），客户端均应支持。
- 标准握手流程：`initialize` → `notifications/initialized` → `tools/list` / `tools/call`。

### 4.2 鉴权

- 与 REST 相同，每个请求必须携带：

```http
Authorization: Bearer sk-xxxx
```

- 无凭证、格式错误、密钥无效/已失效时，在进入 JSON-RPC 层之前直接返回 HTTP **401**；超限返回 **429**。
- REST 与 MCP 共用同一密钥与同一 IP 限流桶（120 次/分钟）；MCP 请求不计入前台 PV。

### 4.3 连接配置示例

后台「接口与 MCP」页面可直接复制如下配置（`url` 指向 `/mcp`，`headers.Authorization` 已内嵌当前密钥）：

```json
{
  "mcpServers": {
    "lylme": {
      "url": "https://example.com/mcp",
      "headers": {
        "Authorization": "Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
      }
    }
  }
}
```

### 4.4 工具结果约定

- `tools/list`：返回 56 个工具；每个工具声明 `name`、`description`（用途）与 `inputSchema`（参数 JSON Schema：类型、必填、枚举/长度约束）。
- `tools/call`：
  - **成功**：返回文本内容（`type: "text"`），文本为 **JSON 字符串，结构与对应 REST 端点的 `data` 完全一致**；
  - **业务错误**：返回 `isError: true` 的工具结果，文本为错误说明（参数错误、记录不存在等哨兵错误附带明确信息；未知内部错误统一为 `操作失败，请稍后再试`，不泄露堆栈）；
  - 参数解析在工具处理器内完成，缺失/类型错误同样返回 `isError: true`。

**握手与调用示例（curl）**

```bash
# initialize
curl -X POST https://example.com/mcp \
  -H "Authorization: Bearer sk-xxxx" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"demo","version":"1.0"}}}'

# tools/call
curl -X POST https://example.com/mcp \
  -H "Authorization: Bearer sk-xxxx" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"group_list","arguments":{}}}'
```

### 4.5 MCP 工具与 REST 端点对应总表（56）

> 参数摘要中，带 `*` 为必填；未标注约束的为可选。参数类型与 MCP `inputSchema` 一致。

| # | 工具名 | 用途 | 对应方法 + 路径 | 参数摘要 |
|---|---|---|---|---|
| 1 | `group_list` | 查询全部分组（group_order 升序），返回列表、总数与各组链接计数 | GET /groups | 无 |
| 2 | `group_create` | 新增分组；纯 URL 图标自动包 img，排序追加末尾 | POST /groups | `name*`(≤30)、icon(≤1000)、pwd |
| 3 | `group_update` | 修改分组名称、图标或关联加密组 | PUT /groups/:id | `id*`、`name*`(≤30)、icon、pwd |
| 4 | `group_delete` | 删除分组（同时删除组内全部链接），不可恢复 | DELETE /groups/:id | `id*` |
| 5 | `group_set_status` | 启用/停用分组，停用后前台不展示 | POST /groups/:id/status | `id*`、`status*`(0/1) |
| 6 | `group_sort` | 按 ids 顺序重排分组 | POST /groups/sort | `ids*`(number[],1–200) |
| 7 | `link_list` | 分页查询链接，可按分组过滤或关键词检索（关键词非空时忽略分组） | GET /links | page、page_size、group_id、keyword |
| 8 | `link_all` | 返回全站链接简表（id/url/name，id 升序） | GET /links/all | 无 |
| 9 | `link_get` | 按 ID 查询单条链接完整信息 | GET /links/:id | `id*` |
| 10 | `link_create` | 新增链接；URL 去末尾斜杠查重，重复时报错并给出已存在 ID；color 非空名称以指定颜色展示 | POST /links | `name*`(≤255)、`url*`(≤255)、color(≤32)、icon(≤2000)、group_id、link_desc(≤255)、link_keywords(≤512)、pwd |
| 11 | `link_update` | 修改链接全部字段 | PUT /links/:id | `id*`、`name*`、`url*`、color、icon、group_id、link_desc、link_keywords、pwd |
| 12 | `link_delete` | 按 ID 列表批量删除链接，不可恢复 | DELETE /links | `ids*`(number[],1–500) |
| 13 | `link_set_status` | 批量启用/停用链接 | POST /links/status | `ids*`、`status*`(0/1) |
| 14 | `link_move_group` | 批量移动链接到目标分组 | POST /links/group | `ids*`、`group_id*`(≥1) |
| 15 | `link_set_pwd` | 批量设置/取消链接访问密码组 | POST /links/pwd | `ids*`、pwd |
| 16 | `link_sort` | 按字符串 ID 顺序重排链接 | POST /links/sort | `ordered*`(string[],1–500) |
| 17 | `link_fetch_head` | 抓取目标站标题/图标等 head 信息，含 SSRF 防护 | GET /links/fetch | `url*`(http/https) |
| 18 | `tag_list` | 查询全部导航菜单（sort 升序） | GET /tags | 无 |
| 19 | `tag_create` | 新增导航菜单，sort 为 0 默认 10 | POST /tags | `name*`(≤30)、`link*`(≤255)、sort、target(0/1) |
| 20 | `tag_update` | 修改导航菜单全部字段 | PUT /tags/:id | `id*`、`name*`、`link*`、sort、target |
| 21 | `tag_delete` | 删除导航菜单 | DELETE /tags/:id | `id*` |
| 22 | `sou_list` | 查询全部搜索引擎（sou_order 升序） | GET /sou | 无 |
| 23 | `sou_create` | 新增搜索引擎，排序自动追加末尾 | POST /sou | `alias*`(≤30)、`name*`(≤30)、hint(≤60)、color(≤20)、link(≤255)、waplink(≤255)、icon(≤2000)、st(bool)、order |
| 24 | `sou_update` | 修改搜索引擎全部字段，order 生效 | PUT /sou/:id | `id*`、`alias*`、`name*`、hint、color、link、waplink、icon、st、order |
| 25 | `sou_delete` | 删除搜索引擎 | DELETE /sou/:id | `id*` |
| 26 | `pwd_list` | 查询全部加密组 | GET /pwd | 无 |
| 27 | `pwd_create` | 新增加密组 | POST /pwd | `name*`(≤30)、`key*`(≤60)、ps(≤60) |
| 28 | `pwd_update` | 修改加密组全部字段 | PUT /pwd/:id | `id*`、`name*`、`key*`、ps |
| 29 | `pwd_delete` | 删除加密组 | DELETE /pwd/:id | `id*` |
| 30 | `settings_get` | 读取全部站点配置（admin_pwd、api_key 被剔除） | GET /settings | 无 |
| 31 | `settings_update` | 批量更新网站基本设置；yan 仅 "true" 为真 | PUT /settings | title、logo、background、wap_background、keywords、description、mode、snapshot、copyright、icp、wztj、cdnpublic、yan |
| 32 | `settings_about_get` | 读取关于页内容，未配置返回默认 | GET /settings/about | 无 |
| 33 | `settings_about_update` | 更新关于页 HTML | PUT /settings/about | `content*` |
| 34 | `settings_about_reset` | 恢复关于页默认内容并返回 | POST /settings/about/reset | 无 |
| 35 | `settings_account_update` | 修改管理员用户名/密码，不回显新密码 | PUT /settings/account | `old_password*`、username、new_password、confirm_password |
| 36 | `settings_system_get` | 读取后台目录与调试模式 | GET /settings/system | 无 |
| 37 | `settings_system_update` | 更新后台目录与调试模式 | PUT /settings/system | `admin_path*`、`debug*`(0/1) |
| 38 | `wxplus_get` | 读取微信推送密钥与每日推送时间 | GET /wxplus | 无 |
| 39 | `wxplus_update` | 更新微信推送配置，空 key 关闭推送 | PUT /wxplus | key、time |
| 40 | `license_get` | 读取授权码与域名获取方式 | GET /license | 无 |
| 41 | `license_update` | 更新授权码与域名获取方式 | PUT /license | code、`hostmode*`(1/2) |
| 42 | `apikey_get` | 查询密钥存在性、创建时间与掩码，不回读明文 | GET /apikey | 无 |
| 43 | `apikey_regenerate` | 重新生成密钥，旧 key 立即失效；唯一返回完整明文的时机 | POST /apikey/regenerate | 无 |
| 44 | `apply_list` | 分页查询收录申请（apply_time 倒序），可按状态过滤 | GET /apply | status(0/1/2)、page、page_size |
| 45 | `apply_settings_get` | 读取收录开关与公告 | GET /apply/settings | 无 |
| 46 | `apply_settings_update` | 更新收录开关与公告 | PUT /apply/settings | `open*`(0/1/2)、gg(≤255) |
| 47 | `apply_update` | 修改收录申请名称、URL、分组、图标 | PUT /apply/:id | `id*`、`name*`(≤60)、url(≤255)、group_id、icon |
| 48 | `apply_review` | 审核申请：1 通过（写链接表）/ 2 拒绝；对待审核记录生效、重复审核幂等 | POST /apply/:id/review | `id*`、`status*`(1/2) |
| 49 | `apply_delete` | 批量删除收录申请 | DELETE /apply | `ids*`(number[]) |
| 50 | `theme_list` | 查询全部可用主题与当前主题目录名 | GET /themes | 无 |
| 51 | `theme_switch` | 按目录名切换当前主题 | PUT /themes/current | `name*` |
| 52 | `theme_config_get` | 读取当前主题字段定义与已保存配置值 | GET /themes/config | 无 |
| 53 | `theme_config_save` | 保存主题自定义配置（全量替换；switch/checkbox 缺省补齐）；MCP 仅支持 values 形式 | PUT /themes/config | theme、values(object) |
| 54 | `dashboard_summary` | 查询仪表盘计数与版本信息 | GET /dashboard | 无 |
| 55 | `files_cleanup_scan` | 扫描未使用文件并返回报告 | GET /files/cleanup | 无 |
| 56 | `files_cleanup_delete` | 删除未使用文件，返回成功/失败计数 | POST /files/cleanup | `rels*`(string[]) |

---

## 5. 变更记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1 | 2026-10-04 | 首版终稿：57 个 REST 已注册路由（含 `GET /ping` 自检）、56 个业务端点、56 个 MCP 工具；文档与路由注册、请求/响应结构体 JSON tag、MCP 工具定义逐条校对一致。 |
