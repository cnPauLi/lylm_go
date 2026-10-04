# 去除全部 PHP 后缀（路由、代码引用、前端、config.php 机制）Spec

## Why

项目已是纯 Go 实现，但路由、后台页面标识、模板链接、JS AJAX 地址、注释与文档中仍残留 PHP 时代的 `.php` 后缀；主题配置还依赖 `config.php`（PHP 数组格式 + Go 手写解析器）。用户要求批量清除所有 PHP 后缀，使项目彻底去 PHP 化。

## 已确认决策（用户拍板）

1. **旧 .php 地址直接切断**：不保留 302 跳转，旧地址返回 404。此决策覆盖 AGENTS.md §7 的「URL 兼容清单」，AGENTS.md 需同步改写。
2. **删除死文件**：template/ 各主题已转换完的 `.php` 参考文件（index.php/footer.php/list.php/foot.php/head.php/go.php/functions.php）与 assets/img/bing.php、cron.php 全部删除。
3. **废除 config.php**：项目唯一配置文件为 `.env`（不新增任何其他配置文件）。主题自定义字段定义迁移进 `theme.ini`（JSON），值仍存数据库（`theme_config_<tpl>`），`.env` 不承载主题字段值。

## What Changes

### 1. 路由去 .php（**BREAKING**：旧 .php 地址 404）

- 后台（internal/admin/handler.go Register）：`/admin/login.php`→`/admin/login`；`/admin/set.php`→`/admin/set`；其余同构：about/user/link/table_link/ajax_link/group/table_group/sou/tag/pwd/apply/ajax_apply/theme/theme_setting/ajax_theme/cleanimg/batch_add/update/wxplus/license/help/mcp/ajax_mcp。`/admin/index.php` 路由删除（`/admin`、`/admin/` 已覆盖）。
- 前台（internal/router/router.go）：`/include/validatecode.php`→`/include/validatecode`；`/include/file.php`→`/include/file`；`/include/go.php`→`/include/go`；`/include/qrcode.php`→`/include/qrcode`；`/apply/index.php` 注册删除（GET/POST `/apply`、`/apply/` 已覆盖，需为 POST 补注册 `/apply`、`/apply/`）；`/apply/wxplus.php`→`/apply/wxplus`；assets 动态拦截 `/img/bing.php`→`/img/bing`、`/img/cron.php`→`/img/cron`。
- 安装向导（internal/install/handler.go）：删除 `base+"/index.php"` GET/POST；新增 `POST base+"/"`（原 POST index.php 表单落到 `/install/`）。
- 静态屏蔽逻辑保留：staticHandler 继续拒绝 `.php` 后缀文件（防御性，注释改写）。

### 2. Go 代码引用同步

- 页面标识：`newData(c, 标题, "xxx.php", 菜单)` → `"xxx"`；后台首页 `"index.php"` → `"index"`。
- 重定向/提示串：`h.Base()+"/login.php"`→`h.Base()+"/login"`；`./login.php`/`./set.php`/`./pwd.php` 等 alert、script 跳转串全部去后缀。
- 生成 HTML/JS 的字符串（misc.go 的 form action、links.go 的下拉 HTML）同步去后缀。
- 用户可见输出：internal/pages/media.go CronImage 报错文案中 `/assets/img/cron.php?key=秘钥` → `/assets/img/cron?key=秘钥`。
- 注释：引用原 PHP 文件名处（如 `site/common.php`、`login.php`、`include/file.php`）统一去后缀或改写措辞。
- 后台首页 AJAX `index.php?check=update`（web/admin/index.html）→ `?check=update`（相对当前 `/admin/`，Dashboard 已按 `check` 参数返回 JSON）。

### 3. 前端模板与 JS 同步

- web/**/*.html：href/action/src 中 `./xxx.php`、`xxx.php?...`、`../include/*.php`、`/include/*.php` 全部去后缀；`.Is "set.php"` 类模板判断与 Go 侧页面标识同步改为无后缀。
- assets/admin/js/{group,link,apply,theme_setting}.js、apply/apply.js、install/templates/js/forms.js、template/dev-theme/js/script.js：AJAX URL 与选择器（`form[action*="xxx.php"]`）去后缀。

### 4. config.php 机制废除 → theme.ini 迁移（**BREAKING**：主题格式变更）

- theme.ini（JSON）新增顶层键 `"theme_config"`：字段对象数组，键为 `type/name/title/description/verify/placeholder/value/enum`，与原 `Field` 结构一一对应。
- `value` 类型：字符串、数字、字符串数组（多选），或 `{"$conf":"site-key"}` 对象（等价原 `$GLOBALS['conf']['xxx']`，解析期代入站点配置值）。
- `enum` 用有序数组表示：`[["0","关闭"],["1","搜索栏下方"],["2","底部"]]`（value,label 二元组数组，保序）。
- internal/theme/config.go：删除 PHP 词法/语法解析器（tokenize/parser/resolveExpr 等），重写为解析 theme.ini 的 `theme_config` JSON → `[]Field`；`Field/EnumOption/DefaultValue/DefaultValueList` 结构保留；`Manager.ConfigFields/ConfigValue/ConfigValues` 对外签名不变。
- 11 个含 config.php 的主题：`$theme_config` 内容迁入各自 theme.ini 后删除 config.php。
- 测试：internal/theme/config_test.go 改为针对 theme.ini 的新测试；internal/service/theme_test.go 同步。

### 5. 删除死文件

- template/*/ 下：index.php、footer.php、list.php、foot.php、head.php、go.php、functions.php。
- assets/img/bing.php、assets/img/cron.php。

### 6. 文档同步

- AGENTS.md：§2 目录说明、§6 主题约定（theme.ini 含 theme_config，删除 config.php 描述）、§7 URL 兼容清单改为去后缀新地址。
- README.md、docs/api/theme.md、docs/api/settings.md、docs/api-v1.md、template/theme-dev.md、template/theme-gen-prompt.md、template/dev-theme/dev.md、各主题 README.md。

## Impact

- Affected specs：无既有能力 spec（rewrite-go-gin-sqlite 的 §7 URL 兼容要求由本 spec 显式推翻；api-mcp-phase2 的后台页面 URL 契约随路由更名）。
- Affected code：internal/admin、internal/router、internal/pages、internal/install、internal/theme、internal/upload(注释)、internal/stats(注释)、internal/view(注释)、internal/service(测试)、web/**、assets/admin/js/**、apply/apply.js、install/templates/js/forms.js、template/**、AGENTS.md、README.md、docs/**。
- 不动项：`.trae/specs/` 历史文档、tmp/、logs/、data/ 运行产物；数据库存量值不做迁移（开发期直接切断）；staticHandler 的 .php 拦截保留。

## ADDED Requirements

### Requirement: 无后缀路由
系统 SHALL 以无 `.php` 后缀的路径注册所有页面/接口路由（清单见 What Changes §1），行为（GET/POST、handler、鉴权）与原路由完全一致。

#### Scenario: 后台登录
- **WHEN** 未登录访问 `/admin/set`
- **THEN** 302 跳转到 `/admin/login`；登录后可正常访问 `/admin/set`。

#### Scenario: 访问密码跳转
- **WHEN** GET/POST `/include/go`
- **THEN** 行为与原 `/include/go.php` 一致；GET/POST `/include/go.php` 返回 404。

#### Scenario: 安装向导表单
- **WHEN** 安装向导 step_1/step_2 表单 POST 到 `/install/`
- **THEN** 流程正常（建表、写 .env、生成 lock）。

### Requirement: 主题配置 theme_config 进 theme.ini
系统 SHALL 从主题 `theme.ini` 的 `theme_config` 数组读取自定义字段定义；`ConfigFields/ConfigValue/ConfigValues` 对外签名不变；值仍持久化于数据库 `theme_config_<tpl>`；`.env` 仍是项目唯一配置文件。

#### Scenario: 后台主题自定义设置
- **WHEN** 打开后台「主题自定义设置」页
- **THEN** 表单字段来自当前主题 theme.ini，保存/回读行为与原先一致。

#### Scenario: 前台主题配置读取
- **WHEN** 主题模板调用 `{{ .ThemeCfg "key" "def" }}`
- **THEN** 取值顺序仍为：数据库值 > theme.ini 默认值 > def；`{"$conf":"k"}` 默认值代入站点配置。

### Requirement: 全仓 .php 归零
除 `.git/`、`.trae/`、运行产物（tmp/、logs/、data/、files/、cache/）外，仓库内 SHALL 不再存在 `.php` 字样（代码、模板、JS、文档），且不存在任何 `.php` 物理文件；staticHandler 对 `.php` 后缀请求仍返回 404。

## REMOVED Requirements

### Requirement: URL 兼容原 PHP 路径（AGENTS.md §7 / rewrite spec）
**Reason**：用户明确要求切断旧 .php 地址，彻底去 PHP 化；开发期无存量用户。
**Migration**：无重定向；文档改写为新地址；数据库存量 .php 配置值不迁移。
