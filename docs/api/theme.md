# Theme 域 API（/api/v1）

主题域端点用于查看可用主题、切换当前主题，以及读写主题的自定义配置（对应主题目录下 `config.php` 定义的表单字段）。

## 通用约定

- Base URL：`/api/v1`
- 鉴权：全部端点均需 `Authorization: Bearer sk-...`（API 密钥），缺失或无效返回 401。
- 限流：默认每 IP 120 次/分钟，超限返回 429。
- 统一响应信封：

  ```json
  { "code": 200, "message": "ok", "data": { ... } }
  ```

- 错误码：HTTP 状态码与业务码一致（400 / 401 / 404 / 429 / 500）。

## 错误码

| 状态码 | 触发条件 |
|--------|----------|
| 400 | 请求体非合法 JSON、缺少必填字段（如 `name`）、`values`/`data` 不是对象 |
| 401 | 缺少 Authorization 头、非 Bearer 方案、密钥无效或已失效 |
| 404 | 切换主题时目标主题目录不存在（ErrThemeNotFound） |
| 429 | 超出 IP 限流阈值 |
| 500 | 服务端内部错误 |

---

## 1. GET /themes —— 主题列表与当前主题

- **用途**：返回全部已发现主题（`template/*` 下含 `theme.ini` 的目录）及当前启用主题目录名。
- **鉴权**：Bearer API Key。
- **入参**：无。

### 出参 data

| 字段 | 类型 | 说明 |
|------|------|------|
| current | string | 当前主题目录名；配置缺失或指向不存在主题时回退 `default` |
| themes | array | 主题列表（按目录名排序） |
| themes[].dir | string | 主题目录名（稳定标识，切换/配置时使用） |
| themes[].name | string | 主题显示名（theme.ini 的 theme_name，缺省取目录名） |
| themes[].author | string | 作者（author_name） |
| themes[].version | string | 版本（theme_version） |
| themes[].description | string | 主题说明（theme_explain） |

### 示例

请求：

```http
GET /api/v1/themes
Authorization: Bearer sk-xxxx
```

响应：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "current": "default",
    "themes": [
      {
        "dir": "5iux",
        "name": "五零优享",
        "author": "lylme",
        "version": "1.0",
        "description": "简洁主题"
      },
      {
        "dir": "default",
        "name": "default",
        "author": "六零二开",
        "version": "2.0",
        "description": "默认主题"
      }
    ]
  }
}
```

---

## 2. PUT /themes/current —— 切换当前主题

- **用途**：将当前主题切换为指定目录名，写入配置键 `template`，前台首页随即使用新主题。
- **鉴权**：Bearer API Key。

### 入参（JSON body）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 目标主题目录名（trim 后不能为空） |

### 出参 data

| 字段 | 类型 | 说明 |
|------|------|------|
| current | string | 切换后的当前主题目录名 |

### 错误码

- 400：`name` 缺失或为空。
- 404：目标主题目录不存在。

### 示例

请求：

```http
PUT /api/v1/themes/current
Authorization: Bearer sk-xxxx
Content-Type: application/json

{"name": "default"}
```

响应：

```json
{
  "code": 200,
  "message": "ok",
  "data": { "current": "default" }
}
```

切换不存在的主题：

```http
PUT /api/v1/themes/current

{"name": "no-such-theme"}
```

响应（HTTP 404）：

```json
{
  "code": 404,
  "message": "theme: 主题不存在",
  "data": null
}
```

---

## 3. GET /themes/config —— 主题配置字段定义与当前值

- **用途**：返回当前主题的配置表单定义（解析自主题 `config.php`）与已保存的自定义配置（配置键 `theme_config_<tpl>`）。
- **鉴权**：Bearer API Key。
- **入参**：无（始终针对当前主题）。

### 出参 data

| 字段 | 类型 | 说明 |
|------|------|------|
| theme | string | 配置所属主题目录名 |
| fields | array | 字段定义列表 |
| fields[].type | string | 表单类型：`text`/`textarea`/`select`/`checkbox`/`radio`/`color`/`switch` |
| fields[].name | string | 配置键（主题内唯一） |
| fields[].title | string | 字段标题 |
| fields[].description | string | 字段说明（可能含 HTML） |
| fields[].verify | string | 校验方式：`required`/`url`/`number`/`email` 等 |
| fields[].placeholder | string | 占位提示 |
| fields[].value | any | 默认值：字符串，或字符串数组（checkbox） |
| fields[].enum | array | 枚举项（select/radio/checkbox 使用，其余为空数组） |
| fields[].enum[].value | string | 枚举值 |
| fields[].enum[].label | string | 枚举显示文本 |
| values | object | 已保存的自定义配置键值 map；从未保存过时为 `{}`（不为 null） |

### 示例

请求：

```http
GET /api/v1/themes/config
Authorization: Bearer sk-xxxx
```

响应：

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

---

## 4. PUT /themes/config —— 保存主题自定义配置

- **用途**：全量保存某主题的自定义配置，JSON 序列化后写配置键 `theme_config_<tpl>`。
  未提交的 `switch` 字段自动补 `0`，未提交（或提交空字符串）的 `checkbox` 字段自动补空数组。
- **鉴权**：Bearer API Key。

### 入参（JSON body）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| theme | string | 否 | 主题目录名；为空或指向不存在的主题时回退当前主题 |
| values | object | 否 | 配置键值对象（推荐形式） |
| data | object | 否 | `values` 的别名，二者择一 |

body 支持三种等价形式：

1. values 包裹：

   ```json
   {"theme": "default", "values": {"lytoday": "1", "gonganbei": "京公安网备1号"}}
   ```

2. data 包裹：

   ```json
   {"theme": "default", "data": {"lytoday": "1"}}
   ```

3. 平铺（除 `theme` 外的键全部视为配置键）：

   ```json
   {"theme": "", "lytoday": "1", "gonganbei": ""}
   ```

> 注意：保存为**全量替换**语义——未出现在本次请求中的既有配置键会被覆盖移除（字段定义层面的 switch/checkbox 默认值补齐除外）。

### 出参 data

| 字段 | 类型 | 说明 |
|------|------|------|
| theme | string | 实际生效的主题目录名 |
| values | object | 保存后重新读回的配置值（即随后 GET /themes/config 的 values） |

### 错误码

- 400：请求体非 JSON 对象；`values`/`data` 存在但不是对象。

### 示例

请求：

```http
PUT /api/v1/themes/config
Authorization: Bearer sk-xxxx
Content-Type: application/json

{"theme":"default","values":{"lytoday":"1"}}
```

响应：

```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "theme": "default",
    "values": {
      "lytoday": "1"
    }
  }
}
```

> 说明：`values` 仅反映 `theme_config_<tpl>` 中本次提交并保存的内容（switch/checkbox 缺省补齐除外），不与其他字段的默认值合并；字段默认值请参考 GET 返回的 `fields[].value`。
