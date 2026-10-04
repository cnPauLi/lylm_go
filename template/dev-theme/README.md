# 开发包devTheme —— 六零导航页主题开发包

单文件入口 + 零框架依赖的规范主题骨架。复制本目录、改名、改 CSS 即可做出自己的主题。

适用版本：六零导航页 v1.2.5 及以上

模板引擎：Go html/template（数据输出自动 HTML 转义，官方 HTML 用 `raw` 原样输出）

## 目录结构

```
dev-theme/
├── index.html         # 入口模板（必须）：单文件，按注释分区
├── theme.ini          # 主题信息 + 配置定义（必须）：缓存版本号与配置表单从这里读
├── README.md          # 本文件
├── css/style.css      # 样式：改 :root 变量即可换肤
└── js/script.js       # 脚本：搜索引擎切换 + 时间显示
```

## 快速开始

1. 复制 `dev-theme` 目录，改名为你的主题目录（小写英文，如 `mynav`）
2. 修改 `theme.ini`：`theme_name` 与目录名对应，`theme_version` 用三段式（如 `1.0.0`）
3. 按需增删 `theme.ini` 顶层键 `theme_config` 中的配置项（`name` 主题内唯一即可，配置按主题独立存储，不会与其它主题冲突）
4. 在后台「系统设置 - 网站主题」中切换到你的主题

> 改动 CSS / JS 后不需要手动改缓存版本号——`.CSS` / `.JS` 会读取 `theme.ini`
> 的 `theme_version`，改动后只需把版本号 +1，缓存自动失效。

## 可用的内置模板函数（等价原版主题的 theme_* 契约函数，核心已内置）

| 函数 | 作用 |
| ---- | ---- |
| `{{ .CSS "css/style.css" }}` | 输出 `<link>` 标签，自动附加 `?v=版本号` |
| `{{ .JS "js/script.js" }}` | 输出 `<script defer>` 标签，自动附加版本号 |
| `{{ .ThemeVersion }}` | 读取 theme.ini 的版本号（去除非字母数字） |
| `{{ .Cfg "键名" }}` | 读取站点配置，自动 HTML 转义 |
| `{{ .ThemeCfg "name" "默认值" }}` | 读取主题自定义配置，自动转义 |
| `{{ range .Tags }}` | 导航菜单数组：`.TagID` `.TagName` `.TagLink` `.TagTarget` |
| `{{ range .Sou }}` | 搜索引擎数组：`.Link` 已按 PC / 手机自动解析 |
| `{{ .ICP }}` | 输出 ICP 备案链接（读 `icp`），留空不输出 |
| `{{ .Gonganbei }}` | 输出公安备案链接（读主题配置 `gonganbei`），自动提取数字拼查询地址，留空不输出 |

> 取数类成员（`.Tags`/`.Sou`）配合 `range` 使用，输出类函数（`.CSS`/`.JS`/`.ICP`/`.Gonganbei`）直接输出完整标签。

列表数据仍使用官方 `lists()` 接口（`{{ .Lists (dict ...) }}`），片段键与占位符未做任何改动。

## 搜索区的 DOM 契约

`js/script.js` 依赖以下结构，**修改搜索区时请保持这些约定**：

```html
<form id="search-form" action="#">
    <input type="text" id="search-input">
</form>
<label class="engine">
    <input type="radio" name="sou" value="接口地址" data-alias="引擎别名" data-hint="提示文字">
    ...
</label>
```

- 切换引擎 → 更新输入框提示 → 写入 `localStorage['theme_sou']`（键全站统一，换主题也保留）
- 提交搜索 → JS 拼接「接口地址 + encodeURIComponent(关键词)」后新窗口打开
  （各引擎参数名不统一：`wd=` / `q=` / `word=`，因此不走表单原生 GET）

## 系统配置项（主题可用白名单）

| 键 | 说明 |
| ---- | ---- |
| `title` / `keywords` / `description` | SEO 与标题 |
| `logo` | LOGO / favicon |
| `background` / `wap_background` | 背景图（用 `.BackgroundImg`，已自适应 PC/手机） |
| `copyright` / `wztj` | 底部版权、自定义统计代码（HTML，`raw` 原样输出） |
| `icp` | ICP 备案号，留空不显示 |
| `yan` | 随机一言开关，判断用 `eq (.Cfg "yan") "true"` |
| `tq` | 天气开关（核心不消费，主题自行决定是否展示） |
| `template` / `version` / `cdnpublic` | 系统项（CDN 地址用 `.Cdn`） |

其他键（`apply`、`about`、`home-title` 等）主题不应依赖。

## 必须保留的三处代码

```html
{{ raw (.Cfg "wztj") }}                                  <!-- 统计代码，放在最底部 -->
<script src="{{ .Cdn }}/assets/js/icon.js"></script>     <!-- 图标雪碧图 -->
<script src="{{ .Cdn }}/assets/js/svg.js"></script>      <!-- 旧版图标雪碧图 -->
```

`svg.js`和`icon.js` 注入了数据库所有 `#lyicon-*` 图标的 SVG symbol，删除后站点图标将全部空白。

## theme.ini 字段说明

必填六项：`author_name`、`author_link`、`theme_name`、`theme_version`、`theme_explain`、`theme_demo`

建议补充：`requires`（最低程序版本）、`updated_at`、`license`（核心忽略未识别字段，向后安全）

可选：`theme_course`（主题教程链接，配置后后台显示"主题教程"按钮）

可选：`theme_config`（JSON 数组，定义后台「主题设置」表单；元素含 `type`/`name`/`title`/`description`/`value`/`enum`/`verify`/`placeholder`，`enum` 用二维字符串数组 `[["值","显示文本"],...]`，`value` 支持字符串、字符串数组或 `{"$conf": "站点配置键"}`）

> `theme_version` 会被 `.CSS` / `.JS` 用作缓存版本参数，改动静态资源后把版本号 +1 即可。

## 自检清单

- [ ] `theme.ini` 六字段齐全、JSON 合法，版本号三段式
- [ ] 目录名与 `theme_name` 对应
- [ ] 模板无未转义输出（官方 HTML 除外，且均经 `raw` 显式声明）
- [ ] 保留 `wztj` 输出与 `icon.js`和`svg.js` 引入
- [ ] 移动端 768px 断点下排版正常
- [ ] 搜索引擎切换后刷新页面，选择被正确记忆
