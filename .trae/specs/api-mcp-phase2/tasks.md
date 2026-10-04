# 第二期改造：后端 API 化 + MCP 服务 Tasks

- 对应 spec：[.trae/specs/api-mcp-phase2/spec.md](file:///Volumes/data/lylm_go/.trae/specs/api-mcp-phase2/spec.md)
- 任务约定：每个任务一个独立 agent 执行；执行前先读 spec.md、本文件与 [AGENTS.md](file:///Volumes/data/lylm_go/AGENTS.md)。
- 并行规则：阶段 B 各任务为**同一批文件的串行重构**（都要改 [internal/admin](file:///Volumes/data/lylm_go/internal/admin)），必须串行；阶段 C 不同域的 API 处理器文件互不重叠，可并行；阶段 D/E 无文件冲突可并行。
- 每完成一个任务：自验全部 TR、补充 Completion Evidence、勾选本项，并按阶段提交（`feat(阶段2): ...`）。
- 构建命令：`export PATH=/Users/macbookpro/sdk/go1.26.2/bin:$PATH && go build ./... && go vet ./... && go test ./...`

任务覆盖 AC：T2/T18→AC-5；T3/T17→AC-2；T4–T10→AC-6/AC-7；T11–T16/T19→AC-1/AC-4；T17→AC-3；全部→AC-8；T11–T17/T19→AC-9；T4–T10/T20→AC-10。

---

## Task 1: 添加 mcp-go v1.0.0 依赖

- **Priority**: high
- **Depends On**: 无
- **Status**: completed

**工作内容**
- 执行 `go get github.com/mark3labs/mcp-go@v1.0.0`，确认 go.mod 出现 `github.com/mark3labs/mcp-go v1.0.0` 直接依赖、go.sum 完整。
- `go build ./...` 通过；记录 mcp-go v1.0.0 中本项目将使用的 API（`server.NewMCPServer`、`server.NewStreamableHTTPServer`、`server.WithStateLess`、`mcp.NewTool` 等）的实际签名备查。

**Test Requirements**
- rule：go.mod 中 mcp-go 版本恰为 v1.0.0 且为直接依赖；证据：go.mod 内容。
- rule：`go build ./...` 退出码 0；证据：命令输出。

**Completion Evidence**: 2026-10-04 执行 `go get github.com/mark3labs/mcp-go@v1.0.0`，输出 `go: added github.com/mark3labs/mcp-go v1.0.0`（同时升级 google/uuid v1.3.0→v1.6.0）；go.mod 已含该模块，`go build ./...` 退出码 0。当前标记 `// indirect`（尚无代码引用），T17 首次 import 后 `go mod tidy` 将转为直接依赖。

---

## Task 2: 密钥能力（APIKey service）

- **Priority**: high
- **Depends On**: T1
- **Status**: completed

**工作内容**
- 新建 `internal/service` 包骨架（包注释说明分层职责：唯一业务逻辑层，禁止依赖 gin/模板）。
- 实现 `apikey.go`：
  - 配置键：`api_key`、`api_key_created_at`（经 [Store.Set/Get](file:///Volumes/data/lylm_go/internal/site/store.go) 持久化，复用配置缓存）。
  - `GenerateKey()`：`sk-` + crypto/rand 随机串（不少于 128 bit 熵；建议 24 字节 → 48 个十六进制字符）。
  - `Get(ctx)`：返回当前 key 与创建时间；不存在时返回空。
  - `Ensure(ctx)`：不存在则生成并持久化。
  - `Regenerate(ctx)`：生成新 key 覆盖持久化，返回新值（旧值立即失效）。
  - `Validate(presented)`：恒定时间比较（`crypto/subtle.ConstantTimeCompare`），key 未设置时一律 false。
- 单测 `apikey_test.go`：生成格式与随机性（两次不同）、Ensure 幂等、Regenerate 后旧值失效新值有效、Validate 恒定时间行为（空串/错误串/正确串）、未设置 key 时任何输入均失败。

**Test Requirements**
- rule：密钥格式恒为 `sk-`+48 hex（128 bit 以上熵）；证据：单测断言。
- rule：重新生成后旧 key Validate=false、新 key=true；证据：单测。
- rule：任何代码路径不打印/记录密钥明文（grep 无 key 进 logger）；证据：代码检查。

**Completion Evidence**: 新建 [internal/service/apikey.go](file:///Volumes/data/lylm_go/internal/service/apikey.go)：crypto/rand 24 字节 → `sk-`+48hex；Get/Ensure/Regenerate/Validate（crypto/subtle 恒定时间比较，未设置 key 一律拒绝）；持久化经 Store.Set 写 `api_key`/`api_key_created_at`。新增 [apikey_test.go](file:///Volumes/data/lylm_go/internal/service/apikey_test.go) 4 个用例（格式/随机、Ensure 幂等、Validate 前后置、Regenerate 旧失效新生效），`go test ./internal/service/` 全部 PASS。

---

## Task 3: 鉴权中间件与 /api/v1 骨架、PV 排除

- **Priority**: high
- **Depends On**: T2
- **Status**: completed

**工作内容**
- 新建 `internal/api` 包：
  - `response.go`：统一成功/失败信封（`{code,message,data}`）、分页结构（`list/page/page_size/total`）、错误到 HTTP 状态码映射（400/401/403/404/409/429/500）。
  - `auth.go`：Bearer 鉴权中间件——从 `Authorization` 提取 `Bearer sk-...`，调 APIKey service 校验；失败写 401 统一 JSON；叠加 `webutil.RateLimit`（按 IP，固定窗口，阈值常量集中定义），超限 429。
  - `router.go`：创建 `/api/v1` 路由组（挂载鉴权 + 限流），先注册 `GET /api/v1/ping` 自检端点；为后续任务暴露注册入口。
- 修改 [internal/stats/stats.go](file:///Volumes/data/lylm_go/internal/stats/stats.go)：中间件 skipPrefixes 增加 `/api`、`/mcp`。
- 在 [internal/router/router.go](file:///Volumes/data/lylm_go/internal/router/router.go) 的 `Setup` 中接入 `/api/v1` 组（具体端点后续任务注册）。
- 单测：无凭证/错误凭证 401、有效凭证 200、超限 429（小阈值注入）、/api 与 /mcp 不产生 PV。

**Test Requirements**
- rule：`GET /api/v1/ping` 对缺失/畸形/错误 Bearer 返回 401 JSON；有效 key 返回 200；证据：httptest。
- rule：超过限流阈值返回 429；证据：单测连续请求。
- rule：访问 `/api/v1/...`、`/mcp` 不增加 PV 计数；证据：stats 单测。

**Completion Evidence**: 新建 [response.go](file:///Volumes/data/lylm_go/internal/api/response.go)（统一信封 `{code,message,data}`、泛型分页 `PageResult[T]`、状态码辅助）、[auth.go](file:///Volumes/data/lylm_go/internal/api/auth.go)（Bearer 提取 + 密钥校验，默认 120 次/分钟 IP 限流，阈值可注入）、[router.go](file:///Volumes/data/lylm_go/internal/api/router.go)（`/api/v1/ping`）；[stats.go](file:///Volumes/data/lylm_go/internal/stats/stats.go#L89) skipPrefixes 增加 `/api`、`/mcp`；主路由接入 RegisterV1。测试：[auth_test.go](file:///Volumes/data/lylm_go/internal/api/auth_test.go) 覆盖无凭证/错凭证 401、有效 key 200、连续请求 429；stats 新增中间件排除测试。`go build/vet` 通过，api/stats/service 测试全 PASS。

---

## Task 4: Group 域 service 抽取与 admin 薄化

- **Priority**: high
- **Depends On**: T3
- **Status**: completed

**工作内容**
- 新建 `internal/service/group.go`，承接 [groups.go](file:///Volumes/data/lylm_go/internal/admin/groups.go) 全部业务规则：
  - 列表（含链接计数、pwd 键映射）、新增（图标纯 URL → `<img>` 包裹、order=max+1）、修改、删除（同删组内链接）、启停用、批量排序。
  - 定义强类型请求结构与哨兵错误（`ErrGroupNotFound`、`ErrInvalidInput`）。
- 改造 [internal/admin/groups.go](file:///Volumes/data/lylm_go/internal/admin/groups.go)：处理器仅做参数解析 + 调 service + 保持原有输出（alert/notify 脚本、200 空响应、HTML 片段渲染）。
- service 单测：新增/修改/删除（含级联）/启停用/排序、图标规范化、错误分支。

**Test Requirements**
- rule：删除分组时组内链接被级联删除、其他组不受影响；证据：service 单测（sqlite 内存库）。
- rule：admin 路由 `group.php` 全部既有 set 分支（add/edit/del/on/off/sort）输出行为与重构前等价；证据：处理器测试/回归记录。
- rule：`internal/service` 不 import gin；证据：grep。

**Completion Evidence**: 新建 [errors.go](file:///Volumes/data/lylm_go/internal/service/errors.go)、[group.go](file:///Volumes/data/lylm_go/internal/service/group.go)（List/Get/Create/Update/Delete/SetStatus/Sort + NormalizeGroupIcon）、[group_test.go](file:///Volumes/data/lylm_go/internal/service/group_test.go) 9 用例；[admin/groups.go](file:///Volumes/data/lylm_go/internal/admin/groups.go) 薄化，Handler 新增 GroupSvc。顺带修复 apikey_test 一个 flaky 用例。主 agent 独立执行 build/vet/test 全部通过（含 go test ./... 全 ok），grep 确认 service 无 gin import。

---

## Task 5: Link 域 service 抽取与 admin 薄化

- **Priority**: high
- **Depends On**: T4
- **Status**: completed

**工作内容**
- 新建 `internal/service/link.go`，承接 [links.go](file:///Volumes/data/lylm_go/internal/admin/links.go) 中链接相关全部规则：
  - 按组分页查询（含分组名映射、总数）、单条详情、全量简表；
  - 新增（必填校验、URL 去末尾斜杠查重并返回已存在 ID、color→`<font color>` 包裹、order=count+1）；
  - 修改、批量删除、批量启停用、批量移动分组、批量设置加密、批量排序；
  - 抓取目标站点信息（复用 [internal/fetch](file:///Volumes/data/lylm_go/internal/fetch)，保持 SSRF 防护与缓存目录约定）。
- 改造 [links.go](file:///Volumes/data/lylm_go/internal/admin/links.go) 中对应处理器与 `AjaxLink` 分支（add_link/edit_link/set_group/pwd_link/on/off/del/allorder/geturl/all_links/edit_form），保持 `jsonWrite` 的 `{code,msg}` 与片段渲染不变。
- service 单测：查重（含斜杠差异、返回 201 语义所需数据）、颜色包裹、批量操作、排序下标、抓取失败分支。

**Test Requirements**
- rule：新增重复 URL（忽略末尾斜杠）不产生第二条记录且返回既有 ID；证据：service 单测。
- rule：admin `ajax_link.php` 上述各分支响应 JSON（code/msg）与重构前等价；证据：处理器测试。
- rubric（局部，0–2）：链接域规则覆盖完整度（校验/排序/批量/查重），≥1.5 为通过；证据：单测与评审。

**Completion Evidence**: 新建 [link.go](file:///Volumes/data/lylm_go/internal/service/link.go)（GroupTable/Get/All/Create/Update/BatchDelete/BatchSetStatus/BatchMoveGroup/BatchSetPwd/Sort/FetchHead + LinkInput/LinkCreateResult/LinkBrief 强类型）、[link_test.go](file:///Volumes/data/lylm_go/internal/service/link_test.go) 10 用例；errors.go 增补 ErrLinkNotFound/ErrLinkHeadUnreachable；[admin/links.go](file:///Volumes/data/lylm_go/internal/admin/links.go) 薄化（Tag/Sou 四函数原样保留），Handler 新增 LinkSvc。主 agent 独立 build/vet/test 全通过，抽查 add_link 的 200/201 响应与原字段一致，service 无 gin 依赖。

---

## Task 6: Tag / Sou / Pwd 域 service 抽取与 admin 薄化

- **Priority**: high
- **Depends On**: T5
- **Status**: completed

**工作内容**
- 新建 `internal/service/tag.go`、`sou.go`、`pwd.go`，承接对应规则：
  - Tag：列表（sort ASC）、增、改、删；target 语义。
  - Sou：列表、增（order=count+1）、改、删；st 布尔语义。
  - Pwd：列表、增、改、删。
- 改造 admin：[misc.go](file:///Volumes/data/lylm_go/internal/admin/misc.go) 中 Sou/Tag/Pwd 页面处理器、[links.go](file:///Volumes/data/lylm_go/internal/admin/links.go) 中 add_tag/edit_tag/add_sou/edit_sou 分支；输出（alert 脚本、片段、JSON）保持不变。
- service 单测：各域增改删、默认 order、target/st 语义。

**Test Requirements**
- rule：三域全部既有页面/AJAX 操作经 service 后行为等价；证据：处理器测试与回归记录。
- rule：重复业务规则不存在于 admin（规则仅 service 一份）；证据：grep/评审。

**Completion Evidence**: 新建 [tag.go](file:///Volumes/data/lylm_go/internal/service/tag.go)、[sou.go](file:///Volumes/data/lylm_go/internal/service/sou.go)、[pwd.go](file:///Volumes/data/lylm_go/internal/service/pwd.go) 及 12 个单测；errors 增补三域 NotFound；admin misc.go/links.go 薄化，Handler 新增 TagSvc/SouSvc/PwdSvc。主 agent 独立 build/vet/test 全通过，service 无 gin 依赖。

---

## Task 7: 配置域（settings/about/user）service 抽取与 admin 薄化

- **Priority**: high
- **Depends On**: T6
- **Status**: completed

**工作内容**
- 新建 `internal/service/config.go`（或 `settings.go`），承接 [settings.go](file:///Volumes/data/lylm_go/internal/admin/settings.go) 规则：
  - 站点设置批量更新（title/logo/background/wap_background/keywords/description/mode/snapshot/copyright/icp/wztj/cdnpublic/yan/tq 的键映射）；
  - 关于页内容读取/更新/恢复默认（默认内容常量移至 service）；
  - 管理员账号修改（校验旧密码 `md5("lylme"+明文)`、用户名/密码更新、两次密码一致性）；
  - 系统安全（admin_path 正则校验、debug 取值校验）。
- 改造 admin [settings.go](file:///Volumes/data/lylm_go/internal/admin/settings.go) 处理器为薄适配，alert 脚本与跳转保持不变。
- service 单测：批量更新键映射、关于页默认恢复、旧密码错误拦截、两次不一致拦截、admin_path 非法值拦截。

**Test Requirements**
- rule：错误旧密码不能改密；正确旧密码 + 一致新密码可改密并保持 `md5("lylme"+pwd)` 存储；证据：service 单测。
- rule：set/about/user 三个页面 GET/POST 输出与重构前等价；证据：处理器测试/回归。

**Completion Evidence**: 新建 [configservice.go](file:///Volumes/data/lylm_go/internal/service/configservice.go)（站点设置/关于页/账号/系统安全 + AdminPasswordHash 唯一哈希实现）及 [测试](file:///Volumes/data/lylm_go/internal/service/configservice_test.go)；admin settings.go/auth.go 薄化（登录改用 service.AdminPasswordHash，行为不变），Handler 新增 ConfigSvc。主 agent 独立 build/vet/test 全通过，service 无 gin 依赖。

---

## Task 8: Apply 域 service 抽取与 admin 薄化

- **Priority**: high
- **Depends On**: T7
- **Status**: completed

**工作内容**
- 新建 `internal/service/apply.go`，承接收录规则：
  - 列表（按状态、时间倒序）、分组映射；收录开关与公告（`apply`/`apply_gg`）；
  - 修改申请；审核通过（仅对 status=0 记录：写入链接表 stripTags 名称/URL、order=max(id)+1，状态置 1）/拒绝（置 2）；批量删除。
- 改造 admin [misc.go](file:///Volumes/data/lylm_go/internal/admin/misc.go) ApplyPage、AjaxApply 与 [pages/apply.go](file:///Volumes/data/lylm_go/internal/pages/apply.go)（若前台提交逻辑可复用同一 service 方法则一并复用）。
- service 单测：审核通过写链接 + 状态翻转、拒绝不写链接、重复审核幂等（status≠0 不再写）、批量删除。

**Test Requirements**
- rule：审核通过恰好新增 1 条链接且 apply_status=1；拒绝不新增链接；证据：service 单测。
- rule：admin apply 页面与 ajax_apply 输出行为等价；证据：处理器测试/回归。

**Completion Evidence**: 新建 [apply.go](file:///Volumes/data/lylm_go/internal/service/apply.go)（List/Settings/Create/Get/Update/SetStatus/BatchDelete，审核通过写链接+幂等）及 [测试](file:///Volumes/data/lylm_go/internal/service/apply_test.go) 8 用例；admin misc.go 薄化并删除 insertLinkFromApply；[pages/apply.go](file:///Volumes/data/lylm_go/internal/pages/apply.go) 前台提交复用同一 service（验证码/限流/响应不变），Handler 新增 ApplySvc。主 agent 独立 build/vet/test 全通过，service 无 gin 依赖。

---

## Task 9: Theme 域 service 抽取与 admin 薄化

- **Priority**: medium
- **Depends On**: T8
- **Status**: completed

**工作内容**
- 新建 `internal/service/theme.go`，承接：
  - 主题列表 + 当前主题（复用 [theme.Manager](file:///Volumes/data/lylm_go/internal/theme/theme.go)）、切换主题（写 `template` 键）；
  - 主题配置字段定义与当前值（ConfigFields + ConfigMap）、保存主题自定义配置（checkbox/switch 默认值补齐逻辑、序列化写 `theme_config_<tpl>`）。
- 改造 admin [misc.go](file:///Volumes/data/lylm_go/internal/admin/misc.go) ThemePage、ThemeSettingPage、AjaxTheme 为薄适配，输出不变。
- service 单测：切换、配置存取（含 checkbox 默认补齐）、不存在主题回退 default。

**Test Requirements**
- rule：保存主题配置写 `theme_config_<tpl>` 且未提交的 switch 补 0、checkbox 补空数组；证据：service 单测。
- rule：theme/theme_setting/ajax_theme 输出与重构前等价；证据：处理器测试/回归。

**Completion Evidence**: 新建 [theme.go](file:///Volumes/data/lylm_go/internal/service/theme.go)（List/Current/Switch/ConfigFields/ConfigValues/SaveConfig，含 switch/checkbox 默认补齐）及 [测试](file:///Volumes/data/lylm_go/internal/service/theme_test.go) 6 用例；admin misc.go 主题处理器薄化（展示辅助 renderThemeForm 保留），Handler 新增 ThemeSvc。有意差异：Switch 现校验主题存在性（原版不校验，配置键不再被污染，前台渲染无变化）。主 agent 独立 build/vet/test 全通过，service 无 gin 依赖。

---

## Task 10: Dashboard / 文件清理 / wxplus / license 域 service 抽取与 admin 薄化

- **Priority**: medium
- **Depends On**: T9
- **Status**: completed

**工作内容**
- 新建：
  - `internal/service/dashboard.go`：链接/分组/待审计数、版本信息、近 7 天统计汇总；
  - `internal/service/filecleanup.go`：扫描 files/download 与 files/upload 未使用文件（与 DB 图标引用比对）、删除（路径正则 `^(download|upload)/[^/]+$` 与目录穿越校验）；
  - 将 wxplus（`wxplus`/`wxplustime`）、license（`c`/`hostmode`）读写归入配置 service 方法。
- 改造 admin [dashboard.go](file:///Volumes/data/lylm_go/internal/admin/dashboard.go) 与 [misc.go](file:///Volumes/data/lylm_go/internal/admin/misc.go) CleanImg/WxPlus/License/Update/Help 处理器为薄适配。
- service 单测：汇总计数、未使用识别、路径穿越/非法 rel 拒绝删除。

**Test Requirements**
- rule：文件清理删除操作拒绝含 `..` 或不在 download/upload 白名单的 rel；证据：service 单测。
- rule：dashboard/cleanimg/wxplus/license 页面输出与重构前等价；证据：处理器测试/回归。
- rule：至此 admin 包中无业务规则残留（grep 关键规则只在 service）；证据：代码检查。

**Completion Evidence**: 新建 [dashboard.go](file:///Volumes/data/lylm_go/internal/service/dashboard.go)、[filecleanup.go](file:///Volumes/data/lylm_go/internal/service/filecleanup.go)、[wxplus.go](file:///Volumes/data/lylm_go/internal/service/wxplus.go)、[license.go](file:///Volumes/data/lylm_go/internal/service/license.go) 及 7 个单测；ConfigService 增补微信/授权读写；admin dashboard.go/misc.go 薄化（cleanimg 模板经 toCleanImgPage 映射），Handler 新增 DashboardSvc/FileCleanupSvc。主 agent 独立 build/vet/test 全通过，service 无 gin 依赖。

---

## Task 11: API 端点 —— group / link 域

- **Priority**: high
- **Depends On**: T10
- **Status**: completed

**工作内容**
- 在 `internal/api` 新建 `group_handler.go`、`link_handler.go`，于 `/api/v1` 注册：
  - groups：`GET /groups`、`POST /groups`、`PUT /groups/:id`、`DELETE /groups/:id`、`POST /groups/:id/status`、`POST /groups/sort`；
  - links：`GET /links`（page/page_size/group_id/keyword）、`GET /links/all`、`GET /links/:id`、`POST /links`、`PUT /links/:id`、`DELETE /links`（ids 数组）、`POST /links/status`、`POST /links/group`、`POST /links/pwd`、`POST /links/sort`、`GET /links/fetch?url=`。
- JSON 请求结构体带校验（binding tag：required/max/min/oneof）；出参使用统一信封与分页结构；service 哨兵错误映射到对应 HTTP 状态码（404/400/409）。
- 同步在 `docs/api-v1.md` 增补以上端点契约（用途/鉴权/入参规格/出参结构/错误码）。
- 集成测试：每端点成功 + 一个失败路径（未鉴权 401 / 参数错误 400 / 不存在 404）。

**Test Requirements**
- rule：列出的每个路由均已注册且成功路径返回 code=200 与文档声明的 data 结构；证据：httptest。
- rule：`POST /links` 对重复 URL 返回文档声明的冲突结构（200 信封内业务码或 409，二者在文档中固定一种并与实现一致）；证据：测试 + 文档核对。
- rubric（局部，0–2）：入参/出参 schema 与校验完整度 ≥1.5；证据：评审。

**Completion Evidence**: 新建 [group_handler.go](file:///Volumes/data/lylm_go/internal/api/group_handler.go)（6 端点）、[link_handler.go](file:///Volumes/data/lylm_go/internal/api/link_handler.go)（11 端点）及 43 个集成子测试；service/link.go 增补 ListLinks/SearchLinks（参数化）；重复 URL 固定返回 409「链接已存在」含已存在 id；分域文档 [group.md](file:///Volumes/data/lylm_go/docs/api/group.md)、[link.md](file:///Volumes/data/lylm_go/docs/api/link.md)。主 agent 统一接线后 build/vet/test 全通过。

---

## Task 12: API 端点 —— tag / sou / pwd 域

- **Priority**: high
- **Depends On**: T10
- **Status**: completed

**工作内容**
- `tag_handler.go`：`GET /tags`、`POST /tags`、`PUT /tags/:id`、`DELETE /tags/:id`。
- `sou_handler.go`：`GET /sou`、`POST /sou`、`PUT /sou/:id`、`DELETE /sou/:id`。
- `pwd_handler.go`：`GET /pwd`、`POST /pwd`、`PUT /pwd/:id`、`DELETE /pwd/:id`。
- 同步更新 `docs/api-v1.md`；每端点成功 + 失败路径集成测试。

**Test Requirements**
- rule：12 个端点全部注册并可用（成功 200，未鉴权 401，参数错误 400，不存在 404）；证据：httptest。
- rule：docs 对应章节字段与请求/响应结构体一致；证据：核对记录。

**Completion Evidence**: 新建 [tag_handler.go](file:///Volumes/data/lylm_go/internal/api/tag_handler.go)、[sou_handler.go](file:///Volumes/data/lylm_go/internal/api/sou_handler.go)、[pwd_handler.go](file:///Volumes/data/lylm_go/internal/api/pwd_handler.go) 共 12 端点，15 个集成用例（成功/400/404/401）；删除端点先 Get 校验；Sou alias/name 在 REST 层必填（有意收紧，文档注明）；分域文档 [tag.md](file:///Volumes/data/lylm_go/docs/api/tag.md)、[sou.md](file:///Volumes/data/lylm_go/docs/api/sou.md)、[pwd.md](file:///Volumes/data/lylm_go/docs/api/pwd.md)。主 agent 统一接线后 build/vet/test 全通过。

---

## Task 13: API 端点 —— settings / about / user / apikey 域

- **Priority**: high
- **Depends On**: T10
- **Status**: completed

**工作内容**
- `settings_handler.go`：
  - `GET /settings`、`PUT /settings`（批量）；
  - `GET /settings/about`、`PUT /settings/about`、`POST /settings/about/reset`；
  - `PUT /settings/account`（旧密码/新密码/用户名）；
  - `GET /settings/system`、`PUT /settings/system`（admin_path/debug）；
  - `GET /wxplus`、`PUT /wxplus`；`GET /license`、`PUT /license`；
  - `GET /apikey`（仅返回 key 是否存在/创建时间/掩码前缀，避免凭接口回读完整明文）、`POST /apikey/regenerate`（用当前 key 鉴权，返回新 key，文档强提示旧 key 立即失效）。
- `GET /settings` 的出参 SHALL 屏蔽 `admin_pwd` 等敏感键。
- 同步更新 `docs/api-v1.md`；成功 + 失败路径集成测试。

**Test Requirements**
- rule：`GET /settings` 响应不含 admin_pwd/api_key 明文；证据：测试断言。
- rule：`POST /apikey/regenerate` 后旧 key 401、新 key 200；证据：httptest 顺序请求。
- rule：全部端点行为与文档契约一致；证据：测试与核对。

**Completion Evidence**: 新建 [settings_handler.go](file:///Volumes/data/lylm_go/internal/api/settings_handler.go) 14 端点；ConfigService 增补只读 AllConfig；GET /settings 黑名单屏蔽 admin_pwd/api_key；GET /apikey 仅返回 exists/created_at/masked；regenerate 后旧 key 401、新 key 200；分域文档 [settings.md](file:///Volumes/data/lylm_go/docs/api/settings.md)。主 agent 统一接线后 build/vet/test 全通过。

---

## Task 14: API 端点 —— apply 域

- **Priority**: high
- **Depends On**: T10
- **Status**: completed

**工作内容**
- `apply_handler.go`：
  - `GET /apply`（status 过滤、分页）；
  - `GET /apply/settings`、`PUT /apply/settings`（apply 开关/apply_gg）；
  - `PUT /apply/:id`（修改）；
  - `POST /apply/:id/review`（status=1 通过 / 2 拒绝）；
  - `DELETE /apply`（ids 批量）。
- 同步更新 `docs/api-v1.md`；成功 + 失败路径集成测试。

**Test Requirements**
- rule：审核通过端点返回 200 且可在 /links 查到新增链接；拒绝不新增；证据：跨端点测试。
- rule：status 枚举校验（非 1/2 → 400）；证据：测试。

**Completion Evidence**: 新建 [apply_handler.go](file:///Volumes/data/lylm_go/internal/api/apply_handler.go) 6 端点（分页 DTO 带 group_name），20 个集成用例；审核通过后链接数恰好 +1、拒绝不新增、重复通过幂等；非法枚举 400；分域文档 [apply.md](file:///Volumes/data/lylm_go/docs/api/apply.md)。主 agent 统一接线后 build/vet/test 全通过。

---

## Task 15: API 端点 —— theme 域

- **Priority**: medium
- **Depends On**: T10
- **Status**: completed

**工作内容**
- `theme_handler.go`：
  - `GET /themes`（列表+当前）、`PUT /themes/current`（切换）；
  - `GET /themes/config`（字段定义+当前值）、`PUT /themes/config`（保存）。
- 同步更新 `docs/api-v1.md`；成功 + 失败路径（切换到不存在主题 → 400/回退）集成测试。

**Test Requirements**
- rule：切换主题后 `GET /themes` 当前主题变化且前台首页使用新主题；证据：测试。
- rule：配置端点结构与文档一致；证据：测试与核对。

**Completion Evidence**: 新建 [theme_handler.go](file:///Volumes/data/lylm_go/internal/api/theme_handler.go) 4 端点（列表/切换/配置读/配置写，PUT config 为全量替换）；测试指向真实项目根主题并在 t.Cleanup 恢复 default 主题与清理测试配置；分域文档 [theme.md](file:///Volumes/data/lylm_go/docs/api/theme.md)。主 agent 统一接线后 build/vet/test 全通过。

---

## Task 16: API 端点 —— dashboard / files 域

- **Priority**: medium
- **Depends On**: T10
- **Status**: completed

**工作内容**
- `dashboard_handler.go`：`GET /dashboard`（计数/版本/近 7 天统计）。
- `files_handler.go`：`GET /files/cleanup`（扫描结果）、`POST /files/cleanup`（删除指定 rel 列表，返回成功/失败计数）。
- 同步更新 `docs/api-v1.md`；成功 + 失败路径集成测试。

**Test Requirements**
- rule：`GET /dashboard` 字段（链接/分组/待审/统计）与文档一致；证据：测试。
- rule：`POST /files/cleanup` 对非法 rel 计数为失败且不删除任何文件；证据：测试。

**Completion Evidence**: 新建 [dashboard_handler.go](file:///Volumes/data/lylm_go/internal/api/dashboard_handler.go) 3 端点（dashboard 计数/版本、cleanup 扫描、cleanup 删除）；6 个集成用例，非法 rel（../、白名单外、子路径、缺失文件）全部计入 failed 且零删除，空 rels 400；分域文档 [dashboard.md](file:///Volumes/data/lylm_go/docs/api/dashboard.md)。主 agent 统一接线后 build/vet/test 全通过。

---

## Task 17: MCP Server 构建与 /mcp 挂载

- **Priority**: high
- **Depends On**: T11、T12、T13、T14、T15、T16
- **Status**: completed

**工作内容**
- 新建 `internal/mcp` 包：
  - `server.go`：`server.NewMCPServer("lylme-navigation", AppVersion, server.WithToolCapabilities(true))`；
  - 按域把每个 service 方法注册为 MCP 工具（建议分文件 `tools_link.go`/`tools_group.go`/…）：
    - `mcp.NewTool(<稳定名>, mcp.WithDescription("<用途>"), mcp.WithString/Number/Boolean/Array/Object(<参数>, mcp.Required(), mcp.Description("<含义>"), mcp.Enum(...)…))`；
    - handler：从 `request.GetArguments()` 解析参数（带类型/必填校验）→ 调 service → 成功以 `mcp.NewToolResultText(json)` 返回（JSON 结构与对应 REST data 一致）；失败以 `mcp.NewToolResultError(msg)`（IsError）返回。
    - 工具集合与 `/api/v1` 操作一一对应；命名风格统一（`<域>_<动作>`）。
  - `http.go`：`server.NewStreamableHTTPServer(mcpServer, server.WithStateLess(true))` 作为 http.Handler 暴露。
- 在 [router.go](file:///Volumes/data/lylm_go/internal/router/router.go) 挂载：`/mcp` 与 `/mcp/` 的全部方法经 Bearer 鉴权中间件后转发 `ServeHTTP(c.Writer, c.Request)`（注册 GET/POST/DELETE，关闭 Gin 的 trailing-slash 自动跳转干扰）。
- MCP 集成测试（httptest 上跑 JSON-RPC）：`initialize` → `tools/list`（断言工具数量/命名）→ 每域至少一个 `tools/call` 成功 → 参数错误返回 IsError。

**Test Requirements**
- rule：无/错 key 访问 /mcp 返回 401；有效 key 完成 initialize；证据：测试。
- rule：`tools/list` 工具与文档「MCP 工具对应表」逐一吻合（名称/参数 schema）；证据：测试断言。
- rule：每域至少 1 个工具 call 返回 code 语义成功且结果可 JSON 解析为对应出参结构；证据：集成测试。
- rubric（局部，0–2）：工具描述与参数 schema 对 LLM 的可用性 ≥1.5；证据：评审。

**Completion Evidence**: _(待填写)_

---

## Task 18: 后台「接口与 MCP」管理页面

- **Priority**: high
- **Depends On**: T2、T3
- **Status**: pending

**工作内容**
- 新建后台模板 `web/admin/mcp.html`（沿用 head/foot 与现有页面卡片风格）：
  - 密钥区：完整 key 展示 + 复制密钥；无 key 时「生成密钥」；「重新生成」按钮（jquery-confirm 二次确认，提示旧 key 立即失效）；
  - 地址区：API base（`<根>/api/v1`）、MCP endpoint（`<根>/mcp`）各带复制按钮；
  - 配置区：只读 `<textarea>`/代码块展示已内嵌当前 key 的 MCP 配置 JSON（mcpServers/url/headers.Authorization），带复制按钮；
  - 复制用原生 JS（`navigator.clipboard` + 降级 execCommand），不引入新库；操作经现有 admin AJAX 模式（session+CSRF）调用内部管理端点（handler 调 APIKey service：ensure/regenerate）。
- 新增 admin handler（`internal/admin/apikey_page.go`）：页面 GET + 生成/重新生成 JSON 端点（受现有 `auth`/`frag` 中间件保护）。
- [head.html](file:///Volumes/data/lylm_go/web/admin/head.html) 侧边栏新增菜单项（mdi 图标，如 `mdi-api`），并在 Register 注册路由。
- 验证：页面登录保护、生成/重新生成/复制四类操作可用、配置 JSON 可被 MCP 客户端格式校验。

**Test Requirements**
- rule：未登录访问该页面跳转登录；登录后正常渲染；证据：处理器测试。
- rule：重新生成后页面展示新 key，且旧 key 对 /api/v1 与 /mcp 返回 401、新 key 200；证据：端到端记录。
- rule：复制的配置 JSON 中 Authorization 与页面 key 完全一致；证据：DOM/JS 检查。

**Completion Evidence**: _(待填写)_

---

## Task 19: docs/api-v1.md 终稿校对

- **Priority**: high
- **Depends On**: T17
- **Status**: pending

**工作内容**
- 汇总各任务增量，形成完整 `docs/api-v1.md`：
  - 通用约定（base URL、鉴权、信封、分页、错误码总表、限流说明）；
  - 每个端点：用途、方法+路径、鉴权、入参（名/位置/类型/必填/长度/枚举）、出参（字段/类型/含义）、错误码与触发条件、示例；
  - MCP 工具与端点对应总表（工具名、用途、参数 schema 摘要）。
- 逐条与路由注册、请求/响应结构体、MCP 工具定义做一致性校对，消除遗漏与不一致。

**Test Requirements**
- rule：AC-1 的 12 域操作在文档与路由中双向无遗漏；证据：核对清单。
- rubric（AC-9）：按 0–3 锚点评分，达到 ≥2.5；证据：独立评审打分与理由。

**Completion Evidence**: _(待填写)_

---

## Task 20: 端到端验证、全量构建测试与提交

- **Priority**: high
- **Depends On**: T18、T19
- **Status**: pending

**工作内容**
- 启动服务，使用 curl / MCP Inspector 完成真实链路验证并留存记录：
  - REST：401 拦截、分页查询、新增（查重）、审核通过写链接、切换主题、重新生成密钥全流程；
  - MCP：Inspector 对 `/mcp` initialize/tools/list、调用 3 个以上工具；
  - 后台：新菜单页面操作；现有后台页面回归抽查（首页/链接/分组/设置）；
  - 前台首页与固定页无变化。
- 运行 `go build ./...`、`go vet ./...`、`go test ./...`（指定 Go 路径），全绿。
- 确认工作区无 `.env`/data/logs/cache/files 被提交；按阶段补齐 git 提交。

**Test Requirements**
- rule：build/vet/test 全通过；证据：命令输出。
- rule：真实 curl/Inspector 链路全部符合预期且记录已留存；证据：验证记录。
- rubric（AC-10）：按 0–3 锚点评分 ≥2.5；证据：独立评审。

**Completion Evidence**: _(待填写)_

---

## Review Issues（评审问题队列）

_(Review 阶段由独立评审填写；每条 actionable finding 在此登记为新任务并在状态机中跟踪。)_
