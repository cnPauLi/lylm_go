# AGENTS.md — 六零导航页 Go 版开发规范

本文件是后续所有开发（含 AI Agent）必须遵循的统一规范。新增功能、修改代码、提交代码前请先阅读本文件。

## 1. 项目定位

将原 PHP 项目 `lylme_spage` 重写为 **Go(Gin) + GORM** 的前后端分离架构：

- 后端：Go 分层架构，负责数据、业务与渲染调度。
- 前端：沿用原版主题「套模板」机制，**HTML/CSS/JS 内容与目录结构保持不变**，仅将主题内的 PHP 模板逻辑等价转换为 Go 模板。
- 数据库：默认 SQLite，经 GORM 抽象，可通过 `.env` 切换 MySQL。

> 原 PHP 仓库 `lylme_spage/` 仅作为参考，属于独立 git 仓库，**不入本仓库、不修改**。

## 2. 目录结构

```
.
├── main.go                     # 程序入口
├── go.mod / go.sum
├── .env / .env.example         # 配置（.env 不入库）
├── AGENTS.md                   # 本规范
├── template/<主题>/            # 原版主题（css/js 原样保留；*.html 为 Go 模板）
├── assets/                     # 原版静态资源（原样保留）
├── site/static/                # 详情页静态资源（原样保留）
├── web/                        # 固定页面 Go 模板（about/apply/pwd/site/404/install 等）
├── internal/
│   ├── config/                 # .env 与环境变量解析
│   ├── logger/                 # 分级文件日志
│   ├── models/                 # GORM 模型
│   ├── database/               # 连接、迁移、种子数据
│   ├── theme/                  # 主题发现、theme.ini（含 theme_config 字段定义）解析
│   ├── site/                   # 数据访问层（配置缓存 + 前台数据查询）
│   ├── view/                   # 模板上下文、lists/rearr、模板缓存
│   ├── router/                 # 路由注册
│   ├── webutil/                # 会话/CSRF/验证码/工具
│   └── pages/ 或 admin/        # 页面处理器（handler）
├── dist/                       # 编译产物（统一输出目录，不入库）
├── data/ logs/ cache/ files/   # 运行期生成物（不入库）
```

## 3. 分层与依赖规范

- 依赖方向：`router → pages/admin → view/site → theme/models → database/config`，**禁止反向依赖与循环依赖**。
- `internal/site` 是唯一的数据查询入口；页面处理器不得直接拼 SQL。
- `internal/view` 只负责渲染与上下文构建，不得写业务规则。
- 运行期生成物一律通过 `config.Config.Path(...)` 拼接路径，禁止硬编码绝对路径。

## 4. 配置规范

- 所有配置全部来自 `.env` / 环境变量；**新增配置项必须同步**：`internal/config/config.go` 与 `.env.example`。
- 支持 `DB_DRIVER=sqlite|mysql`；SQLite 走纯 Go 驱动（`github.com/glebarez/sqlite`），MySQL 走 `gorm.io/driver/mysql`。
- 项目根目录解析：`APP_ROOT` > 可执行文件所在目录（`go run` 临时目录时回退到当前工作目录）。
- 安装向导保存配置时，**必须生成/覆盖 `.env` 文件**（见 §9）。

## 5. 数据库规范

- 一律使用 GORM 模型与参数化查询；除迁移与种子数据外**禁止拼接裸 SQL**。
- 模型字段与原表列名保持一致（`gorm:"column:xxx"`），`TableName()` 固定为原表名（`lylme_*`）。
- 新增字段需保证 SQLite / MySQL 双兼容（类型映射见 `internal/models`）。
- 种子数据来自 `internal/database/seed.sql`（原 `install_struct.sql` 副本），仅在配置表为空时导入。

## 6. 前端与主题规范

- **HTML/CSS/JS 内容保持不变**：转换主题时，除 PHP 标签外不得改动 DOM 结构、类名、ID、脚本引用与顺序。
- PHP → Go 模板约定：
  - `<?php echo $conf['x'] ?>` → `{{ .Cfg "x" }}`；输出 HTML 用 `{{ raw (.Cfg "x") }}`
  - `theme_config('a', def)` → `{{ .ThemeCfg "a" "def" }}`
  - `while ($row = $DB->fetch(...))` → `{{ range .Groups }}` / `{{ range .Sou }}` / `{{ range .Tags }}`
  - `include "list"` → `{{ template "list.html" . }}`
- 必须保留的模板契约：
  - `lists($html)` 的片段键 `g1/g2/g3/l1/l2/l3` 与占位符 `{group_id}/{group_name}/{group_icon}/{link_id}/{link_name}/{link_name_text}/{link_desc}/{link_url}/{link_icon}`
  - 搜索区 DOM：`#super-search-fm`、`input[name=sou]`（`value`/`data-alias`/`data-hint`）
  - 页脚必须保留 `assets/js/icon.js`、`assets/js/svg.js`
- 主题发现：`template/*` 目录需包含 `theme.ini`（JSON）与 `index.html`；theme.ini 顶层 `theme_config` 数组定义字段：type/name/title/description/verify/placeholder/value/enum，value 支持 `{"$conf":"键"}` 引用站点配置，enum 为 [value,label] 二元组数组。
- 前端 JS 逻辑不改；URL 必须与原版兼容（见 §7）。

## 7. URL 清单（无后缀新地址）

| 功能 | 路径 |
|------|------|
| 首页 | `/` |
| 详情页 | `/site-{id}.html` |
| 收录申请 | `/apply/`、`/apply/wxplus` |
| 关于 | `/about/` |
| 访问密码 | `/pwd/`、`/include/go` |
| 验证码 | `/include/validatecode` |
| 图片上传 | `/include/file` |
| 二维码 | `/include/qrcode` |
| 站点地图 | `/sitemap.xml` |
| Bing 壁纸 | `/assets/img/bing`、`/assets/img/cron` |
| 后台 | `/{ADMIN_PATH}/`（默认 `/admin/`） |
| 安装向导 | `/install/` |

静态资源：`/assets/*`、`/template/*`、`/site/static/*`；**必须拦截 PHP 脚本文件请求**（返回 404）。

## 8. 安全规范

- 使用参数化 ORM 查询；所有输入做长度与类型校验。
- CSRF：写操作需校验 CSRF Token。
- SSRF：服务端发起的 HTTP 请求须校验目标 IP，拒绝内网/保留地址。
- 限流：对外接口（抓取、申请、推送等）使用限流。
- **不得移植**原版 `include/include` 等混淆/`eval` 代码，等价能力用 Go 重写。
- 管理员密码沿用 `md5('lylme'.明文)` 存储；会话 Cookie 使用签名。

## 9. 安装向导规范

- 保留 `/install/` 安装流程：填写数据库配置 → 测试连接 → 建表/导入种子 → 创建管理员 → 生成 `install/install.lock`。
- **保存时必须生成 `.env`**：将数据库与应用配置写入项目根目录 `.env`（覆盖或创建），随后按 `.env` 重连。
- 已存在 `install/install.lock` 时，访问 `/install/` 应提示已安装并禁止重复安装。

## 10. 日志规范

- 统一使用 `internal/logger`，输出到 `logs/app-YYYYMMDD.log` 与控制台。
- 请求日志记录：方法、路径、状态码、耗时。
- 禁用 `fmt.Println` 之类临时输出。

## 11. 提交规范

- 建立 Git 版本；**每完成一个阶段提交一次**，保持工作区干净。
- 提交信息格式：`feat(阶段N): 摘要` / `fix(阶段N): 摘要` / `chore(阶段N): 摘要`。
- 禁止提交 `.env`、`dist/`、`data/`、`logs/`、`cache/`、`files/`（见 `.gitignore`）。

## 12. 测试与构建

```bash
export PATH=/Users/macbookpro/sdk/go1.26.2/bin:$PATH
go build -o dist/lylme_spage .   # 编译产物必须输出到 dist/，禁止落在根目录或 tmp/
go vet ./...
go test ./...
```

- **二进制统一构建到项目根 `dist/` 目录**；`go build ./...` 仅用于编译校验，不产生产物。
- 运行 `dist/lylme_spage` 时需在项目根目录执行，或设置 `APP_ROOT` 指向项目根（见 §4），确保日志等运行期路径解析到根目录而非 `dist/`。

- 新增核心解析/业务逻辑需补充单元测试（参考 `internal/theme/config_test.go`）。

## 13. Agent 协作规范

- **每个 task 使用独立 agent 执行**：一个 task 一个 agent，任务边界清晰、互不干扰。
- 执行前先阅读 `spec.md` / `tasks.md` / `checklist.md` 与本文件。
- 无依赖的 task 可并行；有依赖的按 `tasks.md` 的依赖顺序串行。
- 完成一个 task 后：勾选 `tasks.md` 对应项、更新 `checklist.md`、按阶段提交。
