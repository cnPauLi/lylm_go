# 六零导航页 Go 重写 Spec

## Why
现有项目 `lylme_spage` 为 PHP + MySQL 单体应用，运行依赖 PHP 扩展与 Web 服务器，部署重、可维护性差。需要在不改变前端外观与交互的前提下，将后端重写为 Go（Gin），数据库改用 GORM 以便在 SQLite / MySQL 间灵活切换，并统一通过 `.env` 配置。

## What Changes
- 将 PHP 后端整体重写为 Go（Gin）+ GORM 的分层架构，前后端职责分离（业务逻辑 / 数据访问 / 视图渲染分层）。
- 数据库默认 SQLite，通过 ORM 抽象，可通过 `.env` 切换 MySQL；所有数据库配置项全部来自 `.env`，并提供 `.env.example` 示例。
- 前端沿用原版主题「套模板」机制：保留原主题的 HTML/CSS/JS 内容与目录结构（`template/<主题>/`）不变，将主题内 PHP 模板逻辑等价转换为 Go 模板（新增同名 `.html`，原 `.php` 作为参考保留）。
- 简化开发流程：统一日志、统一配置、统一路由，去除非必要的历史兼容代码。
- 运行期生成物（SQLite 数据库、日志、缓存、上传文件）统一落在项目根目录；打包后落在可执行文件同目录。
- 建立 Git 版本管理，每完成一个阶段提交一次，避免代码混乱。
- 保留 `/install/` 安装向导，但保存时改为**生成 `.env` 文件**（不再写 `config.php`），随后建表、导入种子数据并生成安装锁。
- **BREAKING**：不再依赖 PHP 运行时。
- **BREAKING**：原 `include/include.php`（混淆代码，内含 `eval` 动态执行，且 `theme_file()/siteurl()/theme()/wxPlus()/update()` 等契约函数定义于此）不予移植，改为 Go 内聚实现。

## Impact
- Affected specs: 前台首页与主题渲染、详情页、收录申请、关于页、访问密码、站点地图/二维码/Bing 壁纸等工具、后台管理、数据库与配置、日志。
- Affected code（新增 Go 工程）：`main.go`、`internal/{config,logger,models,database,theme,site,view,router,webutil,...}`、`template/*/*.html`、`web/*.html`、`.env(.example)`。
- Affected code（保留参考，不修改）：`lylme_spage/`（原 PHP 仓库，独立 git，不入新仓库）。

## ADDED Requirements

### Requirement: 配置全部来自环境变量
系统 SHALL 从 `.env` / 环境变量读取全部配置（应用、数据库、日志、会话、CDN），构建时提供 `.env.example` 示例文件，缺省值保证开箱可运行。

#### Scenario: 默认 SQLite 启动
- **WHEN** 未提供任何数据库环境变量直接启动
- **THEN** 系统使用 SQLite，在项目根目录 `data/` 下创建数据库文件并完成建表与种子数据导入

#### Scenario: 切换 MySQL
- **WHEN** 设置 `DB_DRIVER=mysql` 及 `MYSQL_*` 系列变量
- **THEN** 系统通过 GORM 的 MySQL 驱动连接并使用同一套模型，无需改动业务代码

### Requirement: 运行期文件落在根目录/可执行文件同目录
系统 SHALL 将 SQLite 数据库、日志、缓存、上传文件写入项目根目录；打包运行时使用可执行文件所在目录。

#### Scenario: 打包后运行
- **WHEN** 将二进制与 `template/`、`assets/`、`.env` 放在同一目录执行
- **THEN** `data/`、`logs/`、`cache/`、`files/` 均在该目录下生成

### Requirement: 主题套模板渲染
系统 SHALL 支持多主题，扫描 `template/*` 目录，读取 `theme.ini` 元信息与 `config.php` 主题配置定义，按主题渲染首页；保持原版 `lists()` 片段占位符（`g1/g2/g3/l1/l2/l3` 与 `{group_*}/{link_*}`）与主题配置读取语义。

#### Scenario: 渲染默认主题首页
- **WHEN** 访问 `/` 且 `template=default`
- **THEN** 返回与原 PHP 版一致的 HTML（分组导航、搜索引擎、链接列表、菜单、页脚备案/版权/统计），静态资源经 `/assets`、`/template` 提供

#### Scenario: 主题自定义配置
- **WHEN** 主题模板读取 `theme_config(name, default)`
- **THEN** 优先返回数据库 `theme_config_<主题>` 的 JSON 值，其次返回 `config.php` 中该字段默认值，最后返回默认参数

### Requirement: 固定功能页
系统 SHALL 提供与原版等价、URL 兼容的固定页面与接口：详情页 `/site-{id}.html`、收录申请 `/apply/`、关于 `/about/`、访问密码 `/pwd/`、站点地图 `/sitemap.xml`、二维码 `/include/qrcode.php`、Bing 壁纸 `/assets/img/bing.php`、壁纸缓存 `/assets/img/cron.php`、图片上传 `/include/file.php`、密码校验 `/include/go.php`、验证码 `/include/validatecode.php`。

#### Scenario: 收录申请
- **WHEN** 用户提交收录表单（含验证码）
- **THEN** 服务端校验验证码与限流，按 `apply` 配置写入 `lylme_apply`（`status=1` 时同时写入链接表），返回 JSON `{code,msg}`

#### Scenario: 详情页
- **WHEN** 访问 `/site-{id}.html`
- **THEN** 在加密可见性允许时渲染详情页；描述/关键词缺失时在线采集并回写数据库

### Requirement: 安装向导（生成 .env）
系统 SHALL 保留 `/install/` 安装向导：填写数据库与应用配置 → 测试数据库连接 → 保存时生成项目根目录 `.env` 文件 → 自动建表并导入种子数据 → 创建管理员账号 → 生成 `install/install.lock`。已存在安装锁时禁止重复安装并给出提示。

#### Scenario: 首次安装
- **WHEN** 未生成 `install/install.lock` 时访问 `/install/` 并提交数据库配置
- **THEN** 系统写入 `.env`、按该配置建表与导入种子数据、写入安装锁，并跳转完成页

#### Scenario: 重复安装
- **WHEN** 已存在 `install/install.lock` 时访问 `/install/`
- **THEN** 提示已安装并阻止再次执行安装流程

### Requirement: 后台管理
系统 SHALL 提供与原版等价的登录鉴权与后台功能：仪表盘、网站设置、链接管理、分组管理、搜索引擎、导航菜单、加密组、收录审核、主题切换与主题设置、账号安全、图片清理、批量导入、更新检查等，并保留 `{code,msg}` JSON 约定。

#### Scenario: 登录
- **WHEN** 提交正确的管理员账号密码
- **THEN** 写入签名会话并跳转后台首页；连续失败触发图形验证码与 IP 锁定

### Requirement: 日志与审计
系统 SHALL 输出分级日志到 `logs/app-YYYYMMDD.log` 并同时输出到控制台。

#### Scenario: 请求日志
- **WHEN** 处理任意 HTTP 请求
- **THEN** 记录方法、路径、状态码与耗时

### Requirement: Git 分阶段提交
系统 SHALL 建立 Git 仓库，每完成一个阶段提交一次，提交信息按「阶段N」标注。

#### Scenario: 阶段提交
- **WHEN** 完成一个阶段（如阶段1 首页渲染）并通过验证
- **THEN** 执行一次独立提交，工作区保持干净

### Requirement: 开发规范文档与 Agent 协作
系统 SHALL 在项目根目录提供 `AGENTS.md`，沉淀分层、配置、数据库、前端模板、URL 兼容、安全、安装向导、日志、提交、测试与 Agent 协作规范；后续任务 SHALL 遵循该规范，且**每个 task 由独立 agent 执行**。

#### Scenario: 新增功能
- **WHEN** 新增一个页面或主题
- **THEN** 实现方式符合 `AGENTS.md` 的分层与模板契约约定，URL 与原版兼容，并有对应阶段提交

## MODIFIED Requirements

### Requirement: 数据库访问
由 PHP 的 `DB` 类（mysqli/PDO 手写 SQL）改为 GORM：模型映射 `lylme_config/links/groups/apply/pwd/sou/tags`，自动建表（AutoMigrate），首次运行从原版 `install_struct.sql` 解析导入种子数据（完整保留 149 条链接等原始内容）。

### Requirement: 主题模板
由 PHP `require`/`include` 原生模板改为 Go `html/template`：原 PHP 标签（`echo`/`if`/`while`/`include`）转换为 Go 模板动作，HTML/CSS/JS 内容与 DOM 结构保持一致；主题 JS 依赖的搜索框 DOM 契约、`icon.js`/`svg.js` 图标注入保持不变。

## REMOVED Requirements

### Requirement: PHP 运行时与 config.php
**Reason**: 重写为 Go 单体可执行文件，不再依赖 PHP，也不需要写 `config.php`。
**Migration**: 改用 `.env` 配置数据库；安装向导保存时生成 `.env`；首次启动自动建表并导入种子数据。

### Requirement: `include/include.php` 混淆代码
**Reason**: 该文件为 base64+XOR+gzinflate+`eval` 的混淆代码（安全风险），且其中定义的主题契约函数缺失静态实现。
**Migration**: 在 Go 中以 `theme`/`view`/`router` 包内聚实现 `theme_file()`、`siteurl()`、主题元信息与更新检查等等价能力；不移植该文件。

### Requirement: WAF 正则拦截
**Reason**: 原 WAF 通过正则拦截 GET/POST/Cookie 并 `exit`，易误伤且与 Go 参数化查询理念冲突。
**Migration**: 以参数化 ORM 查询 + CSRF + SSRF 防护 + 限流替代；如需保留可后续以中间件形式按需加回。
