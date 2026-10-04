# Tasks：去除全部 PHP 后缀

> 执行规范：**每个 task 由独立 agent 执行**（AGENTS.md §13）；Task 1–6 文件互不重叠可并行；Task 7 依赖全部完成后串行收尾。执行前先读 [spec.md](file:///Volumes/data/lylm_go/.trae/specs/remove-php-suffix/spec.md) 与 [AGENTS.md](file:///Volumes/data/lylm_go/AGENTS.md)。
> 构建命令：`export PATH=/Users/macbookpro/sdk/go1.26.2/bin:$PATH && go build ./... && go vet ./... && go test ./...`
> 通用规则：只做去 .php 后缀与 spec 明确的等价改动，不顺手重构；改动后必须 `go build ./...` 通过（涉及 Go 的任务还需 `go vet`，涉及测试的任务跑对应包测试）。

- [x] Task 1: 后台域去 .php（Go + 模板 + JS + 测试）
  - [x] 1.1 internal/admin：handler.go 路由注册去后缀（login.php→login；index.php 注册删除；set/about/user/link/table_link/ajax_link/group/table_group/sou/tag/pwd/apply/ajax_apply/theme/theme_setting/ajax_theme/cleanimg/batch_add/update/wxplus/license/help/mcp/ajax_mcp 全部去后缀）；auth.go/dashboard.go/settings.go/misc.go/groups.go/links.go/apikey_page.go 中页面标识、重定向、alert/script 跳转串、生成 HTML 的 form action 去后缀；注释中的 .php 文件名去后缀
  - [x] 1.2 web/admin/*.html：href/action/src、`.Is "xxx.php"` 判断、内联 AJAX（index.html 的 `index.php?check=update`→`?check=update`、update.html/edit_link.html/mcp.html 等）去后缀；user.html 的 `./filecheck.php`→`./filecheck`
  - [x] 1.3 assets/admin/js/{group,link,apply,theme_setting}.js：AJAX url 与 `form[action*="xxx.php"]` 选择器去后缀
  - [x] 1.4 internal/admin/apikey_page_test.go：URL 与断言去后缀，包测试通过
- [x] Task 2: 前台/公共路由域去 .php
  - [x] 2.1 internal/router/router.go：include 四路由、apply 路由（删 index.php 注册、POST /apply 与 /apply/ 补注册）、wxplus、assets 拦截 bing/cron 去后缀；注释改写（屏蔽 .php 逻辑保留）
  - [x] 2.2 internal/pages/{apply,pwd,site,media}.go：注释与 CronImage 输出文案（`/assets/img/cron.php?key=秘钥`→`/assets/img/cron?key=秘钥`）去后缀；internal/upload/upload.go、internal/stats/stats.go 注释去后缀
  - [x] 2.3 web/{apply,pwd,site}.html 与 apply/apply.js：验证码 src、go.php form action、qrcode src、AJAX url（`./wxplus.php?wx=plus`→`./wxplus?wx=plus`、`index.php?submit=post`→`./`、`index.php`→`./`、`/include/file.php`→`/include/file`）去后缀
- [x] Task 3: install 域去 .php
  - [x] 3.1 internal/install/handler.go：删除 `base+"/index.php"` GET/POST，新增 `POST base+"/"`；注释去后缀
  - [x] 3.2 web/install/step_1.html、step_2.html：`action="index.php"`→`action="./"`；install/templates/js/forms.js：三处 `url: 'index.php'`→`url: './'`
- [x] Task 4: config.php 废除 → theme.ini 迁移
  - [x] 4.1 internal/theme/config.go 重写：删除 PHP 解析器，新增 theme.ini `theme_config` JSON 解析（Field/EnumOption/DefaultValue/DefaultValueList 保留；`{"$conf":"k"}` 代入 conf；enum 用 `[["v","label"],...]` 保序）；Manager.ConfigFields 改读 theme.ini，对外签名不变
  - [x] 4.2 11 个主题：config.php 的 $theme_config 迁入各自 theme.ini（default/cyberwave/dev-theme/5iux/baisu/dashlite/liquidglass/lylme/page/quality6.0；quality 无 config.php 无需处理）后删除 config.php
  - [x] 4.3 internal/theme/config_test.go 重写为 theme.ini 测试；internal/service/theme_test.go 同步；internal/view/view.go 注释去后缀
  - [x] 4.4 主题文档改写：template/theme-dev.md、template/theme-gen-prompt.md、template/dev-theme/dev.md、template/{5iux,baisu,cyberwave,dev-theme,liquidglass}/README.md
- [x] Task 5: 删除死 .php 物理文件
  - [x] 5.1 删除 template/*/ 下 index.php、footer.php、list.php、foot.php、head.php、go.php、functions.php（不碰 config.php——归 Task 4、不碰 .html/.ini/js/css）
  - [x] 5.2 删除 assets/img/bing.php、assets/img/cron.php
- [x] Task 6: 文档更新
  - [x] 6.1 AGENTS.md：§2 目录说明、§6 主题约定（theme.ini 含 theme_config；删除 config.php 描述）、§7 URL 兼容清单改为新地址
  - [x] 6.2 README.md、docs/api/theme.md、docs/api/settings.md、docs/api-v1.md 中 .php 引用改写
- [x] Task 7: 总验证与收尾（主线程执行）
  - [x] 7.1 `go build ./... && go vet ./... && go test ./...` 全绿
  - [x] 7.2 全仓 grep `.php` 归零（排除 .git/.trae/tmp/logs/data/files/cache）
  - [x] 7.3 冒烟：启动服务验证 /admin/login、/admin/set、/apply/、/include/validatecode、/include/go、/assets/img/bing、主题页
  - [x] 7.4 按 AGENTS.md §11 分阶段提交

# Task Dependencies
- Task 1–6 互不重叠，可并行；Task 5 的 5.1 与 Task 4 的 4.2 都作用于 template/ 但删除文件集合不同，无冲突。
- Task 7 依赖 Task 1–6 全部完成。
