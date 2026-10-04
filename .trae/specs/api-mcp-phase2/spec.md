# 第二期改造：后端 API 化 + MCP 服务 Spec

## 问题背景（Why）

第一期已完成 PHP → Go(Gin)+GORM 的重写，但后台所有操作（链接/分组/搜索引擎/收录/主题等管理）只以「HTML 页面 + 表单/AJAX」形式存在，业务逻辑直接耦合在 [internal/admin](file:///Volumes/data/lylm_go/internal/admin) 的处理器中（参数解析、业务规则、HTML 渲染混写），外部程序无法以编程方式调用。

第二期需要：

1. 把后端**全部管理操作**实现为标准化 REST API（`/api/v1`）；
2. 使用 `github.com/mark3labs/mcp-go v1.0.0`，在 `/mcp` 路由以 Streamable HTTP（无状态）方式提供 MCP 服务，MCP 工具与 API 操作一一对应；
3. 在后台新增独立菜单页面，管理 API/MCP 密钥（`sk-` 开头，单一密钥），提供可一键复制、自带上 key 的 MCP 配置；
4. 抽出与 HTTP/HTML 无关的共享逻辑层（service），后台模板处理器与 API/MCP 处理器都调用同一逻辑层，不重复实现业务代码。

## 用户（Who）

- **站长（管理员）**：通过后台新菜单生成/查看/复制密钥与 MCP 配置；继续使用现有后台页面。
- **外部程序 / 脚本开发者**：持 `sk-` key 调用 `/api/v1` REST 接口管理站点。
- **AI 客户端 / Agent（MCP Client）**：通过 `https://站点/mcp` 以 Bearer 密钥接入，调用 MCP 工具完成导航站管理操作。

## 目标（Goals）

- 后台全部管理操作均有对应的 REST API 端点，且每个端点具备明确的用途、入参规格、出参结构与错误码文档。
- `/mcp` 提供规范可用的 MCP Streamable HTTP 服务；每个 MCP 工具有清晰描述（用途）与参数 schema（入参规格），返回结构化结果。
- 同一把 `sk-` key 同时是 REST API 与 MCP 的认证凭证；后台独立菜单页面可生成/重新生成密钥、复制密钥与自带上 key 的 MCP 配置。
- 业务逻辑集中在独立的 service 层；admin HTML 处理器、REST API 处理器、MCP 工具处理器均为薄适配层，复用同一逻辑层。
- 现有后台 URL（`.php` 兼容路径）、页面外观、前端 JS 与响应行为完全不变。

## 非目标（Non-Goals）

- 不改前台首页、主题模板、固定功能页（about/apply/pwd/site 等）的现有行为与 URL。
- 不把后台页面改造为「JS 调 API」的 SPA 模式；后台仍走会话 Cookie + CSRF + 服务端渲染。
- 不提供 OAuth 2.0 授权流程、多用户/多角色体系；API/MCP 仅使用单一静态密钥。
- 不提供 MCP stdio 传输；不提供服务端主动通知（无状态模式）。
- 后台登录本身不开放为 API（API 认证用独立密钥，不复用管理员账号密码）。

## 已确认决策

| 决策点 | 结论 |
|---|---|
| 密钥数量 | **单一密钥**：仅一个有效 key；重新生成后旧 key 立即失效 |
| API 前缀 | **`/api/v1`** |
| MCP 传输 | Streamable HTTP，**无状态 stateless**，每请求 Bearer 鉴权 |
| 后台页面 | URL/UI/JS 完全不变，仅把底层业务逻辑抽到 service |

## ADDED Requirements

### Requirement: 共享业务逻辑层

系统 SHALL 在 `internal/service`（或等价独立包）中实现全部业务逻辑；service 层只依赖 GORM/模型/配置/工具包，**不得依赖 gin 或任何 HTML/模板渲染**，并以强类型入参/出参 + 哨兵错误（参数错误、记录不存在等）对外表达结果。

后台 HTML 处理器、REST API 处理器、MCP 工具处理器 SHALL 仅负责协议适配（参数解析、鉴权、序列化、渲染），业务规则一律调用 service 层，不得在三处重复实现。

依赖方向 SHALL 保持 `router → admin/api/mcp → service → site/models → database/config`，无反向与循环依赖。

### Requirement: 密钥（API Key）

系统 SHALL 支持单一 API/MCP 密钥：

- 格式为 `sk-` 前缀 + 随机串；随机部分使用 `crypto/rand` 生成，熵不低于 128 bit。
- 密钥（及创建时间）持久化于配置存储（`lylme_config` 键值，如 `api_key` / `api_key_created_at`），不新建数据库表。
- 支持「生成（尚不存在时）」「查看」「重新生成」；重新生成 SHALL 使旧 key 立即失效。
- 密钥校验 SHALL 使用恒定时间比较；密钥 SHALL NOT 出现在任何日志中。
- 密钥在 SQLite / MySQL 下行为一致。

### Requirement: REST API（/api/v1）

系统 SHALL 在 `/api/v1` 下为后台**全部管理操作**提供 REST 端点，覆盖范围（与 [internal/admin](file:///Volumes/data/lylm_go/internal/admin) 现有操作一一对应）：

1. **仪表盘**：汇总数据（链接/分组/待审数量、版本信息、近 7 天访问统计）；
2. **网站设置**：读取/批量更新站点配置、关于页内容（含恢复默认）、管理员账号（用户名/密码修改）、系统安全（后台目录、调试模式）；
3. **分组**：列表（含链接计数）、增、改、删（同时删除组内链接）、启停用、排序；
4. **链接**：分页/按组/关键词查询、单条详情、增（含查重与颜色→`<font>` 处理）、改、批量删除、批量启停用、批量移动分组、批量设置加密、批量排序、抓取目标站标题/图标、全量简表；
5. **导航菜单（Tag）**：列表、增、改、删；
6. **搜索引擎（Sou）**：列表、增、改、删；
7. **加密组（Pwd）**：列表、增、改、删；
8. **收录申请（Apply）**：列表（按状态）、收录开关与公告设置、修改申请、审核通过（写入链接表）/拒绝、批量删除；
9. **主题**：主题列表与当前主题、切换主题、主题自定义配置字段定义与当前值、保存主题自定义配置；
10. **文件清理**：扫描未使用文件、删除指定未使用文件；
11. **微信推送 / 网站授权**：读取/保存对应配置；
12. **密钥管理**：读取当前密钥信息、重新生成（允许通过 API 使用当前有效密钥操作）。

统一约定：

- 请求体与响应体均为 `application/json; charset=utf-8`；
- 成功响应统一信封：`{"code":200,"message":"ok","data":<出参结构>}`；
- 失败响应统一信封：`{"code":<业务码>,"message":"<错误说明>","data":null}`，并使用恰当 HTTP 状态码（400/401/403/404/409/429/500）；
- 分页列表统一出参：`data` 含 `list`、`page`、`page_size`、`total`；
- 每个端点必须包含输入校验（长度、类型、枚举、必填），校验失败返回 400 与明确字段错误信息。

### Requirement: API/MCP 鉴权

系统 SHALL 对 `/api/v1/**` 与 `/mcp` 的全部请求强制鉴权：

- 凭证通过 HTTP 头 `Authorization: Bearer sk-xxxx` 提供；
- 缺失、格式错误、密钥不存在或不匹配时返回 `401`（REST 返回统一 JSON 错误；`/mcp` 返回 JSON-RPC/HTTP 层 401）；
- 鉴权中间件 SHALL 复用 `webutil.RateLimit` 对调用方（按 IP）进行限流，超限返回 `429`；
- API/MCP 请求 SHALL NOT 计入前台访问统计 PV。

### Requirement: MCP 服务（/mcp）

系统 SHALL 使用 `github.com/mark3labs/mcp-go v1.0.0` 在 `/mcp` 路由提供 MCP Streamable HTTP 服务：

- 传输为**无状态模式**（`WithStateLess(true)`），不依赖会话粘滞；
- 路由通过 Gin 挂载（`StreamableHTTPServer` 作为 `http.Handler` 接入），并由 Bearer 密钥中间件保护；
- 工具 SHALL 与「后台全部管理操作」一一对应，工具命名稳定、语义清晰（如 `link_list`、`link_create`、`group_delete`）；
- 每个工具 SHALL 声明：用途描述（`description`）、入参 JSON Schema（每个参数的类型、是否必填、含义、枚举/约束）；
- 工具执行 SHALL 调用 service 层；成功时返回结构化 JSON（文本内容为 JSON，出参结构与对应 REST 端点一致），业务错误以 `IsError` 的工具结果返回并附带错误说明；
- MCP 服务 SHALL 通过标准 MCP 握手（`initialize`）、`tools/list`、`tools/call` 的验证。

### Requirement: 后台「接口与 MCP」管理页面

系统 SHALL 在后台侧边栏新增一个独立菜单项（沿用现有后台模板风格、布局与图标体系），页面包含：

- 当前密钥（`sk-` 开头）的完整展示与「复制密钥」按钮；密钥不存在时提供「生成密钥」按钮；
- 「重新生成密钥」操作，并明确提示旧密钥立即失效，需二次确认；
- API 基础地址（`<站点根>/api/v1`）展示与复制；
- MCP 接入地址（`<站点根>/mcp`）展示与复制；
- MCP 配置块（JSON，Streamable HTTP 客户端格式），其 `url` 指向 `/mcp`、`headers.Authorization` 已内嵌当前 key，并提供「复制配置」按钮，例如：

```json
{
  "mcpServers": {
    "lylme": {
      "url": "https://example.com/mcp",
      "headers": { "Authorization": "Bearer sk-xxxx" }
    }
  }
}
```

- 该页面与其他后台页面一样受会话登录保护；复制交互通过 JS 实现，不引入新 UI 框架。

### Requirement: 接口文档

系统 SHALL 提供一份 API/MCP 接口文档（`docs/api-v1.md`），对每个端点逐项说明：用途、HTTP 方法与路径、鉴权要求、入参规格（参数名、位置、类型、必填、长度/枚举等约束）、出参结构（字段名、类型、含义）、可能的错误码与触发条件；并给出每个 MCP 工具与端点的对应关系。文档内容 SHALL 与最终实现一致。

## 非功能需求

- **兼容性**：所有新增持久化（仅配置键值）与查询在 SQLite、MySQL 下均可工作；不改动现有表结构（如确需新增字段须双兼容并同步迁移）。
- **安全**：`crypto/rand` 生成密钥；恒定时间比较；密钥不打日志、不进错误页；写操作经 service 层做输入校验；抓取类接口沿用现有 SSRF 防护（[internal/fetch](file:///Volumes/data/lylm_go/internal/fetch)）。
- **可维护性**：三处入口零业务逻辑重复；错误处理与响应信封统一。
- **可测试性**：service 核心逻辑、鉴权中间件、API 端点、MCP 工具均有自动化测试。
- **日志**：统一使用 [internal/logger](file:///Volumes/data/lylm_go/internal/logger)，记录方法、路径、状态码、耗时；不得使用 `fmt.Println`。

## 约束与依赖

- 新增直接依赖：`github.com/mark3labs/mcp-go v1.0.0`（及其传递依赖），通过 `go get` 固定版本。
- 遵守 [AGENTS.md](file:///Volumes/data/lylm_go/AGENTS.md)：分层依赖、URL 兼容、安全（参数化查询/限流/SSRF）、日志、按阶段提交。
- 构建环境：

```bash
export PATH=/Users/macbookpro/sdk/go1.26.2/bin:$PATH
go build ./...
go vet ./...
go test ./...
```

## 假设

- MCP 客户端均支持 Streamable HTTP 传输与自定义请求头（当前主流 AI 客户端均支持）。
- 站点对外根地址可由请求 Host / `X-Forwarded-Proto` 推导（沿用现有 `siteURL` 逻辑）。

## 验收标准（Acceptance Criteria）

### AC-1 全量操作 API 覆盖（rule）
- **通过条件**：对照本 spec「REST API」覆盖范围 1–12 的每一项操作，均存在已注册的 `/api/v1` 端点，且覆盖 [internal/admin](file:///Volumes/data/lylm_go/internal/admin) `Register` 中全部查询与写操作（登录页除外）；端点清单可从路由注册与 `docs/api-v1.md` 逐一核对。
- **证据来源**：路由注册代码、`docs/api-v1.md`、API 集成测试。

### AC-2 API/MCP 强制鉴权（rule）
- **通过条件**：对任意 `/api/v1/**` 与 `/mcp` 请求：无 `Authorization` 头、非 Bearer 格式、错误或已失效 key 均返回 401；持有效 key 请求成功；超限流阈值返回 429；且这些请求不产生前台 PV 计数。
- **证据来源**：鉴权中间件单测、curl/httptest 记录、统计中间件代码。

### AC-3 /mcp MCP 服务可用且工具齐全（rule）
- **通过条件**：持有效 key 对 `/mcp` 完成 `initialize`；`tools/list` 返回的工具集合与后台管理操作一一对应（数量与名称可对照文档）；至少对每个域各一个工具执行 `tools/call` 成功并返回 JSON 结果；错误参数返回 `IsError` 结果。
- **证据来源**：MCP 集成测试、MCP Inspector / curl 握手记录。

### AC-4 接口契约文档完整（rule）
- **通过条件**：`docs/api-v1.md` 为每个端点写明用途、方法路径、鉴权、入参规格（名/位置/类型/必填/约束）、出参结构、错误码；并列出 MCP 工具与端点对应表；文档描述与代码实现（路由、请求/响应结构体）一致，无遗漏端点。
- **证据来源**：`docs/api-v1.md`、源码人工核对、评审记录。

### AC-5 后台密钥/MCP 管理页面（rule）
- **通过条件**：后台侧边栏出现独立菜单；页面可生成（首次）/查看/重新生成 `sk-` key，复制 key、API 地址、MCP 地址、已内嵌 key 的 MCP 配置 JSON；重新生成后旧 key 调用 API/MCP 返回 401、新 key 调用成功；页面受登录保护。
- **证据来源**：浏览器操作截图/记录、新旧 key 调用对比、模板与路由代码。

### AC-6 共享逻辑层与零重复（rule）
- **通过条件**：全部业务规则位于 `internal/service`，且该包不 import `github.com/gin-gonic/gin`、不做 HTML 渲染；admin、api、mcp 三个处理器包中不存在重复的业务规则实现（同名规则只出现在 service）；`go vet` 无循环依赖。
- **证据来源**：依赖分析（grep import）、代码评审。

### AC-7 现有后台行为不变（rule）
- **通过条件**：重构后现有后台全部 `.php` 兼容路由可正常访问；页面渲染、AJAX JSON（`{code,msg}`）、HTML 片段、表单提交结果与重构前等价；前台首页与固定功能页行为不变。
- **证据来源**：重构前后手工/脚本回归记录、现有测试通过。

### AC-8 构建与自动化测试（rule）
- **通过条件**：在指定 Go 环境下 `go build ./...`、`go vet ./...`、`go test ./...` 全部通过；新增测试覆盖 service 关键规则（链接查重、审核通过写链接、密钥生成/校验等）、鉴权中间件、至少一个 API 端点全链路与一个 MCP 工具调用全链路。
- **证据来源**：命令输出与测试报告。

### AC-9 接口契约质量（rubric，0–3，通过阈值 ≥ 2.5）
- **维度**：入参/出参 schema 的完整度与精确性、错误场景覆盖、命名一致性、文档与实现一致性。
- **锚点**：0=大量接口无规格或规格错误；1=关键接口有规格但粗略/多处不一致；2=全部接口规格完整、偶有小瑕疵；3=规格精确到字段级约束且与实现、MCP schema 完全一致。
- **证据来源**：`docs/api-v1.md`、MCP 工具 schema、独立评审。

### AC-10 重构工程质量（rubric，0–3，通过阈值 ≥ 2.5）
- **维度**：分层清晰度、重复代码消除、错误处理统一性、可测试性、对现有行为的零回归。
- **锚点**：0=逻辑仍混在处理器中或出现回归；1=分层存在但仍有明显重复/风格不统一；2=分层清晰、三处适配统一、仅少量可改进点；3=薄适配+单一逻辑层、错误/校验/命名全面统一、零回归。
- **证据来源**：代码评审、测试结果。
