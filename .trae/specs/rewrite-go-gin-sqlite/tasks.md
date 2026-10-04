# Tasks

> 执行规范：**每个 task 由独立 agent 执行**；无依赖的 task 可并行，其余按依赖顺序串行。开发前请先阅读 `AGENTS.md`。

- [x] Task 0.0: 补充 `AGENTS.md` 项目开发规范文档（分层/配置/数据库/前端模板/URL 兼容/安全/安装向导/日志/提交/测试/Agent 协作）

- [x] Task 0: 项目骨架与数据层
  - [x] SubTask 0.1: `go mod init`、`.gitignore`、`.env`/`.env.example`、git 初始化
  - [x] SubTask 0.2: `internal/config` 加载 .env 与环境变量，解析根目录（可执行文件同目录，go run 回退工作目录）
  - [x] SubTask 0.3: `internal/logger` 分级文件日志（`logs/app-YYYYMMDD.log`）
  - [x] SubTask 0.4: `internal/models` GORM 模型（config/links/groups/apply/pwd/sou/tags）
  - [x] SubTask 0.5: `internal/database` 打开 SQLite/MySQL、AutoMigrate、从 `install_struct.sql` 解析导入种子数据
  - [x] SubTask 0.6: `main.go` 启动服务、静态资源服务、优雅关闭

- [x] Task 1: 主题模板引擎与前台首页
  - [x] SubTask 1.1: `internal/theme` 主题发现、`theme.ini` 解析、PHP 数组子集解析（`config.php`）
  - [x] SubTask 1.2: `internal/site` 配置缓存与前台数据查询（分组+链接可见性、菜单、搜索引擎）
  - [x] SubTask 1.3: `internal/view` ViewData 上下文、`lists()`/`rearr()` 占位符、`listjson()`、模板缓存
  - [x] SubTask 1.4: `internal/router` + `internal/webutil` 路由、移动端识别、静态资源服务（屏蔽 `.php`）
  - [x] SubTask 1.5: 转换 default 主题（`index.html`/`list.html`/`footer.html`）
  - [x] SubTask 1.6: 主题配置解析单元测试

- [ ] Task 2: 会话/安全基础能力
  - [ ] SubTask 2.1: 引入 `gin-contrib/sessions`，注册会话中间件
  - [ ] SubTask 2.2: 会话与 CSRF 助手（解锁组列表、登录态、CSRF 生成/校验、验证码存取）
  - [ ] SubTask 2.3: 图形验证码生成（4 位数字 PNG，自绘点阵字体）
  - [ ] SubTask 2.4: SSRF 安全 HTTP 客户端与 `GetHead()`（标题/图标/描述/关键词，带缓存）
  - [ ] SubTask 2.5: 图片上传/远程抓取接口 `/include/file.php`（类型/大小/限流）
  - [ ] SubTask 2.6: 访问密码解锁与注销 `/include/go.php`

- [ ] Task 3: 固定功能页
  - [ ] SubTask 3.1: 详情页 `/site-{id}.html`（含 404 与在线采集回写）
  - [ ] SubTask 3.2: 收录申请 `/apply/`、`/apply/index.php`、`/apply/wxplus.php`
  - [ ] SubTask 3.3: 关于页 `/about/`
  - [ ] SubTask 3.4: 访问密码页 `/pwd/`
  - [ ] SubTask 3.5: 站点地图 `/sitemap.xml`
  - [ ] SubTask 3.6: 二维码 `/include/qrcode.php`
  - [ ] SubTask 3.7: Bing 壁纸 `/assets/img/bing.php` 与缓存 `/assets/img/cron.php`
  - [ ] SubTask 3.8: 静态资源补齐（`/site/static/*`、`/apply/apply.js`）与路由接线

- [ ] Task 4: 后台管理
  - [ ] SubTask 4.1: 登录/登出与后台鉴权中间件（Cookie 签名、失败锁定、验证码）
  - [ ] SubTask 4.2: 后台布局（head/footer 菜单高亮）与仪表盘
  - [ ] SubTask 4.3: 网站设置 / 关于页设置 / 账号安全
  - [ ] SubTask 4.4: 链接管理（列表片段、增删改、批量操作、排序、查重）
  - [ ] SubTask 4.5: 分组管理、搜索引擎、导航菜单、加密组
  - [ ] SubTask 4.6: 收录审核
  - [ ] SubTask 4.7: 主题切换与主题自定义设置
  - [ ] SubTask 4.8: 图片清理、批量导入、更新检查、微信推送、授权页

- [ ] Task 5: 剩余主题与收尾验证
  - [ ] SubTask 5.1: 其余主题（lylme/cyberwave/dashlite/quality6.0 等）模板转换为 Go 模板
  - [ ] SubTask 5.2: `go build` / `go vet` / `go test` 全量通过，端到端页面自检
  - [ ] SubTask 5.3: 更新 README 与部署说明（构建、打包、目录约定）

- [ ] Task 6: 安装向导（生成 .env）
  - [ ] SubTask 6.1: 安装入口 `/install/` 与步骤页（Go 模板，沿用原版安装页面样式）
  - [ ] SubTask 6.2: 数据库配置表单 + 连接测试 + SQLite/MySQL 选择
  - [ ] SubTask 6.3: 保存时生成项目根目录 `.env` 文件（含数据库/应用配置项）
  - [ ] SubTask 6.4: 按新 `.env` 建表、导入种子数据、创建管理员账号
  - [ ] SubTask 6.5: 生成 `install/install.lock`，重复安装拦截
  - [ ] SubTask 6.6: 未安装时前台访问跳转 `/install/`（对齐原版行为）

# Task Dependencies
- Task 1 依赖 Task 0
- Task 2 依赖 Task 1
- Task 3 依赖 Task 2
- Task 4 依赖 Task 2（部分依赖 Task 3 的上传能力）
- Task 5 依赖 Task 4
- Task 6 依赖 Task 2（复用配置/数据库/种子能力），可与 Task 5 并行
