# LyLme Spage（Go 版）

六零导航页（原 PHP 项目 `lylme_spage`）的 **Go(Gin) + GORM** 重写版。后端负责数据、业务与模板渲染调度；前端沿用原版「套模板」机制，**HTML / CSS / JS 内容与目录结构保持不变**，仅将主题内的 PHP 模板逻辑等价转换为 Go 模板（`*.php` 作为参考保留，`*.html` 为实际渲染模板）。

> 开发规范、分层约束、主题转换契约等详见 [`AGENTS.md`](./AGENTS.md)。本文档面向部署与使用。

---

## 一、项目简介

- 兼容原版前台 URL 与主题机制，默认 SQLite，可切换 MySQL。
- 支持多主题：`default`、`lylme`、`cyberwave`、`dashlite`、`quality6.0`、`baisu`、`ltab`、`5iux`、`page`、`quality`、`dev-theme`、`liquidglass` 等（`template/*`）。
- 提供后台管理（链接/分组/搜索引擎/导航菜单/加密组/收录审核/主题设置/访问统计）、安装向导、站点地图、二维码、Bing 壁纸等能力。

## 二、技术栈

| 组件 | 说明 |
|------|------|
| 语言 | Go 1.26（`go.mod` 模块名 `lylme_spage`） |
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | SQLite（默认，纯 Go 驱动 `github.com/glebarez/sqlite`）/ MySQL（`gorm.io/driver/mysql`） |
| 配置 | `.env` / 环境变量（`github.com/joho/godotenv`） |
| 会话 | `gin-contrib/sessions`（签名 Cookie） |
| 模板 | 标准库 `html/template`（主题 `template/<主题>/*.html`，固定页 `web/*.html`） |

## 三、目录结构

```
.
├── main.go                     # 程序入口（启动服务、优雅关闭）
├── go.mod / go.sum
├── .env / .env.example         # 配置（.env 不入库）
├── AGENTS.md                   # 开发规范
├── template/<主题>/            # 原版主题（css/js 原样保留；*.php 为参考，*.html 为 Go 模板）
├── assets/                     # 原版静态资源
├── site/static/                # 详情页静态资源
├── apply/                      # 收录申请前端资源
├── web/                        # 固定页面 Go 模板（about/apply/pwd/site/404/install 等）
├── internal/
│   ├── config/                 # .env 与环境变量解析、根目录解析
│   ├── logger/                 # 分级文件日志
│   ├── models/                 # GORM 模型
│   ├── database/               # 连接、迁移、种子数据（seed.sql）
│   ├── theme/                  # 主题发现、theme.ini、config.php 解析
│   ├── site/                   # 数据访问层（配置缓存 + 前台数据查询）
│   ├── view/                   # 主题模板上下文、lists/rearr、主题辅助函数
│   ├── router/                 # 路由注册
│   ├── pages/                  # 前台固定页处理器
│   ├── admin/                  # 后台管理
│   ├── stats/                  # 访问统计
│   ├── webutil/                # 会话 / CSRF / 验证码 / 工具
│   └── fetch/ upload/ captcha/ install/
├── data/ logs/ cache/ files/   # 运行期生成物（不入库）
└── README.md
```

## 四、配置

所有配置来自 `.env` / 环境变量，完整字段参考 [`.env.example`](./.env.example)。常用项：

```dotenv
APP_ENV=production            # development | production
APP_ROOT=                     # 项目根目录；留空则取可执行文件所在目录（go run 回退当前工作目录）
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_MODE=release           # debug | release | test
ADMIN_PATH=admin              # 后台目录名（默认 /admin/）

DB_DRIVER=sqlite              # sqlite | mysql
SQLITE_PATH=data/lylme.db
DB_DSN=                       # 自定义 DSN，优先级最高
MYSQL_HOST=127.0.0.1
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=
MYSQL_DATABASE=lylme
MYSQL_CHARSET=utf8mb4

LOG_DIR=logs
LOG_LEVEL=info
SESSION_SECRET=change_me      # 生产环境务必改为随机字符串
CDN_PUBLIC=                   # 静态资源 CDN 前缀，留空使用本地 ./assets
```

### 安装向导（生成 `.env`）

访问 `/install/`，按步骤填写数据库配置 → 测试连接 → 建表并导入种子数据 → 创建管理员账号。**保存时会自动生成/覆盖项目根目录 `.env`**，并写入 `install/install.lock`；已存在锁文件时禁止重复安装。未安装时前台访问会自动跳转 `/install/`。

默认管理员账号：**admin / 123456**（密码按 `md5('lylme' + 明文)` 存储，登录后请及时修改）。

## 五、构建与运行

```bash
export PATH=/Users/macbookpro/sdk/go1.26.2/bin:$PATH

# 构建
go build ./...

# 直接运行（开发）
go run .

# 构建可执行文件
go build -o lylme_spage .

# 运行（默认读取当前目录 .env）
./lylme_spage
```

### 打包与目录约定

发布时将 **可执行文件与以下目录/文件放在同一目录**：

```
lylme_spage              # 可执行文件
.env                     # 配置
template/                # 主题模板（*.html + css/js）
assets/                  # 静态资源
site/ web/ apply/        # 页面模板与静态资源
data/ logs/ cache/ files/# 运行期生成物（可预建空目录）
```

运行时生成物均相对项目根目录：

| 目录 | 用途 |
|------|------|
| `data/` | SQLite 数据库文件（`data/lylme.db`） |
| `logs/` | 应用日志 `app-YYYYMMDD.log` 与访问统计 |
| `cache/` | 抓取/壁纸等缓存 |
| `files/` | 上传文件 |

根目录解析顺序：`APP_ROOT` > 可执行文件所在目录 > （`go run` 临时目录时）当前工作目录。

## 六、URL 路由清单

| 功能 | 路径 |
|------|------|
| 首页 | `/` |
| 详情页 | `/site-{id}.html`、`/site` |
| 收录申请 | `/apply/`、`/apply/index.php`、`/apply/wxplus.php` |
| 关于 | `/about/`、`/about` |
| 访问密码 | `/pwd/`、`/pwd`、`/include/go.php` |
| 验证码 | `/include/validatecode.php` |
| 图片上传 | `/include/file.php` |
| 二维码 | `/include/qrcode.php` |
| 站点地图 | `/sitemap.xml` |
| Bing 壁纸 | `/assets/img/bing.php`、`/assets/img/cron.php` |
| 主题点击统计占位 | `POST /clitool`（返回 `{"code":200}`） |
| 后台管理 | `/{ADMIN_PATH}/`（默认 `/admin/`，登录 `/admin/login.php`） |
| 安装向导 | `/install/` |
| 健康检查 | `/healthz` |

静态资源：`/assets/*`、`/template/*`、`/site/static/*`、`/files/*`；`.php` 源码一律返回 404。

## 七、主题转换机制

- 主题目录 `template/<主题>/` 需包含 `theme.ini`（JSON，含 `theme_version` 等）与 `index.html`；`config.php` 定义主题自定义配置 `$theme_config`。
- 模板引擎解析 `template/<主题>/*.html` 并以 `index.html` 为入口，子模板通过 `{{ template "list.html" . }}` 等方式引入。
- PHP → Go 模板主要约定：

| 原 PHP | Go 模板 |
|--------|---------|
| `<?php echo $conf['x'] ?>` | `{{ .Cfg "x" }}`（输出 HTML 用 `{{ raw (.Cfg "x") }}`） |
| `theme_config('a', def)` | `{{ .ThemeCfg "a" "def" }}` |
| `while ($row = $DB->fetch(...))` | `{{ range .Groups }}` / `{{ range .Sou }}` / `{{ range .Tags }}` |
| `include "list.php"` | `{{ template "list.html" . }}` |
| `lists($html)` | `{{ .Lists (dict "g1" ... "l3" ...) }}` |
| `listjson()` | `{{ .ListJSON }}` |
| `background()` / `yan()` / `rel` | `{{ .BackgroundImg }}` / `{{ .Yan }}` / `{{ .Rel }}` |
| `theme_css()/theme_js()/theme_e()` | `{{ .CSS "css/style.css" }}` / `{{ .JS "js/script.js" }}` / 自动转义 |
| `theme_icp()/theme_security_filing()` | `{{ .ICP }}` / `{{ .Gonganbei }}` |
| `theme_version()` | `{{ .ThemeVersion }}` |

- 模板函数：`raw`、`dict`、`css`、`js`、`b64`、`trim`、`digits`；`internal/view` 另提供 `CSS/JS/Style/ICP/Gonganbei/ThemeVersion/HasModule/EnumPick/RenderIcon/SouJSON/LGGlyph/LGBattery` 等辅助方法。
- 必须保留的契约：`lists()` 片段键与占位符、搜索区 DOM（`#search-form` / `#super-search-fm`、`input[name=sou]` 的 `value`/`data-alias`/`data-hint`）、页脚图标脚本 `assets/js/icon.js`（部分主题另含 `assets/js/svg.js`）。
- 图标（分组/链接/搜索引擎）为原始 SVG/`<use>`，模板中以 `{{ raw ... }}` 原样输出。

### 如何新增主题

1. 在 `template/` 下新建主题目录，放入 `theme.ini`、`config.php`、`index.html` 及所需 `list.html` / `head.html` / `foot.html` 等。
2. 将原 PHP 模板按上表契约改写为 Go 模板，保持 DOM/类名/ID/脚本引用与顺序不变。
3. 后台「主题」页选择该主题（写入配置项 `template`），或直接 `update lylme_config set v='<主题>' where k='template';`。
4. 访问 `/` 验证渲染（模板解析/执行错误会在响应与 `logs/` 中给出提示）。

## 八、测试与校验

```bash
export PATH=/Users/macbookpro/sdk/go1.26.2/bin:$PATH
go build ./...
go vet ./...
go test ./...
```

新增核心解析/业务逻辑请补充单元测试（参考 `internal/theme/config_test.go`、`internal/stats/stats_test.go`）。

---

更多约定（分层依赖、数据库、安全、日志、提交规范等）见 [`AGENTS.md`](./AGENTS.md)。
