# Checklist：去除全部 PHP 后缀

- [x] 后台所有路由无 .php 后缀（/admin/login、/admin/set…），旧 /admin/*.php 返回 404
- [x] 后台 Go 代码中页面标识、重定向、alert/script 跳转串、生成 HTML 全部无 .php
- [x] web/admin/*.html 与 assets/admin/js/*.js 中链接/AJAX/选择器无 .php，`.Is` 判断与新页面标识一致
- [x] 前台路由无 .php：/include/{validatecode,file,go,qrcode}、/apply/wxplus、/assets/img/{bing,cron}；POST /apply 与 /apply/ 可用
- [x] /apply/index.php、/include/go.php 等旧地址返回 404（直接切断，无跳转）
- [x] 安装向导：/install/index.php 路由删除，表单 POST /install/ 流程正常
- [x] 后台首页 AJAX 检查更新（?check=update）返回 JSON 正常
- [x] CronImage 报错文案输出新地址 /assets/img/cron?key=…
- [x] 主题字段定义来自 theme.ini 的 theme_config（JSON，enum 保序，$conf 默认值可解析），11 个主题 config.php 已删除
- [x] Manager.ConfigFields/ConfigValue/ConfigValues 对外签名不变；admin 主题自定义设置、MCP、API 主题工具行为不变
- [x] internal/theme/config.go PHP 解析器代码已删除，config_test.go 改为 theme.ini 测试且通过
- [x] template/ 下仅剩 .html/.ini/css/js/img 等非 PHP 文件；assets/img/bing.php、cron.php 已删除
- [x] staticHandler 仍拦截 .php 后缀请求返回 404
- [x] AGENTS.md §2/§6/§7、README.md、docs/api/*.md、docs/api-v1.md、主题开发文档无 .php 残留
- [x] `go build ./...`、`go vet ./...`、`go test ./...` 全部通过
- [x] 全仓 grep `\.php` 归零（排除 .git/.trae/tmp/logs/data/files/cache）
- [x] 按阶段完成 git 提交，工作区干净
