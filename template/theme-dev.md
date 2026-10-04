### 前言

>   主题开发仅支持 **六零导航页 v1.2.5** 及以上版本

>   文档最后更新：2026 年 10 月 4 日

#### 主题投稿方式

- QQ群：933434961
- 邮件：admin@lylme.com
- PR：Gitee、Github

### 开始

#### 开发包

[下载开发包-dev-theme.zip](https://cdn.lylme.com/lylme_spage/themes/dev-theme.zip)

官方开发包即 `template/dev-theme` 脚手架主题：**复制整个目录 → 改名 → 改 CSS**，即可开始开发。

```
复制 template/dev-theme  →  template/你的主题目录名（小写英文）
修改 theme.ini          →  theme_name 与目录名对应，theme_version 用三段式
修改 theme.ini 的 theme_config →  增删主题自己的配置项
在后台「系统设置 - 网站主题」切换启用
```

### 目录结构

```
/template								# 主题目录（主题上传目录）
└── dev-theme							# 主题文件夹（目录名 = theme_name 的小写形式）
        ├── index.html					# 主题入口模板 (必须)
        ├── theme.ini					# 主题说明 + 自定义配置 (必须)
        ├── README.md					# 主题说明文档 (建议)
        ├── css							# css目录
        │   └── style.css				# 样式文件（唯一入口）
        ├── js							# js目录
        │   └── script.js				# 脚本文件（唯一入口）
        └── img							# 图片目录 (可选)
```

#### 文件说明

| 文件名        | 作用                                   | 必须 |
| ------------- | -------------------------------------- | ---- |
| index.html    | 主题入口模板（Go template）            | 是   |
| theme.ini     | 主题信息 + 配置定义文件（JSON）        | 是   |
| css/style.css | 主题样式文件                           | 否   |
| js/script.js  | 主题脚本文件                           | 否   |
| README.md     | 主题说明文档                           | 否   |

**设计原则：轻量、单文件、零框架。** 页面结构全部写在 `index.html` 内，用注释分区即可，不必拆成 `head.html`/`list.html` 等多个文件（核心不感知主题内部文件，拆分与否由作者决定）。

### 主题信息

**文件：theme.ini**

**格式：json**

**说明：** 前六个字段必填，其余可选；不支持 HTML 标签

```json
{
    "author_name": "六零",
    "author_link": "https://gitee.com/lylme",
    "theme_name": "DevTheme",
    "theme_version": "1.0.0",
    "theme_explain": "基于官方开发包开发",
    "theme_demo": "https://hao.lylme.com/theme/dev-theme",
    "requires": ">=1.2.5",
    "updated_at": "2026-08-29",
    "license": "MIT",
    "theme_course": "https://doc.lylme.com/spage/"
}
```

| 参数          | 说明                                     | 必填 |
| ------------- | ---------------------------------------- | ---- |
| author_name   | 作者名称                                 | 是   |
| author_link   | 作者主页                                 | 是   |
| theme_name    | 主题名称（与目录名对应）                 | 是   |
| theme_version | 主题版本，**三段式**如 `1.0.0`           | 是   |
| theme_explain | 主题说明                                 | 是   |
| theme_demo    | 演示站点                                 | 是   |
| requires      | 最低程序版本                             | 否   |
| updated_at    | 更新日期                                 | 否   |
| license       | 开源协议                                 | 否   |
| theme_course  | 主题教程链接（后台显示「主题教程」按钮） | 否   |

> `theme_version` 会被模板函数 `.ThemeVersion` 用作 css/js 的缓存版本参数（去除非字母数字后拼接），
> 改动静态资源后把版本号 +1 即可自动刷新缓存，无需手写 `?v=20260826`。

### 主题自定义配置

**文件：theme.ini（顶层键 `theme_config`）**

**格式：JSON 数组**

```json
{
    "theme_config": [
        {
            "type": "color",
            "name": "color",
            "title": "主题主色",
            "description": "用于搜索按钮、链接悬停等主色调，留空则使用默认蓝色",
            "value": "#2f6fed"
        },
        {
            "type": "select",
            "name": "link_cols",
            "title": "列表列数",
            "description": "每行显示的链接数量，手机端会自动变为 2 列",
            "value": "4",
            "enum": [["3", "3 列"], ["4", "4 列"], ["5", "5 列"], ["6", "6 列"]]
        },
        {
            "type": "checkbox",
            "name": "modules",
            "title": "首页显示模块",
            "description": "可多选，取消勾选后对应模块不再显示",
            "value": ["clock"],
            "enum": [["clock", "时间显示"]]
        },
        {
            "type": "textarea",
            "name": "notice",
            "title": "首页公告",
            "description": "显示在搜索框下方，<code>支持HTML代码</code>，留空不显示",
            "value": ""
        },
        {
            "type": "text",
            "name": "security_filing",
            "title": "公安备案号",
            "description": "公安备案号，留空不显示",
            "value": "",
            "placeholder": "京公安网备xxxxxxxxxx号"
        }
    ]
}
```

#### `theme_config` 数组元素（配置项对象）参数说明

- type：表单类型，支持 [text：单行文本, textarea：多行文本, select：下拉菜单, checkbox：多选框, radio：单选, color：颜色]，**开关型建议使用 radio 替代 switch**
- name：表单参数的键，主题内需唯一
- title：配置项标题文字
- description：配置项提示文字
- value：默认值，支持字符串（含数字写字符串）、字符串数组（多选默认值）、或对象 `{"$conf": "站点配置键"}`（解析期代入对应站点配置的值，键不存在得空串）
- enum：枚举值，**二维字符串数组** `[["值", "显示文本"], ...]`，按声明顺序展示（select / checkbox / radio 可用）
- verify：表单验证方式，支持 [required：必填, url：网址, number：数字, email：邮箱等]

> 主题配置按主题独立存储（数据表键 `theme_config_{主题目录名}`），键名不会与其它主题冲突，
> 因此 `name` **无需加主题前缀**，保持语义清晰即可。

#### 读取配置

模板中使用：`{{ .ThemeCfg "参数名称" "默认值" }}`，**建议始终传入第二参默认值**。

```html
<!-- 多选项默认值以逗号分隔，可用 .HasModule 判断是否勾选 -->
{{ if .HasModule "clock" }}
    <!-- 显示时间模块 -->
{{ end }}
```

> 多选项全部取消勾选时浏览器不会提交该字段，**核心已做兼容处理**（缺失即视为空数组），
> 主题侧读取时仍需对空值做兜底。

### 内置模板函数（等价原版主题的 theme_* 契约函数）

核心已在渲染上下文中内置下列函数，主题模板（`index.html`）直接调用即可，无需自行引入：

| 函数 | 作用 |
| ---- | ---- |
| `{{ .CSS "css/style.css" }}` | 输出 `<link>` 标签，自动附加 `?v=主题版本` |
| `{{ .JS "js/script.js" }}` | 输出 `<script defer>` 标签，自动附加版本参数 |
| `{{ .ThemeVersion }}` | 读取 theme.ini 的版本号（去除非字母数字） |
| `{{ .Cfg "title" }}` | 读取站点配置项（自动转义）；输出 HTML 用 `{{ raw (.Cfg "copyright") }}` |
| `{{ .ThemeCfg "name" "默认值" }}` | 读取主题自定义配置（自动转义）；输出 HTML 用 `{{ raw .ThemeCfg }}` |
| `{{ range .Tags }}` | 导航菜单数组（字段：`.TagID` `.TagName` `.TagLink` `.TagTarget`） |
| `{{ range .Sou }}` | 搜索引擎数组（字段：`.Alias` `.Name` `.Hint` `.Icon` `.Color` `.Link`，link 已按 PC / 手机自动解析） |
| `{{ .ICP }}` | 输出 ICP 备案链接，留空则不输出 |
| `{{ .Gonganbei }}` | 输出公安备案链接（读主题配置 `gonganbei`，自动提取数字拼查询地址），留空则不输出 |
| `{{ .BackgroundImg }}` | 当前设备的背景图地址（PC / 手机自适应） |
| `{{ .Yan }}` | 随机一言内容 |
| `{{ .Lists (dict ...) }}` | 渲染分组与链接列表，片段键见下文 |

取数类成员（`.Tags`/`.Sou`）返回数组配合 `range` 使用，输出类函数（`.CSS`/`.JS`/`.ICP`/`.Gonganbei`）直接输出完整标签。
Go 模板默认对所有文本输出做 HTML 转义（等价原版 `theme_e`），需要原样输出官方 HTML 时使用 `raw` 函数。

#### 模板语法速查

| 原版写法 | Go 模板写法 |
| -------- | ----------- |
| `echo $conf['x']` | `{{ .Cfg "x" }}` |
| `theme_config('a', def)` | `{{ .ThemeCfg "a" "def" }}` |
| `while ($row = ...)` | `{{ range .Groups }}` / `{{ range .Sou }}` / `{{ range .Tags }}` |
| 主题内引入子模板 | `{{ template "list.html" . }}` |

### 系统配置（主题可用白名单）

主题应使用 `.Cfg` 中的下列配置项，其余键属于后台业务或程序内部，请勿依赖：

| 键 | 说明 | 注意 |
| ---- | ---- | ---- |
| `title` | 网站标题 | 输出已自动转义 |
| `keywords` / `description` | SEO 关键词 / 描述 | 输出已自动转义 |
| `logo` | 网站 LOGO / favicon | 输出已自动转义 |
| `background` / `wap_background` | PC / 手机背景图 | 用 `.BackgroundImg`，已做设备自适应 |
| `copyright` | 底部版权 | HTML 代码，用 `raw` 原样输出 |
| `wztj` | 自定义 footer（统计代码） | HTML/JS/CSS 代码，用 `raw` 原样输出在 `</body>` 前 |
| `icp` | ICP 备案号 | 留空不显示，直接用 `.ICP` 输出 |
| `yan` | 随机一言开关 | 字符串 `'true'`/`'false'`，判断用 `eq` |
| `tq` | 天气开关 | 核心不消费，由主题决定是否展示天气 |
| `template` / `version` / `cdnpublic` | 当前主题名 / 程序版本 / CDN 地址 | 系统项，CDN 地址用 `.Cdn` |

**不要依赖**：`apply`、`apply_gg`、`about`、`about_content`（分别由 `/apply`、`/about` 页面消费）、
`home-title`（已废弃，后台该位置现为「运行模式 `mode`」）、`admin_user`、`admin_pwd`、`wxplus` 等。

> 注意：`version` 的值形如 `v2.6.0`（带 `v` 前缀），不能直接用作缓存版本参数；
> 缓存版本请取 `theme.ini` 的 `theme_version`（模板函数 `.ThemeVersion`）。

### 数据输出

#### 分组和链接

##### 标签说明

| 分组标签 | 说明         | 链接标签 | 说明         |
| -------- | ------------ | -------- | ------------ |
| g1       | 分组开始标签 | l1       | 链接开始标签 |
| g2       | 分组内容     | l2       | 链接内容     |
| g3       | 分组结束标签 | l3       | 链接结束标签 |

##### 参数说明

| 分组参数         | 说明                 |
| ---------------- | -------------------- |
| **{group_id}**   | **分组ID(唯一标识)** |
| **{group_name}** | **分组名称**         |
| **{group_icon}** | **分组图标**         |

| 链接参数         | 说明                               |
| ---------------- | ---------------------------------- |
| {link_id}        | 链接ID(唯一标识)                   |
| {link_name}      | 链接名称(若设字体颜色则带HTML标签) |
| {link_name_text} | 链接名称(纯文本)                   |
| {link_url}       | 链接地址（已处理详情页模式）       |
| {link_icon}      | 链接图标（缺失时自动补默认图标）   |
| {link_desc}      | 链接描述                           |

##### 实例

```html
{{ .Lists (dict
    "g1" `<div class="link-group">`
    "g2" `<h2 class="group-title">{group_icon}<span>{group_name}</span></h2>`
    "g3" `</div>`
    "l1" `<a class="link-item" href="{link_url}" target="_blank" rel="nofollow noopener" title="{link_name_text}">`
    "l2" `<span class="link-icon">{link_icon}</span><span class="link-name">{link_name}</span>`
    "l3" `</a>`) }}
```

**HTML输出：**

```html
<div class="link-group">
    <h2 class="group-title"><svg ...></svg><span>常用导航</span></h2>
    <a class="link-item" href="https://www.lylme.com" target="_blank" rel="nofollow noopener" title="六零主页">
        <span class="link-icon"><svg ...></svg></span>
        <span class="link-name"><font color="#ff0000">六零主页</font></span>
    </a>
</div>
```

> `{link_name}` 可能携带 `<font>` 颜色标签（官方数据格式），因此**不转义**原样输出；
> 需要纯文本的场合（如 `title` 属性）请使用 `{link_name_text}`。

#### 导航菜单

描述：常用于显示在网站顶部的导航菜单

##### 参数

| 参数           | 说明                           |
| -------------- | ------------------------------ |
| `tag_id`       | 导航菜单唯一ID                 |
| `tag_name`     | 导航菜单名称                   |
| `tag_link`     | 导航菜单链接                   |
| `tag_target`   | 打开方式（1 新窗口，0 当前窗口） |

**推荐写法：**

```html
<ul>
    {{ range .Tags }}
    <li class="nav-item">
        <a class="nav-link" href="{{ .TagLink }}"{{ if eq .TagTarget 1 }} target="_blank"{{ end }}>
            {{ .TagName }}
        </a>
    </li>
    {{ end }}
</ul>
```

#### 搜索引擎

##### 参数

| 参数         | 说明                   | 备注                              |
| ------------ | ---------------------- | --------------------------------- |
| `sou_id`     | 搜索引擎ID             | 唯一ID                            |
| `sou_alias`  | 搜索引擎别名           | 英文别名（唯一）                  |
| `sou_name`   | 搜索引擎名称           | 如：百度                          |
| `sou_hint`   | 输入框提示文字         | 如：请输入搜索内容                |
| `sou_link`   | 搜索引擎接口地址       | 如：https://www.baidu.com/s?word= |
| `sou_waplink`| 搜索引擎手机端接口地址 | 如：https://m.baidu.com/s?word=   |
| `sou_icon`   | 搜索引擎图标           | 图标代码，svg 或 img（HTML 原样输出） |
| `sou_color`  | 搜索引擎字体颜色       | 如 #0c498c                        |
| `sou_st`     | 搜索引擎状态           | 是否启用                          |

**推荐写法（`link` 已按设备类型解析、未启用项已过滤，用 `raw` 原样输出图标）：**

```html
<div class="search-engines">
    {{ range $index, $sou := .Sou }}
    <label class="engine" style="--engine-color: {{ $sou.Color }};">
        <input type="radio" name="sou"
               value="{{ $sou.Link }}"
               data-alias="{{ $sou.Alias }}"
               data-hint="{{ $sou.Hint }}"
               {{ if eq $index 0 }}checked{{ end }}>
        {{ raw $sou.Icon }}
        <span class="engine-name">{{ $sou.Name }}</span>
    </label>
    {{ end }}
</div>
```

### 页面结构标准

#### HTML 骨架

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{ .Cfg "title" }}</title>
    <meta name="keywords" content="{{ .Cfg "keywords" }}">
    <meta name="description" content="{{ .Cfg "description" }}">
    <link rel="icon" href="{{ .Cfg "logo" }}">
    {{ .CSS "css/style.css" }}
    {{ .JS "js/script.js" }}
</head>
<body class="{{ if .IsMobile }}is-mobile{{ else }}is-pc{{ end }}">
    <header class="site-header"><!-- 导航 --></header>
    <main class="site-main"><!-- 搜索区 + 链接列表 --></main>
    <footer class="site-footer"><!-- 版权 / 备案 --></footer>

    {{ raw (.Cfg "wztj") }}
    <script src="{{ .Cdn }}/assets/js/icon.js"></script>
    <script src="{{ .Cdn }}/assets/js/svg.js"></script>
</body>
</html>
```

**必须保留的三处：**

1. `{{ raw (.Cfg "wztj") }}` —— 后台自定义的统计代码，放在 `</body>` 前
2. `<script src="{{ .Cdn }}/assets/js/icon.js"></script>` —— 图标雪碧图
3. `<script src="{{ .Cdn }}/assets/js/svg.js"></script>` —— 旧版图标雪碧图

> 数据库存储的全部分组 / 链接 / 搜索引擎图标都是 `<use xlink:href="#lyicon-*">` 形式，
> 其 SVG symbol 由 `assets/js/icon.js`和 `assets/js/svg.js`注入。**不加载该脚本，站点图标将全部空白。**

`<body>` 的 class 建议保留 `is-pc` / `is-mobile` 设备类，便于 CSS / JS 做设备差异处理；
主题自有的状态类（如 `bg-fixed`）追加在后面即可：`class="is-pc bg-fixed"`。

#### 搜索区 DOM 契约

统一结构，便于脚本与样式复用：

```html
<form class="search-form" id="search-form" action="#">
    <input type="text" class="search-input" id="search-input" placeholder="请输入搜索内容" autocomplete="off">
    <button type="submit" class="search-btn">搜索</button>
</form>
```

| 约定 | 说明 |
| ---- | ---- |
| `input[name="sou"]` | 引擎选择器，radio；`value` = 接口地址、`data-alias` = 引擎别名、`data-hint` = 输入提示 |
| 首个引擎 `checked` | 无本地记录时的默认选中项 |
| 提交由 JS 拦截 | 各引擎参数名不统一（`wd=` / `q=` / `word=`），原生表单 GET 无法适配，统一拼接 `接口地址 + encodeURIComponent(关键词)` |
| 选择记忆 | 写入 `localStorage['theme_sou']`（键名全站统一，换主题后用户选择依然保留） |

```js
// 切换：更新 placeholder + localStorage.setItem('theme_sou', data-alias)
// 载入：读取 localStorage，按 data-alias 找回对应 radio 并触发 change
// 提交：preventDefault 后 window.open(选中项.value + encodeURIComponent(关键词), '_blank')
```

### 兼容性要求

主题模板统一使用 **Go 模板（html/template）语法**，数据自动 HTML 转义，无需关心服务端语言版本；
原样输出官方 HTML（`link_name`、`sou_icon`、`copyright`、`wztj` 等）时必须使用 `raw` 函数显式声明。

**建议避免：**

| 场景 | 说明 |
| ---- | ---- |
| 非官方 HTML 不经 `raw` 直接拼接 | 破坏核心的自动转义安全模型 |
| JS 使用 `let/const/箭头函数/模板字符串` | 统一 ES5（`var` / `function`），兼容旧浏览器 |
| CSS 新特性无降级 | 如 `:has()` 需改为兄弟选择器 |

### 发布自检清单

- [ ] `index.html` / `theme.ini` 存在，theme.ini 前六字段齐全、JSON 合法，版本号三段式
- [ ] 目录名与 `theme_name` 对应
- [ ] 模板无未转义输出（官方 HTML 除外，且均经 `raw` 显式声明）
- [ ] 已保留 `wztj` 输出，并引入 `{{ .Cdn }}/assets/js/icon.js` 和 `{{ .Cdn }}/assets/js/svg.js`
- [ ] 未依赖白名单之外的系统配置键
- [ ] 多选项读取已做空值兜底
- [ ] 移动端 768px 断点下排版正常
- [ ] README 含配置项说明与演示地址
