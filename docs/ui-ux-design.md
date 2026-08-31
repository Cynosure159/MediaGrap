# MediaGrap UI/UX Design & Layout Specification

## Overview

MediaGrap is a web-first, self-hosted media metadata manager designed to deliver the power and information density of desktop tools (such as tinyMediaManager and MediaElch) within a modern, responsive, installable Progressive Web Application (PWA).

- **Stitch Design Project**: `MediaGrap - Modern Media Metadata Manager` (Project ID: `8752252538336937573`)
- **Design System Name**: `MediaGrap Core` (`assets/69a29fafc69d4000959529d9896942d9`)
- **Local Prototypes Archive**: `docs/stitch-prototypes/` (Includes all standalone HTML files and PNG mockups)
- **Visual Brand Assets**:
  - `web/public/assets/logo-icon.png` (无字版纯图形几何 Logo 标志)
  - `web/public/assets/logo-full.png` (完整带字版 Logo: 图形 + "MediaGrap")
  - `web/public/assets/logo-text.png` (纯文字版标识)
  - `web/public/assets/logo-icon-square.png` / `pwa-192.png` / `pwa-512.png` (PWA 规范应用图标)

---

## 1. 本地导出的 Stitch 原型清单 (Local Prototypes)

所有页面原型已导出至本地目录 [`docs/stitch-prototypes/`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/)：

| 页面 / 工坊 | 端类型 | 分辨率 | 本地 HTML 原型 | 本地截图预览 | 核心设计亮点 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **电影主工作台** | 桌面端 | 2560×2048 | [`dashboard.html`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/dashboard.html) | [`dashboard.png`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/screenshots/dashboard.png) | 3 栏工作台（64px Nav Rail + 320px 列表 + Bento Grid 详情） |
| **艺术图工坊** | 桌面端 | 2560×2048 | [`artwork-workshop.html`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/artwork-workshop.html) | [`artwork-workshop.png`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/screenshots/artwork-workshop.png) | Poster/Fanart/Logo/Disc/Banner 像素标注、候选画廊与裁切 |
| **NFO 源码与编辑器** | 桌面端 | 2560×2048 | [`nfo-editor.html`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/nfo-editor.html) | [`nfo-editor.png`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/screenshots/nfo-editor.png) | IDE 级深色代码编辑器、Kodi v19/v20 XML Schema 实时校验 |
| **演职员工坊** | 桌面端 | 2560×2048 | [`cast-workshop.html`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/cast-workshop.html) | [`cast-workshop.png`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/screenshots/cast-workshop.png) | 导演/编剧/演员头像卡片流、肖像抓取与拖拽排序 |
| **文件与重命名规划** | 桌面端 | 2560×2048 | [`files-rename-planner.html`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/files-rename-planner.html) | [`files-rename-planner.png`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/screenshots/files-rename-planner.png) | 命名模板引擎 `${title} (${year})`、伴随文件树、Dry-Run 冲突检测 |
| **电视剧多层级工作台** | 桌面端 | 2560×2048 | [`tv-shows-workspace.html`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/tv-shows-workspace.html) | [`tv-shows-workspace.png`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/screenshots/tv-shows-workspace.png) | 剧集总览 -> 季手风琴折叠卡片 -> 单集数据表格及 NFO 状态 |
| **设置中心控制台** | 桌面端 | 2560×2048 | [`settings-console.html`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/settings-console.html) | [`settings-console.png`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/screenshots/settings-console.png) | 媒体源挂载、TMDb 凭证、HTTP/SOCKS5 代理测速与偏好 |
| **刮削差异与安全写入** | 桌面端 | 2560×2048 | [`scraper-diff-preview.html`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/scraper-diff-preview.html) | [`scraper-diff-preview.png`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/screenshots/scraper-diff-preview.png) | 候选匹配卡片、逐字段 Diff 勾选合并、安全写入干跑计划 |
| **移动端详情与各工坊** | 移动端 | 780×2454 | [`mobile-detail-hub.html`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/mobile-detail-hub.html) | [`mobile-detail-hub.png`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/screenshots/mobile-detail-hub.png) | 无字 Logo 顶栏 + 吸顶工坊 Tabs（概览/艺术图/演职员/文件）+ 底部操作栏 |
| **移动端 PWA 媒体库** | 移动端 | 780×1768 | [`mobile-library.html`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/mobile-library.html) | [`mobile-library.png`](file:///Users/cy/Projects/01-home-lab/MediaGrap/docs/stitch-prototypes/screenshots/mobile-library.png) | 触控过滤胶囊、卡片流、后台扫描进度条、移动原生 4 键导航 |

---

## 2. 开发者强制设计与排版规范 (MediaGrap Core)

所有前端页面与组件必须严格遵循以下标准：

### 2.1 分栏与布局约束
1. **左侧活动导航栏 (Nav Rail)**: 固定 `64px` 宽度，深色底层 `--surface-container-lowest` (`#070d1f`)，无文字版纯图形 Logo。
2. **列表栏 (Catalog Panel)**: 固定 `320px` 宽度，包含即时搜索框、状态统计与过滤按钮、56px 高度卡片行。
3. **右侧主工作台 (Inspector Workspace)**: 弹性自适应宽度，顶部必须包含 `40px` 高度的工坊 Tabs 栏（Overview, Artwork, Cast, NFO Raw, File Audit）与操作按钮。
4. **移动端自适应 (Responsive Breakpoint: ≤700px)**:
   - 左侧 Rail 自动变为底部固定 4 键导航栏。
   - 媒体列表与详情检查器在移动端自动切换为分步式钻取（点击项目进入全屏详情，左上角提供返回按钮）。

### 2.2 色阶分层 (Tonal Layering Elevation)
- **`--surface-base` (`#0c1324`)**: L0 最深层底色（应用全局背景）。
- **`--surface-container-lowest` (`#070d1f`)**: 最低容器层（左侧 Rail、代码编辑器背景）。
- **`--surface-container-low` (`#151b2d`)**: 输入框、卡片内容块。
- **`--surface-container` (`#191f31`)**: 列表侧栏、分组卡片。
- **`--surface-container-high` (`#23293c`)**: Hover 悬浮表面、技术规格胶囊。
- **`--surface-container-highest` (`#2e3447`)**: 选中行高亮背景。

### 2.3 色彩与状态语义
- **`--primary` (`#c0c1ff`) / `--primary-container` (`#6366f1`)**: 核心交互色、激活边框（2px left border）、主按钮。
- **`--secondary` (`#4edea3`) / `--secondary-container` (`#00a572`)**: 翡翠绿，用于校验通过、保存写入 NFO 按钮、健康状态点。
- **`--tertiary` (`#ffb95f`)**: 琥珀黄，用于未刮削提醒、星级评分、进行中任务。
- **`--error` (`#ffb4ab`) / `--error-bright` (`#f43f5e`)**: 缺失项告警、错误横幅。

### 2.4 字体规范
- **界面文本**: `Inter`。
- **技术规格 / 分辨率 / XML / 文件路径**: `JetBrains Mono`。
