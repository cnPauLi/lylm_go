# 六零导航页（lylme spage）主题生成提示词

> 用途：把本文件作为提示词，投喂给任意 AI 编程工具（如 WorkBuddy / ChatGPT / Claude / 通义灵码等），
> 让它基于官方开发包规范，生成一个**可运行、合规、且与官方 `dev-theme` 风格截然不同**的六零导航页新主题。
> **主题风格由使用本提示词的人自由指定**（见下方「一、风格输入区」）。

---

## 〇、任务

你是六零导航页（lylme spage，开源导航网站程序）的主题开发助手。请根据**「一、风格输入区」**中用户给出的风格描述，生成一个完整、可直接启用、零框架依赖的新主题。

### 核心要求（务必贯穿始终）

1. **只保留「契约」代码，其余全部从零原创。**
   你必须原样照搬的只有「二、必须保留的契约」里的内容：数据接口函数、主题信息、配置项、链接列表 `lists()` 接口、搜索区 DOM 锚点、系统配置白名单、输出安全规则、模板/JS 兼容性硬约束。
   **除此之外的一切——`index.html` 的 HTML 结构、class 命名、整体布局、全部 CSS、动效——都要基于风格关键词大胆重构，禁止把官方 `dev-theme` 当模板去改色或微调。**

2. **禁止产出"dev-theme 换色版"。**
   官方 `dev-theme` 只是「契约样本 / 参考实现」，目的是告诉你接口怎么接，**不是给你套用的视觉模板**。它的 HTML 骨架、class 命名、CSS 变量体系、配色，一律不得复用（详见「三、视觉与布局：大胆原创」中的禁止清单）。

3. **大胆、具体、有辨识度。**
   优先做"一眼就能看出是什么风格"的设计，而不是安全的居中卡片网格。布局原型、配色体系、字体气质、点缀元素都要从关键词里长出来，而不是从 dev-theme 里改出来。

4. 输出下列 6 个文件（目录名用小写英文，例如 `moonlight`）。

---

## 一、风格输入区（★由使用本提示词的人填写★）

使用本提示词时，请把下面这段替换成你想要的风格。写得越具体、越有画面感，生成结果越贴合；**留白越多，AI 越要大胆发挥**（见第三节原型灵感）。

```
【主题名称（英文/拼音，作为目录名与 theme_name，小写）】：
【风格关键词】（如：极简、玻璃拟态、赛博朋克、复古、自然、暗黑、儿童、二次元、杂志、终端、像素…）：
【主色 / 配色方案】（可给具体色值，或只给感觉如"低饱和莫兰迪""高对比霓虹"）：
【整体氛围 / 情绪】（如：冷静专业、温暖治愈、科技未来、活泼可爱、疏离神秘）：
【布局偏好】（如：全屏居中 / 左侧固定栏 / 横向滑动 / 分屏双栏 / 瀑布流 / 杂志大字 / 终端风 / 便签墙 / 辐射圆 / 极简留白）：
【字体偏好】（如：圆润、纤细、等宽、衬线、手写、像素）：
【特殊视觉元素或动效】（如：毛玻璃、霓虹描边、悬浮放大、渐变流体背景、粒子、滚动视差、噪点质感、描边动画）：
【是否默认暗色 / 亮色 / 跟随系统】：
【其它想要的点】：
```

> 如果用户没有给具体值，则由你（AI）在合理范围内**自行设计一整套协调、有强风格倾向的方案**，并在 README 中说明你的默认设定与采用的布局原型。

---

## 二、必须保留的契约（★硬性，原样照搬，不要在里面"发挥"★）

> 这一节的内容是程序能跑起来的前提，**逐字保留即可**：不要改写函数逻辑，不要改 DOM 锚点，不要增删白名单键。
> 它们只负责"数据与接口"，与"视觉长什么样"完全无关——视觉在第三节另起炉灶。

### 2.1 程序环境与兼容性
- 目标程序：**六零导航页 v1.2.5 及以上**（Go 版，模板引擎为 Go html/template）。
- **模板统一使用 Go 模板语法**：数据输出默认自动 HTML 转义；需要原样输出官方 HTML 时必须显式使用 `raw` 函数。
- **JS 必须 ES5**：用 `var` / `function`，**禁止** `let` / `const` / 箭头函数 / 模板字符串（反引号）。
- 新版 CSS 特性要有降级方案（如 `:has()` 需改为兄弟选择器）。

### 2.2 设计原则（仅约束"工程形态"，不约束视觉）
- **单文件入口、零框架、轻量**。页面结构全部写在 `index.html` 内，用注释分区，不必拆文件。
- 静态资源唯一入口：`css/style.css`、`js/script.js`。
- 换肤通过 `:root` CSS 变量实现（**变量名你自己定义，不要照抄 dev-theme**）；后台可配项用 `theme.ini` 的 `theme_config` 暴露。

### 2.3 必须输出的文件与目录
```
你的主题目录名/                # 小写英文，与 theme_name 对应
├── index.html                # 入口模板（必须）
├── theme.ini                 # 主题信息 + 自定义配置定义（必须，JSON）
├── README.md                 # 主题说明（建议）
├── css/style.css             # 样式（必须，唯一入口）
└── js/script.js              # 脚本（必须，唯一入口）
```

### 2.4 theme.ini 规范（JSON，前六字段必填）
```json
{
    "author_name": "作者名",
    "author_link": "https://你的主页",
    "theme_name": "目录名(如 lylmelight)",
    "theme_version": "1.0.0",
    "theme_explain": "主题一句话说明",
    "theme_demo": "演示地址,如(https://spage.lylme.com/theme/lylmelight)",
    "requires": ">=1.2.5",
    "updated_at": "2026-01-01",
    "license": "MIT",
    "theme_course": "主题文档(https://spage.lylme.com/doc/lylmelight)"
}
```
- `theme_version` 三段式（`1.0.0`），改动静态资源后 +1 即可刷新缓存（`.CSS`/`.JS` 自动拼 `?v=`）。
- 顶层键 `theme_config` 定义后台配置表单（见 2.11）。
- 说明类字段不支持 HTML 标签。

### 2.5 内置模板函数（等价原版主题的 theme_* 契约函数，核心已内置，直接调用）
| 函数 | 作用 |
| ---- | ---- |
| `{{ .ThemeVersion }}` | 读取 theme.ini 的版本号（去除非字母数字，`1.0.0` → `100`） |
| `{{ .CSS "css/style.css" }}` | 输出 `<link>` 标签，自动附加 `?v=版本号` |
| `{{ .JS "js/script.js" }}` | 输出 `<script defer>` 标签，自动附加版本号 |
| `{{ .Cfg "键名" }}` | 读取站点配置（自动 HTML 转义，等价 `theme_e`） |
| `{{ raw (.Cfg "copyright") }}` | 官方 HTML 原样输出（等价跳过 `theme_e`） |
| `{{ .ThemeCfg "name" "默认值" }}` | 读取主题自定义配置（自动转义） |
| `{{ range .Tags }}` | 导航菜单数组：`.TagID` `.TagName` `.TagLink` `.TagTarget` |
| `{{ range .Sou }}` | 搜索引擎数组：`.Alias` `.Name` `.Hint` `.Icon` `.Color` `.Link`（`link` 已按 PC / 手机自动解析） |
| `{{ .ICP }}` | 输出 ICP 备案链接（读 `icp`），留空不输出 |
| `{{ .Gonganbei }}` | 输出公安备案链接（读主题配置 `gonganbei`，自动提取数字拼查询地址），留空不输出 |
| `{{ .BackgroundImg }}` | 当前设备背景图地址（PC / 手机自适应） |
| `{{ .Yan }}` | 随机一言内容 |
| `{{ if .IsMobile }}` | 是否为手机访问 |
| `{{ .Lists (dict ...) }}` | 渲染分组与链接列表（见 2.7） |

> 核心不加载主题内自定义代码文件；如需扩展逻辑，请在模板内用 Go 模板语法实现，或依赖 JS 完成。
> 文本输出由模板引擎自动转义（等价 `theme_e`），**例外**：`{link_name}`、`sou_icon` 等官方 HTML 用 `raw` 原样输出。

### 2.6 index.html 入口契约（只管"接口与必引入项"，不管视觉骨架）
> 下面给出**最小必要骨架**：必引入的 css/js、统计代码、图标雪碧图必须原样保留。
> 中间的导航 / 搜索 / 列表区域，**结构、class、排布全部由你原创**（见第三节），不要在 dev-theme 的 `.site-header` / `.site-main` / `.links` 骨架上改。

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
    <!-- 导航：range .Tags —— 结构与 class 自定 -->
    <!-- 模块：时钟(if .HasModule "clock")、随机一言(eq (.Cfg "yan") "true" 时 .Yan) -->
    <!-- 注意：.Yan 返回字符串，需 {{ .Yan }} 输出；.BackgroundImg 返回背景图地址，需接收后使用 -->
    <!-- 搜索区：见 2.8 DOM 锚点契约 -->
    <!-- 链接列表：见 2.7 lists() 接口 -->
    <footer>
        {{ raw (.Cfg "copyright") }}
        {{ .ICP }}
        {{ .Gonganbei }}
    </footer>

    {{ raw (.Cfg "wztj") }}   <!-- 必须：统计代码，原样输出在最底部 -->
    <script src="{{ .Cdn }}/assets/js/icon.js"></script>  <!-- 必须：图标雪碧图，删除后图标全空白 -->
    <script src="{{ .Cdn }}/assets/js/svg.js"></script>  <!-- 必须：旧版图标雪碧图，删除后图标全空白 -->
</body>
</html>
```

### 2.7 链接列表接口 `lists($html)` → `{{ .Lists (dict ...) }}`
- 仅以下键：`g1`分组开始 / `g2`分组内容 / `g3`分组结束 / `l1`链接开始 / `l2`链接内容 / `l3`链接结束。
- 可用占位符：`{group_id}` `{group_name}` `{group_icon}`；`{link_id}` `{link_name}`(可能含`<font>`色标，原样输出不转义) `{link_name_text}`(纯文本) `{link_url}` `{link_icon}`(缺失自动补默认) `{link_desc}`。
- **图标是官方注入元素（必读）**：`{group_icon}` / `{link_icon}` 以及 §2.8 的 `sou_icon` 都由页面底部 `icon.js`/`svg.js` 注入为**行内 `<svg>` 或 `<img>`**（不是 `<i class="iconfont">` 字体图标），默认无尺寸约束会渲染得很大。**必须**在你的 `style.css` 里显式限制尺寸，且要**同时覆盖 `svg` 和 `img`**，否则选择器命中不到、改了也没效果。详见 §3.6。
- **注意**：下面 `l1/l2/g2` 里的 class 名（如 `link-item` `link-icon`）只是占位示例，**你必须替换成自己的原创命名**，并在自己的 `style.css` 里写对应的原创样式。不要原样照抄这一套 class。

```html
{{ .Lists (dict
    "g1" `<YOUR_OWN_GROUP_WRAPPER>`
    "g2" `<YOUR_OWN_GROUP_TITLE>{group_icon}<span>{group_name}</span></YOUR_OWN_GROUP_TITLE>`
    "g3" `</YOUR_OWN_GROUP_WRAPPER>`
    "l1" `<a class="YOUR_OWN_LINK" href="{link_url}" target="_blank" rel="nofollow noopener" title="{link_name_text}">`
    "l2" `<span class="YOUR_OWN_ICON">{link_icon}</span><span class="YOUR_OWN_NAME">{link_name}</span>`
    "l3" `</a>`) }}
```

### 2.8 搜索区 DOM 锚点契约（JS 依赖，只保这些"锚点"，外观与排布随意）
> JS 只读下面这些 **id / name / data-属性**，class 名、包裹结构、视觉全由你定。
> 务必保留：`form` 的 `id="search-form"`、`input` 的 `id="search-input"`、`radio` 的 `name="sou"`、`value=接口地址`、`data-alias=别名`、`data-hint=提示`；首个 `checked`。

```html
<form id="search-form" action="#">
    <input type="text" id="search-input" placeholder="请输入搜索内容" autocomplete="off">
    <button type="submit">搜索</button>
</form>
<div class="YOUR_OWN_ENGINE_LIST">
    {{ range $index, $sou := .Sou }}
  <!-- 每个引擎：radio name="sou"，value=接口地址，data-alias=别名，data-hint=提示；首个 checked -->
  <label style="--engine-color: {{ $sou.Color }};">
    <input type="radio" name="sou" value="{{ $sou.Link }}"
           data-alias="{{ $sou.Alias }}" data-hint="{{ $sou.Hint }}"
           {{ if eq $index 0 }}checked{{ end }}>
    {{ raw $sou.Icon }}
    <span>{{ $sou.Name }}</span>
  </label>
    {{ end }}
</div>
```
- 提交由 JS 拦截：拼接 `接口地址 + encodeURIComponent(关键词)` 新窗口打开（各引擎参数名不统一）。
- 选择记忆写入 `localStorage['theme_sou']`（键全站统一，换主题保留）。
- JS 用 ES5 实现：切换引擎更新 placeholder + 记忆；载入时恢复；提交时拼接跳转；时间显示模块需做「节点不存在则跳过」的兼容。

### 2.9 系统配置白名单（只可用这些 `.Cfg` 键）
`title` `keywords` `description` `logo` `background`/`wap_background`(用 `.BackgroundImg` 自适应) `copyright`(HTML，`raw` 原样) `wztj`(HTML/JS，`raw` 原样) `icp`(用 `.ICP`) `yan`(判断用 `eq ... "true"`) `tq`(天气开关，主题自决) `template`/`version`/`cdnpublic`(系统项，CDN 地址用 `.Cdn`)。
**不要依赖**：`apply` `about` `home-title` `admin_user` `admin_pwd` `wxplus` 等。
> 注意：`version` 形如 `v2.6.0`（带 v），**不能**用作缓存版本；缓存版本取 `theme.ini` 的 `theme_version`（`.ThemeVersion`）。

### 2.10 安全与输出规则
- **所有文本输出由模板引擎自动转义（等价 `theme_e`）**；例外：`{link_name}`、`sou_icon`（官方 HTML，用 `raw` 原样输出）。
- 多选项（checkbox）读取做空值兜底（全不选时核心给空数组，主题侧仍需判断空值）。
- 导航/引擎/图标等循环输出由模板引擎自动转义。

### 2.11 theme.ini 的 `theme_config` 配置表单
- `theme_config` 为 JSON 数组，每个元素含：`type`(text/textarea/select/checkbox/radio/color，开关建议用 radio 替代 switch)、`name`(主题内唯一，无需前缀)、`title`、`description`、`value`、`enum`(二维字符串数组 `[["值","显示文本"],...]`)、`verify`(required/url/number/email)。
- `value` 支持：字符串（数字也写成字符串）、字符串数组（多选默认值）、或对象 `{"$conf": "站点配置键"}`（解析期代入站点配置的值，键不存在得空串）。
- **必须保留**通用配置（可改默认值以贴合风格）：主色 `color`(color)、列表列数 `link_cols`(select 3~6)、首页模块 `modules`(checkbox，含 clock)、首页公告 `notice`(textarea，支持HTML)、公安备案号 `gonganbei`(text)。
- 可按风格增删配置（如：圆角大小、卡片透明度、是否毛玻璃、强调色、背景模糊度、字体族等），增强可玩性。

```json
{
    "theme_config": [
        {"type": "color", "name": "color", "title": "主题主色", "description": "用于搜索按钮、链接悬停等主色调", "value": "#2f6fed"},
        {"type": "select", "name": "link_cols", "title": "列表列数", "description": "每行显示的链接数量", "value": "4",
         "enum": [["3", "3 列"], ["4", "4 列"], ["5", "5 列"], ["6", "6 列"]]},
        {"type": "checkbox", "name": "modules", "title": "首页显示模块", "description": "可多选", "value": ["clock"],
         "enum": [["clock", "时间显示"]]},
        {"type": "textarea", "name": "notice", "title": "首页公告", "description": "显示在搜索框下方，支持HTML代码", "value": ""},
        {"type": "text", "name": "gonganbei", "title": "公安备案号", "description": "留空不显示", "value": "",
         "placeholder": "京公安网备xxxxxxxxxx号"}
    ]
}
```

---

## 三、视觉与布局：大胆原创（★本节是本次生成的重点，禁止照搬 dev-theme★）

> 这一节决定主题"像不像 dev-theme"。请把它当成硬约束来执行。

### 3.1 禁止清单（出现任意一条即视为不合格）
**禁止复用 dev-theme 的下列 class 名**（即使你改了颜色也不行）：
`site-header` `site-logo` `site-nav` `site-main` `clock` `clock-time` `clock-date` `yan` `search` `search-form` `search-input` `search-btn` `search-engines` `engine` `engine-name` `notice` `links` `link-group` `group-title` `link-item` `link-icon` `link-name` `site-footer` `icp` `gonganbei`

**禁止照搬 dev-theme 的 CSS 变量体系**：不要原样使用 `--theme-color / --text-color / --text-muted / --card-bg / --card-radius / --card-shadow / --gap / --link-cols` 这一套命名与数值。你应当**定义自己的一套语义变量**（命名体现你的风格，例如 `--ink` `--paper` `--accent` `--glow` `--depth` 等，随风格自定）。

**禁止套用 dev-theme 的布局骨架**：即"顶部一条横向导航 → 中间 max-width:1200px 居中容器 → 搜索框居中 → 下方卡片网格（CSS grid + repeat(var(--link-cols),1fr)）"这种组合。这是最容易被复制的默认版式，务必避开。

### 3.2 布局原型灵感（至少从里面挑一个，或融合，或自己发明；不要退回默认版式）
- **全屏英雄居中**：时钟/一言作视觉主体占满首屏，搜索框悬浮中央，链接区在下方可滚。
- **左侧固定侧栏**：导航与品牌固定在左侧竖向栏，右侧为可滚动的内容画布（移动端折叠为顶部条）。
- **横向滑动分类**：每个分组是一行可横向滑动的卡片，整页纵向排布"轨道"。
- **分屏双栏**：左半是信息/时钟/搜索的"控制台"，右半为链接瀑布；或大图背景 + 前景浮层。
- **不规则网格 / 瀑布流**：卡片尺寸、跨列、错位随内容变化，拒绝整齐方格。
- **杂志 / 编辑风**：超大衬线标题、分栏排版、首字下沉、留白即设计。
- **终端 / CLI 风**：等宽字体、`$`、`>` 提示符、块光标闪烁、纯文本质感。
- **便签 / 卡片墙**：链接像便利贴、拍立得、笔记本页随意散布（用旋转/偏移制造手感）。
- **辐射 / 同心圆**：链接以中心为原点环形或射线排布，悬停展开。
- **极简留白**：巨量负空间、单一焦点、近乎无边框，靠排版与字重建立层级。
- **材质拟态**：新拟物（soft UI）、黏土（claymorphism）、玻璃（glassmorphism）、噪点纸纹、金属拉丝等其中一种明确质感。
- **暗黑霓虹 / 赛博朋克**：深色底 + 高饱和描边/辉光/扫描线/网格地平线。
- **像素 / 复古**：像素质感边框、CRT 扫描线、8-bit 配色、复古操作系统窗口。
- **日系 / 自然**：和纸底纹、低饱和、手写感、留白与季节意象。

> 鼓励：把上述多个原型融合，或提出文档没列的原型。目标是**风格一眼可辨**。

### 3.3 配色与视觉语言
- 配色必须**从关键词推导**，形成有逻辑的色板（主色 / 背景 / 文字 / 强调 / 阴影或辉光），而不是只改一个 `--theme-color`。
- 暗色 / 亮色 / 跟随系统由风格决定；如做双主题，用自己命名的变量组切换。
- 字体气质要与风格一致（圆润→儿童/治愈；纤细无衬线→极简/科技；衬线→杂志/复古；等宽→终端；像素字体→像素风）。优先用系统字体栈与 `.BackgroundImg` 背景图，**不要引入外部字体 CDN 依赖**（保持零框架、可离线）。

### 3.4 CSS 必须原创
- 自己写 `:root` 变量体系、自己命名 class、自己写每一处样式。**不要从 dev-theme 的 style.css 复制任何规则再调参数。**
- 允许使用现代 CSS（`grid` / `flex` / `clamp()` / `min()` / `gradient` / `transform` / `filter` / `mask`），但需给出老旧浏览器的可用降级（如 `:has()` 改兄弟选择器）。
- 动效要有风格：悬停、入场、加载、焦点态都服务于整体气质（如赛博用辉光脉冲，便签用轻微旋转回正，玻璃用模糊与位移）。

### 3.5 移动端（保留 768px 断点，但版式自定）
- 必须有 `@media (max-width: 768px)` 适配，保证手机可用、不溢出、可点。
- 不一定"降为 2 列"——如果你的原型是侧栏，就折叠成顶部导航；是横向轨道，就保持滑动；按你的布局自然收敛即可。

---

### 3.6 图标尺寸提醒（AI 必读）

> 新手最容易被卡住的点：改完 CSS 图标还是巨大、强制刷新也没用，根因是选择器没命中真实图标元素。请务必注意：

- 系统的搜索图标 `sou_icon`、分组图标 `{group_icon}`、链接图标 `{link_icon}` 都由页面底部 `icon.js` / `svg.js` 注入为**行内 `<svg>` 或 `<img>` 元素**（不是 `<i class="iconfont">` 字体图标），默认无尺寸约束，会按原始尺寸渲染得很大。
- **必须**在 `style.css` 里对承载图标的自定义 class 显式限制尺寸；**且要同时覆盖 `svg` 和 `img` 两种选择器**（只写 `i` 或只写 `img` 都会漏掉另一种，导致完全不生效）。若遇到字体图标再补 `i` 兜底。
- **具体尺寸由你（AI）按风格自行决定**，不必与官方一致。

### 3.7 搜索引擎交互形态
请明确主题中搜索引擎选择组件的交互方式，从以下方案中选择一种或提出自定义方案：
- **A. 平铺直显**：在搜索框下方/上方直接横向/网格展示所有启用的搜索引擎图标或文字，点击即切换，无下拉动作。
- **B. 下拉列表**：点击搜索框旁的引擎 Logo/名称，弹出下拉菜单供用户选择（默认方案）。
- **C. 折叠面板**：默认只显示一个默认引擎，点击后展开一个全屏/半屏的引擎面板，支持分组和搜索。
  
### 3.8 搜索引擎联想词与站内搜索列表（建议增强）

> 这两项都是**建议增强**（强烈建议实现，而非可选点缀），系统核心**并未提供**联想词接口或站内搜索端点：
> - 搜索引擎表只存了 `sou_link` / `sou_waplink`（搜索跳转地址），**没有**联想词 API 字段；
> - 站内链接数据可通过模板函数 `{{ .ListJSON }}` 获取结构化 JSON（字段：`title` `url` `icon` 等），主题 JS 自行检索。
> 二者与 §3.7 的 A/B/C **引擎选择形态正交**——无论选哪种形态，都可叠加下面两类增强。实现时同样遵循 §2 的契约与 §3.1 的禁止清单（下拉浮层、结果列表一律用你自己的 class，禁止复用 `search-*` 等 dev-theme 命名）。

#### 3.8.1 搜索引擎联想词（Suggestions / 自动补全）

**用途**：用户在 `#search-input` 输入时，下拉展示该引擎的联想补全词，点击即填充并提交，提升输入效率。

**默认联想源（优先级）**：**百度联想词优先度最高，Google 最低**。除非用户在「一、风格输入区」明确指定其它联想源，否则**默认只对接一种——百度（JSONP）即可**，不必同时接多个 API。优先级参考：百度 > Bing > Google。

**数据来源（外部联想 API，需主题自行对接）**：
- **百度（JSONP，默认首选，优先级最高，CORS 友好，国内可用）**：
  `https://suggestion.baidu.com/su?wd=关键词&cb=全局回调名`
  返回 `cb({q:"关键词",p:false,s:["词1","词2",...]})`，补全词在 `s` 数组。
- **Bing（JSON，XHR，优先级次之）**：
  `https://api.bing.com/osjson.aspx?query=关键词`
  返回 `["关键词",["词1","词2",...]]`，补全词在索引 `1` 的数组（可能受 CORS 限制，失败需静默降级）。
- **Google（JSON，XHR，优先级最低）**：
  `https://www.google.com/complete/search?client=firefox&q=关键词`
  返回 `["关键词",["词1",...]]`（仅当用户在风格输入区**明确指定**时才启用）。

**与引擎联动**：默认情况下联想词**与所选搜索引擎无关**，统一调用百度联想（仅作输入补全，最终提交仍按 §2.8 的引擎 `value` 跳转）。仅当用户在风格输入区**明确指定**"某引擎使用其自有联想源"时，才在 `js/script.js` 里维护一个按 `data-alias` 映射的表（例如 `var SUG_API = { baidu: 'https://suggestion.baidu.com/su?wd=%s&cb=%s', bing: 'https://api.bing.com/osjson.aspx?query=%s' };`），当前引擎 `data-alias` 命中时启用对应源，未命中则回退到百度。也可在 `theme.ini` 的 `theme_config` 增加「联想词开关 + 自定义 API 模板」配置项增强可玩性（变量命名贴合你的主题）。

**ES5 实现要点（全部原生，无 fetch/let/const/箭头/Promise）**：
1. **输入防抖**：监听 `input` 事件，约 200–300ms 后才请求，避免高频打接口。
2. **JSONP 注入**：默认采用百度 JSONP 接口，动态 `document.createElement('script')` 注入 `src`（回调名用唯一全局函数，请求完成后移除该 `<script>`）；仅当用户明确指定 Bing/Google 时，才改用 `XMLHttpRequest` + `overrideMimeType('application/json')` 读取 JSON。
3. **渲染浮层**：在搜索框下方插入一个**自定义容器**（你的 class，如 `MY_SUG_BOX`），逐条渲染候选词；浮层只展示、不干扰主搜索。
4. **键盘交互**：支持 `↑`/`↓` 移动高亮、`Enter` 填充并提交、`Esc` 关闭；点击候选词同样填充并提交。
5. **失败降级**：请求超时 / 解析失败 / 网络异常时直接隐藏浮层，**绝不**阻塞或报错影响主搜索流程。
6. **跟随引擎切换**：引擎 `change` 时清空已有联想并刷新（见 §2.8 的 `data-alias`）。

> ⚠️ **JSONP 回调生命周期（高频踩坑，务必照做）**：回调**绝对不能**用"发新请求时删除上一个全局回调"的方式实现。若在新请求发起时 `delete window[cbName]`（或置空），那个**仍可能在途**的旧 `<script>` 响应（HTTP 缓存 / 慢网络）稍后执行时会裸调用 `_xxx_N()`，但全局函数已被删除 → `Uncaught ReferenceError: _xxx_N is not defined`，联想直接瘫痪。同理，**超时 / `onerror` 的兜底 `finish([])` 也绝不能删回调**——否则百度响应比 1600ms 超时才到达时，回调已被超时删掉，真实响应一来照样 `_xxx_N is not defined`。正确做法三条：
> 1. `cleanup` 只 `removeChild` 旧的 `<script>` 节点（取消请求），**绝不**碰 `window[cbName]`；
> 2. **删除回调只发生在"真实响应触发的那个 JSONP 回调函数内部"**（在调用 `finish` 之前 `delete window[cbName]`）；超时 / `onerror` 的兜底 `finish([])` **不要**删回调，它只负责渲染空结果并隐藏浮层；
> 3. 用递增 `reqId` 守卫：过期响应即便误触发也直接 `return` 忽略，绝不渲染旧数据。

**移动端**：浮层在 `≤768px` 下全宽、可独立滚动、不得遮挡搜索按钮与功能区域；候选词点击区域不小于 44px 便于触控。

#### 3.8.2 站内搜索列表（搜索本站导航自身的链接）

**用途**：在站内检索导航页**自己收录的链接**（而非联网搜索），输入时实时过滤，或提交后在浮层/结果区列出匹配项。

**方案 A — 客户端过滤（推荐，零后端）**
- 若主题已用链接瓦片渲染列表（§2.7 的 `l1`），**直接遍历瓦片节点即可**，读取其子节点里的名称、描述、图标 HTML 以及 `getAttribute('href')`（链接），**无需在 `l1/l2` 里额外补 `data-*` 属性**。
- 也可在模板中输出 `{{ .ListJSON }}`（结构化链接 JSON，字段含 `title` `url` `icon`），供 JS 全量检索。
- 用户输入时，对 `name` / `desc` / `url` 三者做 `toLowerCase()` + `indexOf` 包含匹配，把命中的链接渲染进一个**自定义结果浮层 / 列表**（你的 class）。
- 优点：纯前端、离线可用、无数据库压力；缺点：只能匹配**已渲染**的链接（分页 / 懒加载场景需配合数据兜底）。

> 💡 **推荐：把「站内结果」与「联想词」合并进同一下拉**：二者数据源不同（站内来自本地 DOM，联想来自外部 API），但都是"输入时的补全建议"。最自然的做法是主搜索框下拉同时包含两组，并用分组头区分——例如先排「站内搜索」命中（最多 5 条，点击在新标签直达该链接），再排「搜索建议」联想词（最多 10 条，点击回填并按当前引擎 `value` 提交）。合并时 `items` 用对象数组区分类型（`{type:'site',...}` / `{type:'suggest',text}`），渲染时按类型插入分组头，**避免**把它们做成两套互相打架的浮层。

**结果展示约束**：站内结果列表的配色、圆角、卡片/浮层样式必须与主题视觉一致；图标仍须按 §3.6 同时约束 `svg` 与 `img` 尺寸；动态渲染文本时用 `textContent` 赋值避免 XSS。

#### 3.8.3 统一设计约束（收口）

- **风格一致**：联想浮层、站内结果区在配色 / 圆角 / 动效上必须与主题整体视觉呼应，不得出现"另一个风格的控件"。
- **与 §3.7 形态协同**：
  - 若选 **A（平铺直显）**：注意 `≤768px` 下引擎条的横向滚动或换行适配，联想浮层宽度与之对齐。
  - 若选 **C（折叠面板）**：面板弹出动效在 `js/script.js` 内实现，且**不能影响**功能区域布局（用 `position: fixed/absolute` 浮层，而非挤压文档流）。
- **无依赖**：两项增强都不引入框架或外部 CDN（保持零框架、可离线）；JS 必须 ES5。
- **性能与降级**：联想词必须防抖、请求失败静默降级（隐藏浮层即可）；列表渲染量过大时做截断（如最多 20 条）并支持滚动。
- **安全**：联想 API 的回调名用白名单/随机化避免 XSS；动态插入的站内结果用 `textContent` / 受控的 DOM 构建渲染。

#### 3.8.4 AI 实现避坑清单（来自真实踩坑）

- **`render()` 绝不能先清空数据数组**：渲染函数只应清空 DOM（`boxEl.innerHTML = ''` + 重置 `cursor`），**绝不能**调用会把 `items` 重置为 `[]` 的 `clearBox()` 之后再去遍历 `items`——否则数组已被清空、循环一次不执行，列表永远为空且浮层保持 `hidden`。正确姿态：数据由请求回调在调用 `render()` **之前**赋值，`render()` 只负责把当前 `items` 画出来。
- **键盘高亮 / 滚动用 `data-idx` 定位，别用 DOM 顺序索引**：当下拉里混有非 `<div>` 行、或插入了分组头（不带 `lg-sug-item` 类）时，`getElementsByTagName('div')` 的下标与 `items[]` 下标会错位。正确：每行带 `data-idx`（= 其在 `items` 的下标），`setCursor` / `scrollCursor` 用 `$$('.lg-sug-item', boxEl)` 仅取有效行，并按 `getAttribute('data-idx')` 匹配，与 `items[cursor]` 一致。
- **JSONP 回调不要提前删除**：见 §3.8.1 的 ⚠️——只 `removeChild` 旧 `<script>` 节点，回调在自身触发后才自我清除。
- **合并下拉时 `items` 用类型化对象数组**：`{type:'site',url,name,desc,glyph,hue}` 与 `{type:'suggest',text}` 混排，渲染按类型插分组头（"站内搜索" / "搜索建议"）；分组头**不要**带 `lg-sug-item` 类，否则会被键盘导航计入导致错位。`pick()` 按 `type` 分流：站点行 `window.open(url,'_blank')`，联想行回填并走当前引擎。
- **站点结果行用 `<div>` 而非 `<a>`**：站点行在 `pick()` 里用 `window.open(url,'_blank')` 打开即可；若用 `<a href>` 又 `preventDefault`，容易触发"原生跳转 + `window.open`"双重打开。
- **`window.open` 必须在用户手势内调用**：放在 `mousedown` / `click` 回调里（联想词的 `mousedown` 已 `preventDefault` 防输入框失焦），不要放在异步回调里，否则被浏览器当作弹窗拦截。

## 四、生成步骤（AI 执行顺序）

1. 解析「一、风格输入区」→ 选定语义布局原型（来自第三节），形成一整套**原创**视觉规范（配色、间距、圆角/质感、字体、动效）。
2. 确定目录名（小写英文）与 `theme_name`，写 `theme.ini`（用 2.4）。
3. 在 `theme.ini` 写 `theme_config` 配置表单：保留 2.11 通用项，按风格增删（变量命名贴合你的主题）。
4. **写 `index.html`：抛弃 dev-theme 骨架，按你选的布局原型原创 DOM 结构与 class；只保留 2.6 的入口契约与必引入项、2.7/2.8 的接口锚点。**
5. **写 `css/style.css`：从零原创，定义自己的变量体系与全部样式，实现你选的布局原型与风格视觉；含 768px 移动端适配（版式自定）。**
6. 写 `js/script.js`：ES5，保留搜索切换/记忆/提交 + 时钟（DOM 锚点同 2.8），可叠加风格化动效；做"节点不存在则跳过"兼容。
7. 写 `README.md`：含风格说明、**采用的布局原型**、配置项说明、演示地址、适用版本、自检清单。
8. 自查（见第五节），特别核对「是否仍长得像 dev-theme」。

---

## 五、发布自检清单（生成后逐条核对）

- [ ] `index.html` / `theme.ini` 存在；theme.ini 前六字段齐全、JSON 合法、版本三段式；`theme_config` 可被解析。
- [ ] 目录名与 `theme_name` 对应。
- [ ] 模板文本输出经自动转义，官方 HTML（`link_name`/`sou_icon`/`copyright`/`wztj`）经 `raw` 原样输出。
- [ ] 保留 `wztj` 输出与 `icon.js`/`svg.js` 引入。
- [ ] 未依赖白名单外配置键。
- [ ] 多选项读取已做空值兜底。
- [ ] JS 为 ES5，无 `let/const/箭头/模板字符串`；搜索 DOM 锚点（id/form/input/name="sou"/data-alias/data-hint）完整。
- [ ] `lists()` 接口结构完整；占位符使用正确。
- [ ] 768px 断点下排版正常、不溢出。
- [ ] **【创新校验】未使用第三节 3.1 禁止清单里的任何 dev-theme class 名。**
- [ ] **【创新校验】CSS 变量体系为自创，未原样照搬 `--theme-color/--card-bg/--link-cols` 等。**
- [ ] **【创新校验】布局不是"顶部导航条 + 居中 1200px 容器 + 卡片网格"的 dev-theme 默认版式，而是明确采用了第三节的某一种（或融合的）原创原型。**
- [ ] **【图标校验】已对承载 `sou_icon` / `{group_icon}` / `{link_icon}` 的自定义 class 同时写 `svg` 与 `img` 尺寸规则（见 §3.6），避免图标过大、且改动后 `theme_version` 已 +1。**(详见 §3.6)
- [ ] README 含配置说明、演示地址、**采用的布局原型**。

---

## 六、输出格式要求

请一次性给出全部 6 个文件的**完整代码**（不要省略、不要写「此处省略」），并在开头用 3-5 句话概述你基于用户输入所做的风格设定、采用的**布局原型**、以及默认补全项。若用户输入的风格信息不足，明确说明你采用的默认设定与选型理由。
