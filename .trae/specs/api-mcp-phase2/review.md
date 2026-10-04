# 第二期改造「API 化 + MCP 服务」独立评审报告

- 评审日期：2026-10-04
- 评审人：独立评审 agent（只读）
- 评审基准：[AGENTS.md](file:///Volumes/data/lylm_go/AGENTS.md)、[spec.md](file:///Volumes/data/lylm_go/.trae/specs/api-mcp-phase2/spec.md)（AC-1～AC-10）、[tasks.md](file:///Volumes/data/lylm_go/.trae/specs/api-mcp-phase2/tasks.md)、[docs/api-v1.md](file:///Volumes/data/lylm_go/docs/api-v1.md)
- 说明：交接摘要中提及的 `checklist.md` 实际不存在，本评审未使用该文件。

## 总体结论：PASS

| 指标 | 数量 |
|---|---|
| Blocker | 0 |
| Major | 0 |
| Minor | 1 |
| Nit | 2 |

- AC-1～AC-8（rule）：全部 **PASS**。
- AC-9（rubric）：**2.7 / 3**（通过阈值 ≥ 2.5）。
- AC-10（rubric）：**2.8 / 3**（通过阈值 ≥ 2.5）。

评审方法：源码/文档人工逐条核对、grep 依赖与安全检查、独立复跑 `go build/vet/test`、以及用真实编译二进制 + 隔离 SQLite 库完成端到端 curl 链路（401 拦截 → 登录 → 密钥 ensure → REST 鉴权请求 → MCP initialize/tools/list/tools/call → 密钥 regenerate → 前后台回归）。

---

## 逐条 AC 裁定

### AC-1 全量操作 API 覆盖 — PASS（2.5/3）

**证据：**

- 路由统一入口 [internal/api/router.go](file:///Volumes/data/lylm_go/internal/api/router.go)：`v1 := r.Group("/api/v1", RequireKey(d.Keys))`，注册 ping + 9 个 Register 函数，共 57 个路由（1 ping + 56 业务）。
- 逐域清点与 spec 覆盖范围 1–12 对照：
  - group 6（[group_handler.go#L13-L20](file:///Volumes/data/lylm_go/internal/api/group_handler.go#L13-L20)）
  - link 11（[link_handler.go#L16-L28](file:///Volumes/data/lylm_go/internal/api/link_handler.go#L16-L28)）
  - tag/sou/pwd 各 4（CRUD）
  - settings 域 14 个注册（[settings_handler.go#L48-L69](file:///Volumes/data/lylm_go/internal/api/settings_handler.go#L48-L69)）
  - apply 6（[apply_handler.go#L16-L23](file:///Volumes/data/lylm_go/internal/api/apply_handler.go#L16-L23)）
  - theme 4（[theme_handler.go#L19-L24](file:///Volumes/data/lylm_go/internal/api/theme_handler.go#L19-L24)）
  - dashboard/files 3（[dashboard_handler.go#L67-L98](file:///Volumes/data/lylm_go/internal/api/dashboard_handler.go#L67-L98)）
- 未发现遗漏端点。apply_create 有意不提供（REST 与 MCP 均无，与决策一致）。
- 唯一与 spec 字面覆盖项的偏差：仪表盘未返回「近 7 天访问统计」，见 Finding F-1（Minor）。端点本身存在且可用，不构成 AC-1 失败。

### AC-2 API/MCP 强制鉴权 — PASS（3/3）

**证据：**

- [internal/api/auth.go#L25-L47](file:///Volumes/data/lylm_go/internal/api/auth.go#L25-L47)：限流先于鉴权；必须有 `Bearer ` 前缀；token 须 `sk-` 开头且 `keys.Validate` 通过；默认 120 次/分钟/IP（[auth.go#L13-L17](file:///Volumes/data/lylm_go/internal/api/auth.go#L13-L17)），超限 429。
- MCP Mount 复用同一 `api.RequireKey`（[internal/mcp/http.go#L26-L36](file:///Volumes/data/lylm_go/internal/mcp/http.go#L26-L36)），REST 与 MCP 共用限流桶。
- 独立 curl 复跑（真实二进制，端口 19123，隔离库）：
  - 无凭证 `GET /api/v1/ping` → 401；无凭证 body 返回统一信封 `{"code":401,...,"data":null}`；
  - 错误 key（`Bearer sk-wrong`）→ 401；非 Bearer 格式（`Authorization: sk-xxxx`）→ 401；
  - 无凭证 `POST /mcp` → 401；
  - 持有效 key → 200。
- 429：[auth_test.go#L96](file:///Volumes/data/lylm_go/internal/api/auth_test.go#L96) 连续请求触发限流，断言通过。
- PV 排除：[internal/stats/stats.go#L89](file:///Volumes/data/lylm_go/internal/stats/stats.go#L89) skipPrefixes 含 `/api`、`/mcp`；[stats_test.go#L60-L78](file:///Volumes/data/lylm_go/internal/stats/stats_test.go#L60-L78) 实测两者各访问一次 PV=0。

### AC-3 /mcp MCP 服务可用且工具齐全 — PASS（3/3）

**证据：**

- 直接依赖：go.mod 第 10 行 `github.com/mark3labs/mcp-go v1.0.0`（主 require 块）。
- 无状态 Streamable HTTP：[internal/mcp/http.go#L14-L19](file:///Volumes/data/lylm_go/internal/mcp/http.go#L14-L19)（`WithStateLess(true)`）；显式注册 `/mcp` 与 `/mcp/` 的 GET/POST/DELETE，全部先过 Bearer 鉴权。
- 独立 curl 复跑：`initialize` → HTTP 200，返回 `protocolVersion: 2025-11-25`、`serverInfo: lylme-navigation v2.7.0`，无 `Mcp-Session-Id`（无状态）。
- `tools/list` → **恰好 56 个工具**，与期望清单（见下）逐一程序比对：缺失 0、多余 0；每个工具含 `name`/`description`/`inputSchema`(object)。前缀分布：group6/link11/tag4/sou4/pwd4/settings8/wxplus2/license2/apikey2/apply6/theme4/dashboard1/files2。
- `tools/call group_list` → 200，`content[0].text` 为 JSON 字符串，结构与 `GET /api/v1/groups` 的 data 一致。
- 每域 call 成功、参数错误返回 IsError：[mcp_test.go#L410](file:///Volumes/data/lylm_go/internal/mcp/mcp_test.go#L410)、[mcp_test.go#L530](file:///Volumes/data/lylm_go/internal/mcp/mcp_test.go#L530) 覆盖并通过。

### AC-4 接口契约文档完整 — PASS（3/3）

**证据：**

- [docs/api-v1.md](file:///Volumes/data/lylm_go/docs/api-v1.md) 共 2779 行：§1 概述、§2 通用约定（2.1–2.8）、§3 REST（3.1–3.10，57 个端点小节）、§4 MCP（4.1–4.5）、§5 变更记录。
- 每个端点均写明：用途、方法+路径、鉴权、入参（名/位置/类型/必填/约束）、出参、错误码与触发条件，并附请求/响应示例。
- MCP 对应总表：[docs/api-v1.md#L2709-L2770](file:///Volumes/data/lylm_go/docs/api-v1.md#L2709-L2770) 列出全部 56 个工具↔REST 对应关系，与实际工具名核对无差异。
- 文档与实现的字段级抽查详见 AC-9。

### AC-5 后台密钥/MCP 管理页面 — PASS（3/3）

**证据：**

- 侧边栏菜单：[web/admin/head.html#L47](file:///Volumes/data/lylm_go/web/admin/head.html#L47)「接口与 MCP」（mdi-console-line，Active "mcp"）。
- 页面 [web/admin/mcp.html](file:///Volumes/data/lylm_go/web/admin/mcp.html) 三区齐全：密钥区（完整 key + 复制 + 生成 + 重新生成）、地址区（API base、MCP endpoint + 复制）、配置区（内嵌 key 的 mcpServers JSON textarea + 复制）；复制含 `navigator.clipboard` + `execCommand` 降级；重新生成经 jquery-confirm 二次确认并提示旧 key 失效。
- 处理器 [internal/admin/apikey_page.go](file:///Volumes/data/lylm_go/internal/admin/apikey_page.go)：MCPPage（GET）、AjaxMCP（POST ensure/regenerate），**有 CSRF 校验**（X-CSRF-Token / csrf_token，[apikey_page.go#L81-L88](file:///Volumes/data/lylm_go/internal/admin/apikey_page.go#L81-L88)）；路由受 `auth`/`frag`（均 RequireAuth）保护。
- 独立 curl 复跑：未登录 `GET /admin/mcp.php` → 302 跳登录；登录后 → 200；缺 CSRF 调 ajax → `{code:-1,msg:"CSRF校验失败..."}`；带 CSRF ensure → 200 返回 `sk-`+48hex；regenerate 后**旧 key 对 REST 与 /mcp 均 401、新 key 200**。

### AC-6 共享逻辑层与零重复 — PASS（3/3）

**证据：**

- grep 确认 [internal/service](file:///Volumes/data/lylm_go/internal/service) 全目录**无 `gin-gonic/gin`、无 `html/template` 导入**；仅依赖 models/site/config/fetch 等。
- admin/api/mcp 三处 handler 均调用 service，强类型入参/出参 + 哨兵错误（[internal/service/errors.go](file:///Volumes/data/lylm_go/internal/service/errors.go)）。
- REST 与 MCP 共用同一 Deps 实例与同一 service 对象（router.go 121–123：NewDeps → RegisterV1 + mcp.Mount）。
- `go vet ./...` 通过，无循环依赖；依赖方向符合 `router → admin/api/mcp → service → site/models → database/config`。

### AC-7 现有后台行为不变 — PASS（3/3）

**证据：**

- 阶段 2 的 14 个提交（161affd..fca353d）对前端资产零改动：未触 `template/`、`assets/`、`site/static/`；web 侧仅新增 [web/admin/mcp.html](file:///Volumes/data/lylm_go/web/admin/mcp.html) 与 head.html 菜单项（+1 行）。
- 唯一前台代码改动 [internal/pages/apply.go](file:///Volumes/data/lylm_go/internal/pages/apply.go)：申请落库与免审核路径改为复用 `ApplyService.Create/SetStatus`。对照原 PHP [function.php#L770-L827](file:///Volumes/data/lylm_go/lylme_spage/include/function.php#L770-L827) 分析：终态等价（apply_status=1、链接 link_order=MAX(id)+1、link_status=1），对外 JSON（200/-5）与验证码/限流响应不变。
- 独立 curl 回归：`/`、`/about/`、`/apply/`、`/sitemap.xml` 均 200；`.php` 源码屏蔽有效：`GET /template/default/index.php` → 404；全部 `.php` 兼容路由仍在 admin Register 注册。
- admin 现有 AJAX `{code,msg}` 与 HTML 片段输出在各 handler 薄化时保持原样（tasks T4–T10 证据 + admin 包测试通过）。

### AC-8 构建与自动化测试 — PASS（3/3）

**证据（评审中独立复跑，Go 1.26.2）：**

- `go build ./...` → BUILD_OK
- `go vet ./...` → VET_OK
- `go test ./... -count=1` → 全部 ok：admin 5.7s、api 4.1s、captcha 1.3s、fetch 6.9s、mcp 11.9s、service 10.4s、stats 13.2s、theme 7.9s。
- 测试非空壳：service 覆盖链接查重（含尾斜杠归一）、审核通过写链接+幂等、密钥生成格式/随机性/Ensure 幂等/Regenerate 旧失效、密码哈希；auth 覆盖 4 类坏凭证 401 + 429；API 与 MCP 集成测试覆盖全端点成功路径与失败路径。

### AC-9 接口契约质量（rubric） — PASS（2.7/3，阈值 ≥ 2.5）

**字段级一致性抽查（≥5 端点深度核对）：**

1. `POST/PUT /groups`：doc 约束 name 必填≤30、icon≤1000、pwd≥0、status 枚举 0/1、ids 1–200 正整数 — 与 [group_handler.go#L23-L37](file:///Volumes/data/lylm_go/internal/api/group_handler.go#L23-L37) binding tag 完全一致。
2. `GET /links`：doc 的 page/page_size 默认值（1/10）、非法静默回退、page_size>100 收敛 100、keyword 非空忽略 group_id — 与 [link_handler.go#L85-L148](file:///Volumes/data/lylm_go/internal/api/link_handler.go#L85-L148) 一致。
3. `POST /links`：8 个字段的必填/最大长度（255/255/32/2000/255/512）与 409 查重（含已存在 id、尾斜杠归一）— 与 [link_handler.go#L36-L45](file:///Volumes/data/lylm_go/internal/api/link_handler.go#L36-L45)、[link_handler.go#L174-L195](file:///Volumes/data/lylm_go/internal/api/link_handler.go#L174-L195) 一致。
4. `GET /apply`：doc 明确区分「参数绑定非法 → 400」（与 /links 静默回退相反），status 枚举 0/1/2、page_size 1–100 — 与 [apply_handler.go#L28-L32](file:///Volumes/data/lylm_go/internal/api/apply_handler.go#L28-L32) 一致；批量删除 ids 无上限与 doc 一致。
5. `GET/PUT /settings`、`/settings/system`、`/license`：敏感键黑名单、yan 仅恰好 `"true"` 为真、debug 与 hostmode 枚举 — 与 [settings_handler.go#L13-L16](file:///Volumes/data/lylm_go/internal/api/settings_handler.go#L13-L16)、[settings_handler.go#L208-L212](file:///Volumes/data/lylm_go/internal/api/settings_handler.go#L208-L212)、[settings_handler.go#L261-L265](file:///Volumes/data/lylm_go/internal/api/settings_handler.go#L261-L265) 一致。
6. 错误码映射：400（ErrInvalidInput/ErrWrongPassword/ErrNoChange）、404（各 NotFound）、409（查重，handler 显式）、500（default「internal error」，含 ErrLinkHeadUnreachable）— 与 [errors.go#L13-L30](file:///Volumes/data/lylm_go/internal/api/errors.go#L13-L30) 及文档描述一致。
7. MCP §4.5 总表 56 行工具名/参数摘要与 tools_*.go 核对一致（sou_create 九参数逐一吻合）。

扣分点：dashboard 文档与 spec 覆盖项不一致（F-1，-0.2）；sou_update 参数描述与代码不符（F-2，-0.1）。

### AC-10 重构工程质量（rubric） — PASS（2.8/3，阈值 ≥ 2.5）

- 分层清晰：业务规则集中于 service，三处入口均为薄适配；错误映射、响应信封、分页结构、参数解析辅助全面统一。
- 可测试性强：service/api/mcp/stats 均有实质断言测试；评审独立复跑全绿。
- 零回归：前后台路由与页面经 curl 与测试双重验证（见 AC-7）。
- 扣分点：dashboard 统计缺口（F-1，-0.1）；admin 包保留少量展示/会话辅助函数（F-3，-0.1）。

---

## Findings

### F-1（Minor，已修复）GET /api/v1/dashboard 缺少 spec 要求的「近 7 天访问统计」

- 位置：
  - 出参结构 [internal/api/dashboard_handler.go#L11-L18](file:///Volumes/data/lylm_go/internal/api/dashboard_handler.go#L11-L18)（仅 links/groups/apply_rows/app_version/db_version/build 六字段）
  - [internal/service/dashboard.go#L8-L18](file:///Volumes/data/lylm_go/internal/service/dashboard.go#L8-L18)（注释明确以「service 不依赖 stats 包」排除统计）
  - 文档 [docs/api-v1.md#L2488-L2494](file:///Volumes/data/lylm_go/docs/api-v1.md#L2488-L2494)（按六字段成文）
- 问题：spec「REST API」覆盖范围第 1 项（spec.md 第 69 行）明确要求仪表盘含「近 7 天访问统计」，admin HTML 页也确实通过 `h.Stats.Summary(7)`（[internal/admin/dashboard.go#L39-L41](file:///Volumes/data/lylm_go/internal/admin/dashboard.go#L39-L41)）展示该数据；REST 端点未提供，外部程序无法经 API 获得同口径数据。
- 修复建议：二选一——
  1. 在 Deps 中注入一个 stats 只读接口（避免 service 直接依赖 stats 包），dashboard data 增加 `visit_7d`（或等价结构），同步更新文档与测试；
  2. 若确认该数据不属于「管理操作」契约，则在 spec/文档中显式记录此排除决策并关闭本项。
- **Resolution**：采用方案 1：Deps 新增 `Stats *stats.Store`；REST `GET /dashboard` 与 MCP `dashboard_summary` 均增加 `visits`（today/yesterday/total/today_ip/total_ip/month/labels/pv/ip，无 stats 时数组归一为空）；新增 TestGetDashboardWithVisits；docs/api/dashboard.md 与 docs/api-v1.md 已同步。

### F-2（Nit，已修复）MCP `sou_update` 参数 `st` 的描述与实际默认行为相反

- 位置：[internal/mcp/tools_sou.go#L76](file:///Volumes/data/lylm_go/internal/mcp/tools_sou.go#L76)
- 问题：参数描述为「是否启用：true 启用，false 停用；**不传默认启用**」，但处理器在缺省时传 false（[tools_sou.go#L100](file:///Volumes/data/lylm_go/internal/mcp/tools_sou.go#L100)），经 `SouService.Update` 保存 `sou_st=0`（停用，[service/sou.go#L92](file:///Volumes/data/lylm_go/internal/service/sou.go#L92)）。描述会误导调用方（含 LLM）省略该参数。同工具 create 路径描述「不传默认 false」是正确的。
- 修复建议：将描述改为「不传默认停用（false）；修改时如需保持启用请显式传 true」。
- **Resolution**：描述已改为「不传默认 false（停用）」。

### F-3（Nit）admin 包保留少量展示/会话辅助逻辑

- 位置：[internal/admin/handler.go](file:///Volumes/data/lylm_go/internal/admin/handler.go)（adminSign、truncateRunes、stripTags、htmlEscape）、[internal/admin/links.go](file:///Volumes/data/lylm_go/internal/admin/links.go)（allGroupsPwds 直接用 store.DB() 查询）
- 问题：这些函数属于会话签名与展示适配，非业务规则残留，不违反 AC-6；但与「零业务规则残留」的洁癖目标相比有少量边界模糊。
- 修复建议：无需本期处理；后续可将展示辅助收敛到独立 presentation/util 包，admin 页面的只读查询保持直接访问即可（不构成业务规则）。

---

## 已核查但未构成问题的事项

- 裸 SQL：全仓仅 2 处 `Exec`——种子数据执行（[internal/database/seed.go#L48](file:///Volumes/data/lylm_go/internal/database/seed.go#L48)，AGENTS.md 允许）与安装向导建库（[internal/install/handler.go#L312](file:///Volumes/data/lylm_go/internal/install/handler.go#L312)）；后者库名经 `^[A-Za-z0-9_]+$` 白名单校验（handler.go 第 195、405 行）、charset 取自服务端配置，无注入风险。
- GET /settings 敏感字段：`admin_pwd`、`api_key` 黑名单剔除（代码 + 测试双重确认）；GET /apikey 仅回掩码；密钥明文仅出现在登录态后台页与 regenerate 响应中，grep 未发现密钥进日志。
- SSRF：抓取链路保留 dialer.Control 解析后 IP 二次校验（防 DNS rebinding）、禁内网/保留地址、手动重定向限跳。
- 主题切换的有意收紧（Switch 现校验主题存在性，tasks T9 已记录）：避免配置污染且前台无回归。

## 复核建议

1. 处理 F-1：补齐 dashboard 近 7 天统计字段，或在 spec 中显式豁免。
2. 顺手修正 F-2 的参数描述。
3. 建议增加一项轻量契约守护：CI 中启动服务后自动比对 `tools/list` 名称集合与 docs §4.5 总表，防止后续文档/代码漂移。
4. F-3 可在后续重构窗口统一收敛，不阻断本期验收。
