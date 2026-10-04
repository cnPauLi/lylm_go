# Settings 域 REST API

六零导航页 Go 版第二期 · Task 13。覆盖网站设置、关于页、管理员账号、系统安全、微信推送、授权码、API 密钥管理。

## 通用约定

- **Base URL**：`/api/v1`
- **鉴权**：除特别说明外，本文件所有端点均需在请求头携带当前有效 API 密钥：

  ```http
  Authorization: Bearer sk-xxxxxxxx...
  ```

- **请求体**：写操作（PUT/POST 带 body）均使用 `application/json; charset=utf-8`。
- **统一响应信封**：

  ```json
  { "code": 200, "message": "ok", "data": {} }
  ```

- **通用错误码**：

  | HTTP | code | 含义 |
  |------|------|------|
  | 400 | 400 | 请求体不是合法 JSON，或参数校验/业务规则校验失败（message 为具体原因） |
  | 401 | 401 | 缺少/格式错误的 Authorization 头，或密钥无效、已失效 |
  | 429 | 429 | 超出单 IP 限流阈值（默认 120 次/分钟） |
  | 500 | 500 | 服务端内部错误（响应 message 固定为 `internal error`） |

---

## 1. GET /settings —— 读取全部配置项

读取 `lylme_config` 表中的全部键值对。

- **鉴权**：Bearer 当前密钥。
- **入参**：无。
- **出参 `data`**：`object`，键为配置名、值均为字符串。
- **敏感键屏蔽（显式黑名单）**：以下键**永远不会**出现在响应中：

  | 黑名单键 | 说明 |
  |----------|------|
  | `admin_pwd` | 管理员密码哈希 |
  | `api_key` | API/MCP 接口密钥明文 |

  其余配置项（含 `wxplus` 推送密钥、`c` 授权码、`admin_user` 管理员用户名等）照常返回。

**请求示例**

```http
GET /api/v1/settings HTTP/1.1
Authorization: Bearer sk-1a2b3c4d...
```

**成功响应（200）**

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

**错误**：401（密钥缺失/无效）、429、500。

---

## 2. PUT /settings —— 批量更新网站设置

全字段批量更新 12 个网站设置项，同时写入 `yan`（仅当提交值**恰好为字符串 `"true"`** 时落 `"true"`，其余任何取值——含 `"1"`、`"yes"`、空串——一律落 `"false"`）与固定值 `tq="false"`。

- **鉴权**：Bearer 当前密钥。
- **入参（JSON object，全部字段为字符串，空串为合法值，表示清空该项）**：

  | 字段 | 落库配置键 | 说明 |
  |------|------------|------|
  | `title` | `title` | 网站名称 |
  | `logo` | `logo` | 网站 LOGO 地址 |
  | `background` | `background` | PC 背景图片 |
  | `wap_background` | `wap_background` | 手机背景图片 |
  | `keywords` | `keywords` | 网站关键词 |
  | `description` | `description` | 网站描述 |
  | `mode` | `mode` | 网站运行模式（原版取值 `"1"`/`"2"`，服务端不强制枚举） |
  | `snapshot` | `snapshot` | 详情页快照 API 地址 |
  | `copyright` | `copyright` | 底部版权 |
  | `icp` | `icp` | ICP 备案号 |
  | `wztj` | `wztj` | 自定义 footer |
  | `cdnpublic` | `cdnpublic` | 静态资源 CDN 地址 |
  | `yan` | `yan` | 随机一言开关，仅 `"true"` 为真 |

- **出参 `data`**：

  ```json
  { "updated": true }
  ```

**请求示例**

```http
PUT /api/v1/settings HTTP/1.1
Authorization: Bearer sk-1a2b3c4d...
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

**成功响应（200）**

```json
{ "code": 200, "message": "ok", "data": { "updated": true } }
```

**错误**：400（JSON 格式错误）、401、429、500。更新后可用 `GET /settings` 读回校验。

---

## 3. GET /settings/about —— 读取关于页内容

- **鉴权**：Bearer 当前密钥。
- **入参**：无。
- **出参 `data`**：

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | `content` | string | 关于页 HTML 内容；未配置时返回内置默认内容 |

**成功响应（200）**

```json
{
  "code": 200,
  "message": "ok",
  "data": { "content": "<h3>关于本站</h3>\r\n<p>欢迎访问本站…</p>" }
}
```

**错误**：401、429、500。

---

## 4. PUT /settings/about —— 更新关于页内容

- **鉴权**：Bearer 当前密钥。
- **入参（JSON object）**：

  | 字段 | 类型 | 约束 | 说明 |
  |------|------|------|------|
  | `content` | string | **必填**（binding required，不得为空串） | 关于页 HTML，按原始提交值落库 |

- **出参 `data`**：`{ "updated": true }`

**请求示例**

```http
PUT /api/v1/settings/about HTTP/1.1
Authorization: Bearer sk-1a2b3c4d...
Content-Type: application/json

{ "content": "<p>这是自定义关于页</p>" }
```

**错误**：

| HTTP | 触发条件 |
|------|----------|
| 400 | 缺少 `content` 字段或为空串；JSON 格式错误 |
| 401 / 429 / 500 | 见通用约定 |

---

## 5. POST /settings/about/reset —— 恢复关于页默认内容

- **鉴权**：Bearer 当前密钥。
- **入参**：无请求体。
- **出参 `data`**：

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | `content` | string | 恢复后的内置默认 HTML 内容 |

**成功响应（200）**

```json
{
  "code": 200,
  "message": "ok",
  "data": { "content": "<h3>关于本站</h3>…" }
}
```

**错误**：401、429、500。

---

## 6. PUT /settings/account —— 修改管理员账号

修改后台管理员用户名/密码。业务分支与原后台 `user.php` 完全一致：旧密码校验最先执行，之后根据用户名与新密码的组合决定改名、改密或两者都改。

- **鉴权**：Bearer 当前密钥。
- **入参（JSON object）**：

  | 字段 | 类型 | 约束 | 说明 |
  |------|------|------|------|
  | `username` | string | 可空；服务端 trim 首尾空白 | 新用户名；空串表示不改用户名 |
  | `old_password` | string | **必填**（binding required） | 当前密码明文 |
  | `new_password` | string | 可空 | 新密码；空串表示不改密码 |
  | `confirm_password` | string | 可空 | 新密码确认值 |

- **出参 `data`（不回显新密码明文）**：

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | `changed_user` | bool | 是否修改了用户名 |
  | `changed_pwd` | bool | 是否修改了密码 |
  | `username` | string | 修改后的用户名（未改名时为空串） |

**业务规则与错误（均为 400）**：

| 条件 | HTTP | message 要点 |
|------|------|--------------|
| `old_password` 与当前密码不符 | 400 | 当前密码错误 |
| `new_password` 与 `confirm_password` 不一致 | 400 | 两次新密码不一致 |
| `username` trim 后为空且 `new_password` 为空（无任何更改） | 400 | 未做出更改 |

**请求示例（仅改密）**

```http
PUT /api/v1/settings/account HTTP/1.1
Authorization: Bearer sk-1a2b3c4d...
Content-Type: application/json

{
  "username": "",
  "old_password": "oldpass1",
  "new_password": "newpass2",
  "confirm_password": "newpass2"
}
```

**成功响应（200）**

```json
{
  "code": 200,
  "message": "ok",
  "data": { "changed_user": false, "changed_pwd": true, "username": "" }
}
```

> 说明：密码存储沿用原版规则 `md5("lylme" + 明文)`，全项目唯一实现。本端点仅影响后台登录，不影响 API 密钥鉴权。

---

## 7. GET /settings/system —— 读取系统安全设置

- **鉴权**：Bearer 当前密钥。
- **入参**：无。
- **出参 `data`**：

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | `admin_path` | string | 后台目录名；未配置时回退到应用配置（`.env` 的 `ADMIN_PATH`，默认 `admin`） |
  | `debug` | string | 调试模式，仅 `"0"`/`"1"`；未配置或非 `"1"` 一律返回 `"0"` |

**成功响应（200）**

```json
{
  "code": 200,
  "message": "ok",
  "data": { "admin_path": "admin", "debug": "0" }
}
```

**错误**：401、429、500。

---

## 8. PUT /settings/system —— 更新系统安全设置

- **鉴权**：Bearer 当前密钥。
- **入参（JSON object）**：

  | 字段 | 类型 | 约束 | 说明 |
  |------|------|------|------|
  | `admin_path` | string | **必填**；trim 后非空且匹配 `^[A-Za-z0-9_-]+$` | 后台目录名 |
  | `debug` | string | **必填**，枚举 `"0"` / `"1"` | 调试模式 |

  两项校验全部通过后才落库；任一项非法则不写任何配置。

- **出参 `data`**：`{ "updated": true }`

**请求示例**

```http
PUT /api/v1/settings/system HTTP/1.1
Authorization: Bearer sk-1a2b3c4d...
Content-Type: application/json

{ "admin_path": "panel-1", "debug": "1" }
```

**错误**：

| HTTP | 触发条件 |
|------|----------|
| 400 | `admin_path` 缺失/为空/含字母数字下划线中划线以外字符（如 `a/b`、`a b`、`../x`、中文）；`debug` 非 `"0"`/`"1"` |
| 401 / 429 / 500 | 见通用约定 |

> 注意：`admin_path` 落库后于**下次启动**注册路由时生效，需通过新路径访问后台。

---

## 9. GET /wxplus —— 读取微信推送配置

- **鉴权**：Bearer 当前密钥。
- **入参**：无。
- **出参 `data`**：

  | 字段 | 类型 | 落库配置键 | 说明 |
  |------|------|------------|------|
  | `key` | string | `wxplus` | 微信推送密钥；空串表示未开启 |
  | `time` | string | `wxplustime` | 每日推送时间（如 `"20:00"`） |

**成功响应（200）**

```json
{
  "code": 200,
  "message": "ok",
  "data": { "key": "", "time": "20:00" }
}
```

**错误**：401、429、500。

---

## 10. PUT /wxplus —— 更新微信推送配置

- **鉴权**：Bearer 当前密钥。
- **入参（JSON object，字段均可空，空 key 表示关闭推送，按原始提交值落库）**：

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | `key` | string | 微信推送密钥 |
  | `time` | string | 每日推送时间 |

- **出参 `data`**：`{ "updated": true }`

**请求示例**

```http
PUT /api/v1/wxplus HTTP/1.1
Authorization: Bearer sk-1a2b3c4d...
Content-Type: application/json

{ "key": "WXSECRET", "time": "08:30" }
```

**错误**：400（JSON 格式错误）、401、429、500。

---

## 11. GET /license —— 读取网站授权配置

- **鉴权**：Bearer 当前密钥。
- **入参**：无。
- **出参 `data`**：

  | 字段 | 类型 | 落库配置键 | 说明 |
  |------|------|------------|------|
  | `code` | string | `c` | 程序授权码；未配置为空串 |
  | `hostmode` | string | `hostmode` | 域名获取方式：`"1"` = HTTP_HOST，`"2"` = SERVER_NAME；缺省或其他值归一为 `"2"` |

**成功响应（200）**

```json
{
  "code": 200,
  "message": "ok",
  "data": { "code": "", "hostmode": "2" }
}
```

**错误**：401、429、500。

---

## 12. PUT /license —— 更新网站授权配置

- **鉴权**：Bearer 当前密钥。
- **入参（JSON object）**：

  | 字段 | 类型 | 约束 | 说明 |
  |------|------|------|------|
  | `code` | string | 可空 | 程序授权码，按原始提交值落库 |
  | `hostmode` | string | **必填**，枚举 `"1"` / `"2"` | 域名获取方式 |

- **出参 `data`**：`{ "updated": true }`

**请求示例**

```http
PUT /api/v1/license HTTP/1.1
Authorization: Bearer sk-1a2b3c4d...
Content-Type: application/json

{ "code": "LIC-123", "hostmode": "1" }
```

**错误**：400（`hostmode` 缺失或非 `"1"`/`"2"`）、401、429、500。

---

## 13. GET /apikey —— 查询当前 API 密钥信息

只返回密钥是否存在、创建时间与掩码，**任何情况下都不回读完整明文密钥**。

- **鉴权**：Bearer 当前密钥（能通过本端点鉴权即证明密钥有效）。
- **入参**：无。
- **出参 `data`**：

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | `exists` | bool | 当前是否存在有效密钥 |
  | `created_at` | string | 密钥创建时间，**RFC3339** 格式；无法解析时为空串 |
  | `masked` | string | 密钥掩码，形如 `sk-xxxx…`（保留 `sk-` 与其后 4 个十六进制字符，其余以 `…` 代替）；密钥不存在时为空串 |

**成功响应（200）**

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

**错误**：401、429、500。

---

## 14. POST /apikey/regenerate —— 重新生成 API 密钥

> ## ⚠️ 旧密钥立即失效
>
> 重新生成成功的瞬间，**旧密钥立即、永久失效**，无法恢复。所有使用旧密钥的 REST API / MCP 客户端、脚本与 CI 配置必须立即更换为响应中返回的新密钥，否则随后的请求一律返回 **401**。

- **鉴权**：Bearer **当前**（即将被作废的）密钥。中间件在本请求内仍以旧密钥鉴权，service 在处理时完成轮换。
- **入参**：无请求体。
- **出参 `data`（本端点是唯一一次返回完整明文的时机）**：

  | 字段 | 类型 | 说明 |
  |------|------|------|
  | `key` | string | 新密钥完整明文，形如 `sk-` + 48 个十六进制字符；**仅此一次展示，请立即妥善保存** |
  | `created_at` | string | 新密钥创建时间，RFC3339 格式 |

**请求示例**

```http
POST /api/v1/apikey/regenerate HTTP/1.1
Authorization: Bearer sk-旧密钥...
```

**成功响应（200）**

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "key": "sk-9f8e7d6c5b4a...（48 位 hex）",
    "created_at": "2026-10-04T11:30:00+08:00"
  }
}
```

**轮换后的验证时序**

1. 用新密钥请求任意端点（如 `GET /apikey`）→ **200**；
2. 用旧密钥请求任意端点 → **401**（无效或已失效的 API 密钥）。

**错误**：401（旧密钥在调用前已失效）、429、500。

---

## 端点清单速览

| # | 方法 | 路径 | 用途 |
|---|------|------|------|
| 1 | GET | `/settings` | 读取全部配置（屏蔽 `admin_pwd`、`api_key`） |
| 2 | PUT | `/settings` | 批量更新网站设置 |
| 3 | GET | `/settings/about` | 读取关于页 |
| 4 | PUT | `/settings/about` | 更新关于页 |
| 5 | POST | `/settings/about/reset` | 恢复关于页默认内容 |
| 6 | PUT | `/settings/account` | 修改管理员用户名/密码 |
| 7 | GET | `/settings/system` | 读取系统安全设置 |
| 8 | PUT | `/settings/system` | 更新系统安全设置 |
| 9 | GET | `/wxplus` | 读取微信推送配置 |
| 10 | PUT | `/wxplus` | 更新微信推送配置 |
| 11 | GET | `/license` | 读取授权配置 |
| 12 | PUT | `/license` | 更新授权配置 |
| 13 | GET | `/apikey` | 查询密钥（存在性/创建时间/掩码） |
| 14 | POST | `/apikey/regenerate` | 重新生成密钥（**旧密钥立即失效**） |
